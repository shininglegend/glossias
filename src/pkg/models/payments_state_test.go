package models

import (
	"context"
	"errors"
	"testing"
	"time"

	"glossias/src/pkg/database"

	"github.com/jackc/pgx/v5/pgtype"
)

func TestPausePayments_AlertsOncePerTransition(t *testing.T) {
	a := installAlerter(t)
	mockDB := database.NewMockDBTX()
	SetDB(mockDB)
	t.Cleanup(func() { SetDB(struct{}{}) })
	ctx := context.Background()

	PausePayments(ctx, PauseReasonGrantFailed, "first")
	PausePayments(ctx, PauseReasonGrantFailed, "second")
	PausePayments(ctx, PauseReasonGrantFailed, "third")
	if !PaymentsPaused() {
		t.Fatal("payments should be paused")
	}
	if len(a.raised) != 1 {
		t.Fatalf("alerts = %d, want exactly one for repeated pauses", len(a.raised))
	}
	if n := len(mockDB.Calls("UpsertPaymentsState")); n != 1 {
		t.Fatalf("persist writes = %d, want 1 (only on transition)", n)
	}
	if got := mockDB.Calls("UpsertPaymentsState")[0].Args; got[0] != true || got[1] != string(PauseReasonGrantFailed) {
		t.Fatalf("persisted args = %v", got)
	}

	ResumePayments(ctx, "grant ok")
	ResumePayments(ctx, "grant ok again")
	if PaymentsPaused() {
		t.Fatal("payments should resume")
	}
	if len(a.resolved) != 1 || a.resolved[0] != PaymentsIncidentKey || a.proofs[0] != "grant ok" {
		t.Fatalf("resume alerts = %v/%v, want one resolve of the payments incident", a.resolved, a.proofs)
	}
	if inc := a.raised[0]; inc.Key != PaymentsIncidentKey || !containsAll(inc.Title, "grant_failed") || inc.Detail != "first" {
		t.Fatalf("incident = %+v", inc)
	}
	if n := len(mockDB.Calls("UpsertPaymentsState")); n != 2 {
		t.Fatalf("persist writes = %d, want 2", n)
	}
	if got := mockDB.Calls("UpsertPaymentsState")[1].Args; got[0] != false {
		t.Fatalf("resume should persist paused=false, got %v", got)
	}
}

func TestResumePaymentsFrom_IsReasonAware(t *testing.T) {
	a := installAlerter(t)
	ctx := context.Background()

	PausePayments(ctx, PauseReasonGrantFailed, "db down")
	// A verified webhook proves the handshake, not the write path.
	ResumePaymentsFrom(ctx, PauseReasonWebhookRejected, "webhook verified")
	if !PaymentsPaused() || PaymentsPauseReason() != PauseReasonGrantFailed {
		t.Fatal("a webhook verification must not clear a grant-failure pause")
	}
	// A successful Stripe call proves Stripe, not the write path.
	RecordGatewaySuccess(ctx, "GetPrice")
	if !PaymentsPaused() {
		t.Fatal("a Stripe API success must not clear a grant-failure pause")
	}
	// A recorded grant proves everything.
	ResumePayments(ctx, "grant recorded")
	if PaymentsPaused() {
		t.Fatal("a grant should resume payments")
	}
	if len(a.raised) != 1 || len(a.resolved) != 1 {
		t.Fatalf("alerts paused=%d resumed=%d", len(a.raised), len(a.resolved))
	}
}

func TestWebhookRejectStreak_PausesAtLimitAndResetsOnVerify(t *testing.T) {
	a := installAlerter(t)
	ctx := context.Background()

	for i := range WebhookRejectStreakLimit - 1 {
		RecordWebhookRejected(ctx, "bad sig")
		if PaymentsPaused() {
			t.Fatalf("paused after %d rejections, limit is %d", i+1, WebhookRejectStreakLimit)
		}
	}
	// A good delivery in between resets the count.
	RecordWebhookVerified(ctx)
	for range WebhookRejectStreakLimit - 1 {
		RecordWebhookRejected(ctx, "bad sig")
	}
	if PaymentsPaused() {
		t.Fatal("verified webhook should have reset the rejection streak")
	}
	RecordWebhookRejected(ctx, "bad sig")
	if !PaymentsPaused() || PaymentsPauseReason() != PauseReasonWebhookRejected {
		t.Fatalf("paused=%v reason=%q, want webhook_rejected", PaymentsPaused(), PaymentsPauseReason())
	}
	if len(a.raised) != 1 || !containsAll(a.raised[0].Title, string(PauseReasonWebhookRejected)) || a.raised[0].Key != PaymentsIncidentKey {
		t.Fatalf("alerts = %v", a.raised)
	}
	// The handshake working again lifts this pause.
	RecordWebhookVerified(ctx)
	if PaymentsPaused() {
		t.Fatal("a verified webhook should resume a webhook-rejected pause")
	}
	if len(a.resolved) != 1 {
		t.Fatalf("resume alerts = %d", len(a.resolved))
	}
}

func TestGatewayErrorStreak_PausesAndResumesOnSuccess(t *testing.T) {
	a := installAlerter(t)
	ctx := context.Background()
	err := errors.New("dial tcp: i/o timeout")

	for range GatewayErrorStreakLimit - 1 {
		RecordGatewayError(ctx, "GetPrice", err)
	}
	if PaymentsPaused() {
		t.Fatal("one Stripe blip must not pause payments")
	}
	RecordGatewayError(ctx, "CreateCheckoutSession", err)
	if !PaymentsPaused() || PaymentsPauseReason() != PauseReasonGatewayDown {
		t.Fatalf("paused=%v reason=%q", PaymentsPaused(), PaymentsPauseReason())
	}
	RecordGatewaySuccess(ctx, "GetPrice")
	if PaymentsPaused() {
		t.Fatal("Stripe answering again should resume a gateway pause")
	}
	if len(a.raised) != 1 || len(a.resolved) != 1 {
		t.Fatalf("alerts paused=%d resumed=%d", len(a.raised), len(a.resolved))
	}
}

func TestLoadPaymentsState_ReappliesPersistedPauseAndRealerts(t *testing.T) {
	a := installAlerter(t)
	mockDB := database.NewMockDBTX()
	since := time.Now().Add(-time.Hour)
	mockDB.StubQuery("GetPaymentsState", [][]any{{
		true, string(PauseReasonGrantFailed), "insert failed",
		pgtype.Timestamptz{Time: since, Valid: true},
	}}, nil)
	SetDB(mockDB)
	t.Cleanup(func() { SetDB(struct{}{}) })

	if err := LoadPaymentsState(context.Background()); err != nil {
		t.Fatal(err)
	}
	if !PaymentsPaused() || PaymentsPauseReason() != PauseReasonGrantFailed {
		t.Fatalf("paused=%v reason=%q", PaymentsPaused(), PaymentsPauseReason())
	}
	if len(a.raised) != 1 || !containsAll(a.raised[0].Detail, "still paused after restart", "insert failed") {
		t.Fatalf("a restart while paused must re-alert with the persisted detail; alerts = %+v", a.raised)
	}
	if n := len(mockDB.Calls("UpsertPaymentsState")); n != 0 {
		t.Fatalf("loading must not rewrite state; writes = %d", n)
	}
}

func TestLoadPaymentsState_NotPausedIsQuiet(t *testing.T) {
	a := installAlerter(t)
	mockDB := database.NewMockDBTX()
	mockDB.StubQuery("GetPaymentsState", [][]any{{false, "", "", pgtype.Timestamptz{}}}, nil)
	SetDB(mockDB)
	t.Cleanup(func() { SetDB(struct{}{}) })
	if err := LoadPaymentsState(context.Background()); err != nil {
		t.Fatal(err)
	}
	if PaymentsPaused() || len(a.raised) != 0 {
		t.Fatal("an unpaused record must not pause or alert")
	}
}

func TestSetPaymentsPaused_StillWorksAsManualSwitch(t *testing.T) {
	installAlerter(t)
	SetPaymentsPaused(true)
	if !PaymentsPaused() || PaymentsPauseReason() != PauseReasonManual {
		t.Fatalf("paused=%v reason=%q", PaymentsPaused(), PaymentsPauseReason())
	}
	SetPaymentsPaused(false)
	if PaymentsPaused() {
		t.Fatal("should resume")
	}
}
