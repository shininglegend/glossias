package billing

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"glossias/src/auth"
	"glossias/src/pkg/models"

	"github.com/gorilla/mux"
	"github.com/stripe/stripe-go/v86"
)

type Handler struct {
	log     *slog.Logger
	gateway Gateway
}

func NewHandler(logger *slog.Logger, gateway Gateway) *Handler {
	return &Handler{log: logger, gateway: gateway}
}

// NoteGatewayStartup records whether the Stripe gateway could be built. With
// the paywall on and no gateway, students would be locked out with no way to
// pay, so payments pause (fail-open) and the administrator is alerted. A
// gateway that builds clears only that kind of pause: the persisted state is
// loaded before this runs, so a pause from an earlier start is cleared here
// once the configuration is fixed.
func NoteGatewayStartup(ctx context.Context, gatewayErr error) {
	if gatewayErr == nil {
		models.ResumePaymentsFrom(ctx, models.PauseReasonNotConfigured, "Stripe gateway configured at startup")
		return
	}
	if !models.PaywallEnabled() {
		return
	}
	models.PausePayments(ctx, models.PauseReasonNotConfigured,
		"PAYWALL_ENABLED is on but Stripe could not be configured: "+gatewayErr.Error())
}

// PublicAppURL is this environment's PUBLIC_APP_URL (default local Vite).
// It builds the checkout return URLs and tags each session so the webhook
// can tell this environment's checkouts from another's.
func PublicAppURL() string {
	if v := strings.TrimRight(os.Getenv("PUBLIC_APP_URL"), "/"); v != "" {
		return v
	}
	return "http://localhost:5173"
}

// checkoutEnabled is false without a gateway or while payments are paused
// after a failed grant (models.PaymentsPaused), which also lifts the paywall.
func (h *Handler) checkoutEnabled() bool {
	return h.gateway != nil && !models.PaymentsPaused()
}

func (h *Handler) RegisterRoutes(router *mux.Router) {
	router.HandleFunc("/pricing", h.GetPricing).Methods("GET", "OPTIONS")
	router.HandleFunc("/checkout", h.CreateCheckout).Methods("POST", "OPTIONS")
	router.HandleFunc("/checkout/confirm", h.ConfirmCheckout).Methods("POST", "OPTIONS")
}

type pricingCourse struct {
	CourseID     int    `json:"course_id"`
	CourseNumber string `json:"course_number"`
	Name         string `json:"name"`
	StoryCount   int    `json:"story_count"`
	HasAccess    bool   `json:"has_access"`
	ExpiresAt    string `json:"expires_at,omitempty"`
}

func (h *Handler) GetPricing(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.GetUserIDWithOk(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	// Without a gateway there is no price to show, but the page must still
	// load so it can explain that payments are paused (and the paywall is
	// lifted) instead of erroring.
	price := &PriceInfo{Currency: "usd"}
	if h.gateway != nil {
		p, err := h.gateway.GetPrice(r.Context())
		if err != nil {
			h.log.Error("failed to retrieve price", "error", err)
			h.noteGatewayResult(r, "GetPrice", err)
			if !models.PaymentsPaused() {
				http.Error(w, "Failed to load pricing", http.StatusBadGateway)
				return
			}
			// Stripe is down and payments have paused: render the page with
			// payments disabled so it can say so, rather than erroring.
		} else {
			h.noteGatewayResult(r, "GetPrice", nil)
			price = p
		}
	}

	courses, err := models.GetCoursesForUser(r.Context(), userID)
	if err != nil {
		h.log.Error("failed to list enrollments", "error", err)
		http.Error(w, "Failed to load pricing", http.StatusInternalServerError)
		return
	}
	ids := make([]int32, 0, len(courses))
	for _, c := range courses {
		if !c.IsTrial {
			ids = append(ids, int32(c.CourseID))
		}
	}
	counts, err := models.StoryCountsForCourses(r.Context(), ids)
	if err != nil {
		h.log.Error("failed to count stories", "error", err)
		http.Error(w, "Failed to load pricing", http.StatusInternalServerError)
		return
	}

	qCourse := r.URL.Query().Get("course_id")
	var selected *pricingCourse
	var payable []pricingCourse
	for _, c := range courses {
		if c.IsTrial {
			continue
		}
		pc := pricingCourse{
			CourseID:     c.CourseID,
			CourseNumber: c.CourseNumber,
			Name:         c.Name,
			StoryCount:   counts[int32(c.CourseID)],
			HasAccess:    c.HasAccess,
		}
		if c.ExpiresAt != nil {
			pc.ExpiresAt = c.ExpiresAt.UTC().Format(time.RFC3339)
		}
		if qCourse != "" && strconv.Itoa(c.CourseID) == qCourse {
			cp := pc
			selected = &cp
		}
		if !c.HasAccess {
			payable = append(payable, pc)
		}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"amount_cents":     price.AmountCents,
		"currency":         price.Currency,
		"name":             price.Name,
		"course":           selected,
		"payable_courses":  payable,
		"payments_enabled": h.checkoutEnabled(),
	})
}

type checkoutRequest struct {
	CourseID int32 `json:"course_id"`
}

func (h *Handler) CreateCheckout(w http.ResponseWriter, r *http.Request) {
	if h.gateway == nil {
		http.Error(w, "Payments are not configured", http.StatusServiceUnavailable)
		return
	}
	if !h.checkoutEnabled() {
		http.Error(w, "Payments are temporarily paused; paid content is open in the meantime", http.StatusServiceUnavailable)
		return
	}
	userID, ok := auth.GetUserIDWithOk(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	var req checkoutRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.CourseID == 0 {
		http.Error(w, "course_id is required", http.StatusBadRequest)
		return
	}

	course, err := models.GetCourse(r.Context(), req.CourseID)
	if err != nil {
		if errors.Is(err, models.ErrNotFound) {
			http.Error(w, "Course not found", http.StatusNotFound)
			return
		}
		h.log.Error("failed to load course", "error", err)
		http.Error(w, "Failed to start checkout", http.StatusInternalServerError)
		return
	}
	if course.IsTrial {
		http.Error(w, "Trial courses do not require payment", http.StatusBadRequest)
		return
	}
	enrolled, err := models.IsUserEnrolledInCourse(r.Context(), userID, req.CourseID)
	if err != nil {
		h.log.Error("failed to check enrollment", "error", err)
		http.Error(w, "Failed to start checkout", http.StatusInternalServerError)
		return
	}
	if !enrolled {
		http.Error(w, "Not enrolled in this course", http.StatusBadRequest)
		return
	}
	if exp, err := models.ActiveAccessExpiresAt(r.Context(), userID, req.CourseID); err != nil {
		h.log.Error("failed to check entitlement", "error", err)
		http.Error(w, "Failed to start checkout", http.StatusInternalServerError)
		return
	} else if exp != nil {
		writeJSON(w, http.StatusOK, map[string]any{
			"already_active": true,
			"expires_at":     exp.UTC().Format(time.RFC3339),
		})
		return
	}

	user, err := models.GetUser(r.Context(), userID)
	if err != nil {
		h.log.Error("failed to load user", "error", err)
		http.Error(w, "Failed to start checkout", http.StatusInternalServerError)
		return
	}
	customerID, err := models.UserStripeCustomerID(r.Context(), userID)
	if err != nil {
		h.log.Error("failed to load stripe customer", "error", err)
		http.Error(w, "Failed to start checkout", http.StatusInternalServerError)
		return
	}
	if customerID == "" {
		customerID, err = h.gateway.CreateCustomer(r.Context(), user.Email, user.Name, userID)
		if err != nil {
			h.log.Error("failed to create stripe customer", "error", err)
			h.noteGatewayResult(r, "CreateCustomer", err)
			http.Error(w, "Failed to start checkout", http.StatusBadGateway)
			return
		}
		if err := models.SetUserStripeCustomerID(r.Context(), userID, customerID); err != nil {
			// Bookkeeping only: the Stripe customer exists and the session
			// below carries the user id in metadata, so the payment still
			// grants. Failing here would block a willing payer; a retry would
			// only create another orphan customer.
			h.log.Error("failed to store stripe customer; continuing to checkout", "error", err, "customer", customerID)
		}
	}

	appURL := PublicAppURL()
	session, err := h.gateway.CreateCheckoutSession(r.Context(), CheckoutParams{
		CustomerID: customerID,
		UserID:     userID,
		CourseID:   req.CourseID,
		SuccessURL: appURL + "/checkout/success?session_id={CHECKOUT_SESSION_ID}",
		CancelURL:  fmt.Sprintf("%s/pricing?course=%d", appURL, req.CourseID),
		AppURL:     appURL,
	})
	if err != nil {
		h.log.Error("failed to create checkout session", "error", err)
		h.noteGatewayResult(r, "CreateCheckoutSession", err)
		http.Error(w, "Failed to start checkout", http.StatusBadGateway)
		return
	}
	h.noteGatewayResult(r, "CreateCheckoutSession", nil)
	writeJSON(w, http.StatusOK, map[string]any{"url": session.URL})
}

type confirmRequest struct {
	SessionID string `json:"session_id"`
}

func (h *Handler) ConfirmCheckout(w http.ResponseWriter, r *http.Request) {
	if h.gateway == nil {
		http.Error(w, "Payments are not configured", http.StatusServiceUnavailable)
		return
	}
	userID, ok := auth.GetUserIDWithOk(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	var req confirmRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.SessionID == "" {
		http.Error(w, "session_id is required", http.StatusBadRequest)
		return
	}
	session, err := h.gateway.RetrieveCheckout(r.Context(), req.SessionID)
	if err != nil {
		h.log.Error("failed to retrieve checkout session", "error", err, "session", req.SessionID)
		h.noteGatewayResult(r, "RetrieveCheckout", err)
		http.Error(w, "Failed to confirm payment", http.StatusBadGateway)
		return
	}
	h.noteGatewayResult(r, "RetrieveCheckout", nil)
	if session.UserID == "" || session.UserID != userID || session.CourseID == 0 {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}
	if !session.Paid {
		writeJSON(w, http.StatusOK, map[string]any{"pending": true})
		return
	}
	if err := models.GrantFromCheckoutSession(
		r.Context(),
		session.UserID,
		session.CourseID,
		session.SessionID,
		session.PaymentIntentID,
		session.AmountCents,
		session.Source,
	); err != nil {
		// The student has paid and we cannot record it: fail open and alert,
		// exactly as the webhook path does.
		models.PausePayments(r.Context(), models.PauseReasonGrantFailed,
			"checkout confirm grant failed for session "+session.SessionID+": "+err.Error())
		http.Error(w, "Failed to confirm payment", http.StatusInternalServerError)
		return
	}
	// A grant that lands here proves the write path works again.
	models.ResumePayments(r.Context(), "confirm grant recorded for session "+session.SessionID)
	exp, err := models.ActiveAccessExpiresAt(r.Context(), session.UserID, session.CourseID)
	if err != nil {
		h.log.Error("failed to read entitlement", "error", err, "session", session.SessionID)
		http.Error(w, "Failed to confirm payment", http.StatusInternalServerError)
		return
	}
	body := map[string]any{"ok": true, "course_id": session.CourseID}
	if exp != nil {
		body["expires_at"] = exp.UTC().Format(time.RFC3339)
	}
	writeJSON(w, http.StatusOK, body)
}

// noteGatewayResult feeds Stripe API outcomes into the fail-open streak.
// Only outages count: network errors and Stripe 5xx/429. A 4xx (bad session
// id, invalid price) is our bug or the caller's, not Stripe being down, and
// must not lift the paywall.
func (h *Handler) noteGatewayResult(r *http.Request, op string, err error) {
	if err == nil {
		models.RecordGatewaySuccess(r.Context(), op)
		return
	}
	if isGatewayOutage(err) {
		models.RecordGatewayError(r.Context(), op, err)
	}
}

// isGatewayOutage reports whether a Stripe call failed because Stripe was
// unreachable or erroring, as opposed to rejecting the request.
func isGatewayOutage(err error) bool {
	if errors.Is(err, context.Canceled) {
		// The caller went away mid-request; says nothing about Stripe.
		return false
	}
	var se *stripe.Error
	if errors.As(err, &se) {
		return se.HTTPStatusCode == 0 || se.HTTPStatusCode >= 500 || se.HTTPStatusCode == http.StatusTooManyRequests
	}
	// Not a Stripe API error: a transport failure (DNS, TLS, timeout).
	return true
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}
