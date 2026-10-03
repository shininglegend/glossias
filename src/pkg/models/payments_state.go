package models

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"glossias/src/pkg/generated/db"
)

// Payments pause ("fail-open").
//
// When the payment system cannot do its job, students must not be locked
// out: payments are paused, which refuses new checkouts and lifts the
// paywall, and the administrator is alerted. The state lives here so that
// every trigger goes through one transition: it is logged at ERROR once,
// persisted so a restart cannot forget it, and sent to the alerter once per
// change rather than per request.

// PauseReason names what broke. Resuming is reason-aware: proof that the
// webhook handshake works again clears a webhook pause but not a grant
// failure, while a successful grant proves the whole path and clears any.
type PauseReason string

const (
	// PauseReasonGrantFailed: a verified payment could not be recorded.
	PauseReasonGrantFailed PauseReason = "grant_failed"
	// PauseReasonWebhookRejected: Stripe's signed webhooks keep failing
	// verification (wrong STRIPE_WEBHOOK_SECRET, clock skew, wrong endpoint).
	PauseReasonWebhookRejected PauseReason = "webhook_rejected"
	// PauseReasonGatewayDown: Stripe's API keeps returning server errors or
	// is unreachable.
	PauseReasonGatewayDown PauseReason = "gateway_down"
	// PauseReasonNotConfigured: the paywall is on but the Stripe gateway
	// could not be built at startup (missing key/price, or the webhook
	// secret failed its self-test), so nobody could pay.
	PauseReasonNotConfigured PauseReason = "not_configured"
	// PauseReasonManual: set by an operator or a test.
	PauseReasonManual PauseReason = "manual"
)

// Consecutive failures before a streak-based trigger pauses payments. A
// single rejected request from a scanner or one dropped Stripe call must not
// lift the paywall; a misconfigured secret or a Stripe outage repeats.
const (
	WebhookRejectStreakLimit = 3
	GatewayErrorStreakLimit  = 2
)

// PaymentsAlerter receives one call per pause/resume transition. Both are
// called synchronously from the request path, so implementations must return
// quickly (do their network work in the background) and never panic.
type PaymentsAlerter interface {
	PaymentsPaused(ctx context.Context, reason PauseReason, detail string)
	PaymentsResumed(ctx context.Context, proof string)
}

type paymentsPauseState struct {
	mu            sync.Mutex
	paused        bool
	reason        PauseReason
	detail        string
	since         time.Time
	webhookReject int
	gatewayErrors int
	alerter       PaymentsAlerter
}

var payState paymentsPauseState

// SetPaymentsAlerter installs the alerter notified on each transition. nil
// disables alerting (transitions are still logged and persisted).
func SetPaymentsAlerter(a PaymentsAlerter) {
	payState.mu.Lock()
	defer payState.mu.Unlock()
	payState.alerter = a
}

// PaymentsPaused reports whether payments are paused (and the paywall lifted).
func PaymentsPaused() bool {
	payState.mu.Lock()
	defer payState.mu.Unlock()
	return payState.paused
}

// PaymentsStatus is the operator's view of the pause state.
type PaymentsStatus struct {
	Paused bool        `json:"paused"`
	Reason PauseReason `json:"reason,omitempty"`
	Detail string      `json:"detail,omitempty"`
	// Since is when the current pause began; zero when not paused.
	Since          time.Time `json:"since,omitempty"`
	PaywallEnabled bool      `json:"paywall_enabled"`
}

// GetPaymentsStatus returns the current pause state for the admin UI.
func GetPaymentsStatus() PaymentsStatus {
	payState.mu.Lock()
	defer payState.mu.Unlock()
	st := PaymentsStatus{Paused: payState.paused, PaywallEnabled: PaywallEnabled()}
	if st.Paused {
		st.Reason = payState.reason
		st.Detail = payState.detail
		st.Since = payState.since
	}
	return st
}

// PaymentsPauseReason returns why payments are paused, or "" when they are not.
func PaymentsPauseReason() PauseReason {
	payState.mu.Lock()
	defer payState.mu.Unlock()
	if !payState.paused {
		return ""
	}
	return payState.reason
}

// SetPaymentsPaused pauses or resumes payments with no specific cause. It is
// the operator/test entry point; request paths call PausePayments and
// ResumePayments so the reason is recorded.
func SetPaymentsPaused(paused bool) {
	if paused {
		PausePayments(context.Background(), PauseReasonManual, "set by operator")
		return
	}
	ResumePayments(context.Background(), "cleared by operator")
}

// PausePayments pauses payments if they are not already paused. The first
// call for a given outage logs at ERROR, persists the state and alerts; later
// calls while paused only update the stored detail.
func PausePayments(ctx context.Context, reason PauseReason, detail string) {
	payState.mu.Lock()
	wasPaused := payState.paused
	payState.paused = true
	payState.reason = reason
	payState.detail = detail
	if !wasPaused {
		payState.since = time.Now()
	}
	alerter := payState.alerter
	payState.mu.Unlock()

	if wasPaused {
		slog.Default().WarnContext(ctx, "payments still paused", "reason", reason, "detail", detail)
		return
	}
	slog.Default().ErrorContext(ctx, "payments paused; paywall lifted and checkout refused",
		"reason", reason, "detail", detail)
	persistPaymentsState(ctx, true, reason, detail)
	if alerter != nil {
		alerter.PaymentsPaused(ctx, reason, detail)
	}
}

// ResumePayments resumes payments whatever the pause reason: the caller has
// proof the end-to-end path works (a payment was recorded). It also clears
// the failure streaks.
func ResumePayments(ctx context.Context, proof string) {
	resumePayments(ctx, proof, func(PauseReason) bool { return true })
}

// ResumePaymentsFrom resumes payments only when they were paused for the
// given reason, and always clears that reason's failure streak. A verified
// webhook proves the handshake works but says nothing about a failed grant.
func ResumePaymentsFrom(ctx context.Context, reason PauseReason, proof string) {
	resumePayments(ctx, proof, func(current PauseReason) bool { return current == reason })
}

func resumePayments(ctx context.Context, proof string, applies func(PauseReason) bool) {
	payState.mu.Lock()
	payState.webhookReject = 0
	payState.gatewayErrors = 0
	if !payState.paused || !applies(payState.reason) {
		payState.mu.Unlock()
		return
	}
	reason := payState.reason
	pausedFor := time.Since(payState.since).Round(time.Second)
	payState.paused = false
	payState.reason = ""
	payState.detail = ""
	alerter := payState.alerter
	payState.mu.Unlock()

	slog.Default().InfoContext(ctx, "payments resumed; paywall restored",
		"was_paused_for", reason, "paused_duration", pausedFor.String(), "proof", proof)
	persistPaymentsState(ctx, false, "", "")
	if alerter != nil {
		alerter.PaymentsResumed(ctx, proof)
	}
}

// RecordWebhookRejected counts a Stripe webhook whose signature did not
// verify. Only requests carrying a Stripe-Signature header should be counted
// (the handler checks), so random POSTs do not build a streak. The streak
// pauses payments at WebhookRejectStreakLimit; a verified webhook resets it.
func RecordWebhookRejected(ctx context.Context, detail string) {
	payState.mu.Lock()
	payState.webhookReject++
	n := payState.webhookReject
	payState.mu.Unlock()
	if n < WebhookRejectStreakLimit {
		slog.Default().WarnContext(ctx, "stripe webhook rejected", "streak", n,
			"pause_at", WebhookRejectStreakLimit, "detail", detail)
		return
	}
	PausePayments(ctx, PauseReasonWebhookRejected,
		fmt.Sprintf("%d consecutive webhooks failed signature verification; last: %s", n, detail))
}

// RecordWebhookVerified notes that a signed Stripe webhook verified. It
// clears the rejection streak and lifts a pause that was caused by rejections.
func RecordWebhookVerified(ctx context.Context) {
	ResumePaymentsFrom(ctx, PauseReasonWebhookRejected, "a Stripe webhook verified")
}

// RecordGatewayError counts a Stripe API call that failed because Stripe was
// unreachable or returned a server error (callers filter out client errors
// such as a bad session id). The streak pauses payments at
// GatewayErrorStreakLimit; any successful Stripe call resets it.
func RecordGatewayError(ctx context.Context, op string, err error) {
	payState.mu.Lock()
	payState.gatewayErrors++
	n := payState.gatewayErrors
	payState.mu.Unlock()
	if n < GatewayErrorStreakLimit {
		slog.Default().WarnContext(ctx, "stripe api error", "op", op, "error", err,
			"streak", n, "pause_at", GatewayErrorStreakLimit)
		return
	}
	PausePayments(ctx, PauseReasonGatewayDown,
		fmt.Sprintf("%d consecutive Stripe API failures; last (%s): %v", n, op, err))
}

// RecordGatewaySuccess notes a Stripe API call that succeeded. It clears the
// error streak and lifts a pause that was caused by Stripe being down.
func RecordGatewaySuccess(ctx context.Context, op string) {
	ResumePaymentsFrom(ctx, PauseReasonGatewayDown, "Stripe API call succeeded: "+op)
}

// LoadPaymentsState restores the persisted pause on startup. A pause that was
// raised before a redeploy is re-applied and re-alerted, so a restart never
// hides a lifted paywall. Without a database (tests, mock) it is a no-op.
func LoadPaymentsState(ctx context.Context) error {
	if queries == nil {
		return nil
	}
	row, err := queries.GetPaymentsState(ctx)
	if err != nil {
		return err
	}
	if !row.Paused {
		return nil
	}
	payState.mu.Lock()
	payState.paused = true
	payState.reason = PauseReason(row.Reason)
	payState.detail = row.Detail
	payState.since = row.ChangedAt.Time
	alerter := payState.alerter
	payState.mu.Unlock()
	slog.Default().ErrorContext(ctx, "payments were paused before this start; paywall stays lifted",
		"reason", row.Reason, "detail", row.Detail, "since", row.ChangedAt.Time)
	if alerter != nil {
		alerter.PaymentsPaused(ctx, PauseReason(row.Reason),
			fmt.Sprintf("still paused after restart (since %s): %s", row.ChangedAt.Time.UTC().Format(time.RFC3339), row.Detail))
	}
	return nil
}

// persistPaymentsState is best effort: the in-memory flag already protects
// students, and a failing write is the very outage that paused payments.
func persistPaymentsState(ctx context.Context, paused bool, reason PauseReason, detail string) {
	if queries == nil {
		return
	}
	err := queries.UpsertPaymentsState(ctx, db.UpsertPaymentsStateParams{
		Paused: paused,
		Reason: string(reason),
		Detail: detail,
	})
	if err != nil {
		slog.Default().ErrorContext(ctx, "could not persist payments state; it will be lost on restart",
			"paused", paused, "error", err)
	}
}

// resetPaymentsStateForTest clears every counter and the pause. Tests only.
func resetPaymentsStateForTest() {
	payState.mu.Lock()
	defer payState.mu.Unlock()
	payState.paused = false
	payState.reason = ""
	payState.detail = ""
	payState.since = time.Time{}
	payState.webhookReject = 0
	payState.gatewayErrors = 0
}
