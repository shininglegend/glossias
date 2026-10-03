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
	sig := r.Header.Get("Stripe-Signature")
	event, err := h.gateway.ParseWebhook(payload, sig)
	if err != nil {
		if sig == "" {
			// Not a Stripe delivery (no signature at all): a scanner or a
			// misrouted request. Refuse it but never let it count toward the
			// fail-open streak.
			h.log.Warn("stripe webhook rejected: no Stripe-Signature header", "error", err, "ip", r.RemoteAddr)
		} else {
			// Signed but not by our secret: wrong STRIPE_WEBHOOK_SECRET, a
			// second Dashboard endpoint, or clock skew. Repeated rejections
			// pause payments (fail-open) and alert.
			models.RecordWebhookRejected(r.Context(), err.Error())
		}
		http.Error(w, "Invalid signature", http.StatusBadRequest)
		return
	}
	models.RecordWebhookVerified(r.Context())
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
			models.PausePayments(r.Context(), models.PauseReasonGrantFailed,
				"webhook grant failed for session "+event.SessionID+": "+err.Error())
			http.Error(w, "Failed to grant access", http.StatusInternalServerError)
			return
		}
		models.ResumePayments(r.Context(), "webhook grant recorded for session "+event.SessionID)
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
