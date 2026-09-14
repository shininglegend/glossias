package billing

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"

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

type Gateway interface {
	CreateCustomer(ctx context.Context, email, name, userID string) (string, error)
	CreateCheckoutSession(ctx context.Context, p CheckoutParams) (*CheckoutResult, error)
	GetPrice(ctx context.Context) (*PriceInfo, error)
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
	return &stripeGateway{
		client:        stripe.NewClient(key),
		priceID:       priceID,
		webhookSecret: os.Getenv("STRIPE_WEBHOOK_SECRET"),
	}, nil
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
	event, err := webhook.ConstructEvent(payload, sigHeader, g.webhookSecret)
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
