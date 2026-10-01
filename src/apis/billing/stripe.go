package billing

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/stripe/stripe-go/v86"
	"github.com/stripe/stripe-go/v86/webhook"
)

type PriceInfo struct {
	AmountCents int64
	Currency    string
	Name        string
}

type CheckoutParams struct {
	CustomerID string
	UserID     string
	CourseID   int32
	SuccessURL string
	CancelURL  string
}

type CheckoutResult struct {
	URL string
}

type WebhookEvent struct {
	Type            string
	SessionID       string
	UserID          string
	CourseID        int32
	PaymentIntentID string
	AmountCents     int32
	Source          string
}

// CompletedCheckout is a paid Checkout Session, read back so the return URL
// can grant access without waiting on the webhook.
type CompletedCheckout struct {
	SessionID       string
	UserID          string
	CourseID        int32
	PaymentIntentID string
	AmountCents     int32
	Source          string
	Paid            bool
}

type Gateway interface {
	CreateCustomer(ctx context.Context, email, name, userID string) (string, error)
	CreateCheckoutSession(ctx context.Context, p CheckoutParams) (*CheckoutResult, error)
	GetPrice(ctx context.Context) (*PriceInfo, error)
	RetrieveCheckout(ctx context.Context, sessionID string) (*CompletedCheckout, error)
	ParseWebhook(payload []byte, sigHeader string) (*WebhookEvent, error)
}

type stripeGateway struct {
	client        *stripe.Client
	priceID       string
	webhookSecret string
}

func NewStripeGatewayFromEnv() (Gateway, error) {
	key := os.Getenv("STRIPE_SECRET_KEY")
	priceID := os.Getenv("STRIPE_PRICE_ACCESS")
	if key == "" || priceID == "" {
		return nil, fmt.Errorf("STRIPE_SECRET_KEY and STRIPE_PRICE_ACCESS are required")
	}
	g := &stripeGateway{
		client:        stripe.NewClient(key),
		priceID:       priceID,
		webhookSecret: os.Getenv("STRIPE_WEBHOOK_SECRET"),
	}
	if err := g.verifyWebhookSecret(); err != nil {
		return nil, err
	}
	return g, nil
}

// verifyWebhookSecret signs a checkout.session.completed event with the configured
// secret and requires the parser to recover the payer and course. Checkout stays
// disabled when this cannot succeed.
func (g *stripeGateway) verifyWebhookSecret() error {
	if g.webhookSecret == "" {
		return fmt.Errorf("STRIPE_WEBHOOK_SECRET is required")
	}
	payload := []byte(`{
		"id": "evt_selftest",
		"object": "event",
		"type": "checkout.session.completed",
		"api_version": "2026-07-29.dahlia",
		"data": {"object": {
			"id": "cs_selftest",
			"object": "checkout.session",
			"client_reference_id": "user_selftest",
			"amount_total": 1,
			"payment_intent": "pi_selftest",
			"metadata": {"clerk_user_id": "user_selftest", "course_id": "9"}
		}}
	}`)
	signed := webhook.GenerateTestSignedPayload(&webhook.UnsignedPayload{
		Payload:   payload,
		Secret:    g.webhookSecret,
		Timestamp: time.Now(),
	})
	ev, err := g.ParseWebhook(signed.Payload, signed.Header)
	if err != nil {
		return fmt.Errorf("webhook self-test: %w", err)
	}
	if ev.Type != "checkout.session.completed" || ev.UserID != "user_selftest" || ev.CourseID != 9 || ev.SessionID != "cs_selftest" {
		return fmt.Errorf("webhook self-test did not recover the payer and course")
	}
	return nil
}

func (g *stripeGateway) CreateCustomer(ctx context.Context, email, name, userID string) (string, error) {
	c, err := g.client.V1Customers.Create(ctx, &stripe.CustomerCreateParams{
		Email: stripe.String(email),
		Name:  stripe.String(name),
		Metadata: map[string]string{
			"clerk_user_id": userID,
		},
	})
	if err != nil {
		return "", err
	}
	return c.ID, nil
}

func (g *stripeGateway) CreateCheckoutSession(ctx context.Context, p CheckoutParams) (*CheckoutResult, error) {
	params := &stripe.CheckoutSessionCreateParams{
		Mode:                stripe.String(stripe.CheckoutSessionModePayment),
		SuccessURL:          stripe.String(p.SuccessURL),
		CancelURL:           stripe.String(p.CancelURL),
		ClientReferenceID:   stripe.String(p.UserID),
		AllowPromotionCodes: stripe.Bool(true),
		LineItems: []*stripe.CheckoutSessionCreateLineItemParams{
			{
				Price:    stripe.String(g.priceID),
				Quantity: stripe.Int64(1),
			},
		},
		Metadata: map[string]string{
			"clerk_user_id": p.UserID,
			"course_id":     strconv.Itoa(int(p.CourseID)),
		},
	}
	if p.CustomerID != "" {
		params.Customer = stripe.String(p.CustomerID)
	}
	session, err := g.client.V1CheckoutSessions.Create(ctx, params)
	if err != nil {
		return nil, err
	}
	return &CheckoutResult{URL: session.URL}, nil
}

func (g *stripeGateway) RetrieveCheckout(ctx context.Context, sessionID string) (*CompletedCheckout, error) {
	session, err := g.client.V1CheckoutSessions.Retrieve(ctx, sessionID, nil)
	if err != nil {
		return nil, err
	}
	out := checkoutFromSession(session)
	return &out, nil
}

func checkoutFromSession(session *stripe.CheckoutSession) CompletedCheckout {
	out := CompletedCheckout{
		SessionID: session.ID,
		Paid:      session.PaymentStatus == stripe.CheckoutSessionPaymentStatusPaid || session.PaymentStatus == stripe.CheckoutSessionPaymentStatusNoPaymentRequired,
		Source:    "purchase",
	}
	if session.AmountTotal == 0 {
		out.Source = "promo"
	}
	out.AmountCents = int32(session.AmountTotal)
	out.UserID = session.ClientReferenceID
	if session.Metadata != nil {
		if session.Metadata["clerk_user_id"] != "" {
			out.UserID = session.Metadata["clerk_user_id"]
		}
		if id, err := strconv.Atoi(session.Metadata["course_id"]); err == nil {
			out.CourseID = int32(id)
		}
	}
	if session.PaymentIntent != nil {
		out.PaymentIntentID = session.PaymentIntent.ID
	}
	return out
}

func (g *stripeGateway) GetPrice(ctx context.Context) (*PriceInfo, error) {
	price, err := g.client.V1Prices.Retrieve(ctx, g.priceID, &stripe.PriceRetrieveParams{
		Expand: []*string{stripe.String("product")},
	})
	if err != nil {
		return nil, err
	}
	name := ""
	if price.Product != nil {
		name = price.Product.Name
	}
	return &PriceInfo{
		AmountCents: price.UnitAmount,
		Currency:    string(price.Currency),
		Name:        name,
	}, nil
}

func (g *stripeGateway) ParseWebhook(payload []byte, sigHeader string) (*WebhookEvent, error) {
	if g.webhookSecret == "" {
		return nil, fmt.Errorf("STRIPE_WEBHOOK_SECRET is required")
	}
	// The Dashboard endpoint uses the account's current API version, which can
	// be a newer release train than stripe-go. We only read stable session
	// fields, so a train mismatch must not reject the event.
	event, err := webhook.ConstructEventWithOptions(payload, sigHeader, g.webhookSecret, webhook.ConstructEventOptions{
		IgnoreAPIVersionMismatch: true,
	})
	if err != nil {
		return nil, err
	}
	out := &WebhookEvent{Type: string(event.Type)}
	if event.Data == nil {
		return out, nil
	}
	switch event.Type {
	case stripe.EventTypeCheckoutSessionCompleted:
		var raw struct {
			ID                string            `json:"id"`
			ClientReferenceID string            `json:"client_reference_id"`
			AmountTotal       int64             `json:"amount_total"`
			PaymentIntent     json.RawMessage   `json:"payment_intent"`
			Metadata          map[string]string `json:"metadata"`
		}
		if err := json.Unmarshal(event.Data.Raw, &raw); err != nil {
			return nil, err
		}
		out.SessionID = raw.ID
		out.UserID = raw.ClientReferenceID
		if raw.Metadata != nil {
			if raw.Metadata["clerk_user_id"] != "" {
				out.UserID = raw.Metadata["clerk_user_id"]
			}
			if id, err := strconv.Atoi(raw.Metadata["course_id"]); err == nil {
				out.CourseID = int32(id)
			}
		}
		out.PaymentIntentID = jsonID(raw.PaymentIntent)
		out.AmountCents = int32(raw.AmountTotal)
		out.Source = "purchase"
		if raw.AmountTotal == 0 {
			out.Source = "promo"
		}
	case stripe.EventTypeChargeRefunded, stripe.EventTypeChargeDisputeCreated:
		var raw struct {
			PaymentIntent json.RawMessage `json:"payment_intent"`
		}
		if err := json.Unmarshal(event.Data.Raw, &raw); err != nil {
			return nil, err
		}
		out.PaymentIntentID = jsonID(raw.PaymentIntent)
	}
	return out, nil
}

func jsonID(raw json.RawMessage) string {
	if len(raw) == 0 || string(raw) == "null" {
		return ""
	}
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return s
	}
	var obj struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(raw, &obj); err == nil {
		return obj.ID
	}
	return ""
}
