package billing

import (
	"io"
	"net/http"

	"glossias/src/pkg/models"
)

func (h *Handler) HandleWebhook(w http.ResponseWriter, r *http.Request) {
	if h.gateway == nil {
		http.Error(w, "Payments are not configured", http.StatusServiceUnavailable)
		return
	}
	payload, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read body", http.StatusBadRequest)
		return
	}
	event, err := h.gateway.ParseWebhook(payload, r.Header.Get("Stripe-Signature"))
	if err != nil {
		h.log.Warn("stripe webhook rejected", "error", err)
		http.Error(w, "Invalid signature", http.StatusBadRequest)
		return
	}
	switch event.Type {
	case "checkout.session.completed":
		if event.UserID == "" || event.CourseID == 0 {
			// A Dashboard or CLI test event is signed but has no course to grant.
			// Signature success is enough to allow checkout; it does not write access.
			h.log.Info("stripe webhook verified; no course metadata to grant", "session", event.SessionID)
			w.WriteHeader(http.StatusOK)
			return
		}
		if err := models.GrantFromCheckoutSession(
			r.Context(),
			event.UserID,
			event.CourseID,
			event.SessionID,
			event.PaymentIntentID,
			event.AmountCents,
			event.Source,
		); err != nil {
			models.SetPaymentsPaused(true)
			h.log.Error("failed to grant access from webhook; payments paused and paywall lifted", "error", err, "session", event.SessionID)
			http.Error(w, "Failed to grant access", http.StatusInternalServerError)
			return
		}
		models.SetPaymentsPaused(false)
	case "charge.refunded", "charge.dispute.created":
		if event.PaymentIntentID == "" {
			w.WriteHeader(http.StatusOK)
			return
		}
		if err := models.ExpireEntitlementsForPaymentIntent(r.Context(), event.PaymentIntentID); err != nil {
			h.log.Error("failed to expire entitlement", "error", err, "pi", event.PaymentIntentID)
			http.Error(w, "Failed to expire access", http.StatusInternalServerError)
			return
		}
	}
	w.WriteHeader(http.StatusOK)
}
