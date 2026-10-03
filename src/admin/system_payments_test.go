package admin

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"glossias/src/auth"
	"glossias/src/pkg/database"
	"glossias/src/pkg/models"

	"github.com/jackc/pgx/v5/pgtype"
)

func stubSuperAdmin(t *testing.T, super bool) {
	t.Helper()
	mockDB := database.NewMockDBTX()
	mockDB.StubQuery("name: GetUser :one", [][]any{{
		"admin-1", "a@example.com", "Admin", pgtype.Bool{Bool: super, Valid: true}, pgtype.Timestamp{}, pgtype.Timestamp{},
	}}, nil)
	models.SetDB(mockDB)
	t.Cleanup(func() { models.SetDB(struct{}{}) })
}

func adminReq(method, path string) *http.Request {
	r := httptest.NewRequest(method, path, nil)
	return r.WithContext(context.WithValue(r.Context(), auth.UserIDKey, "admin-1"))
}

func TestResumePayments_SuperAdminClearsPause(t *testing.T) {
	stubSuperAdmin(t, true)
	models.SetPaymentsPaused(true)
	t.Cleanup(func() { models.SetPaymentsPaused(false) })
	h := NewHandler(slog.New(slog.DiscardHandler))

	rr := httptest.NewRecorder()
	h.paymentsStatusHandler(rr, adminReq(http.MethodGet, "/api/admin/system/payments"))
	if rr.Code != http.StatusOK {
		t.Fatalf("status %d: %s", rr.Code, rr.Body.String())
	}
	var st models.PaymentsStatus
	if err := json.Unmarshal(rr.Body.Bytes(), &st); err != nil {
		t.Fatal(err)
	}
	if !st.Paused || st.Reason != models.PauseReasonManual {
		t.Fatalf("status = %+v, want paused/manual", st)
	}

	rr = httptest.NewRecorder()
	h.resumePaymentsHandler(rr, adminReq(http.MethodPost, "/api/admin/system/payments/resume"))
	if rr.Code != http.StatusOK {
		t.Fatalf("status %d: %s", rr.Code, rr.Body.String())
	}
	var body struct {
		Resumed bool                  `json:"resumed"`
		Status  models.PaymentsStatus `json:"status"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if !body.Resumed || body.Status.Paused {
		t.Fatalf("body = %+v, want resumed and not paused", body)
	}
	if models.PaymentsPaused() {
		t.Fatal("payments should be resumed")
	}

	// Resuming when not paused is a no-op that still answers 200.
	rr = httptest.NewRecorder()
	h.resumePaymentsHandler(rr, adminReq(http.MethodPost, "/api/admin/system/payments/resume"))
	if rr.Code != http.StatusOK || !json.Valid(rr.Body.Bytes()) {
		t.Fatalf("status %d: %s", rr.Code, rr.Body.String())
	}
	json.Unmarshal(rr.Body.Bytes(), &body)
	if body.Resumed {
		t.Fatal("nothing to resume should report resumed=false")
	}
}

func TestResumePayments_RequiresSuperAdmin(t *testing.T) {
	stubSuperAdmin(t, false)
	models.SetPaymentsPaused(true)
	t.Cleanup(func() { models.SetPaymentsPaused(false) })
	h := NewHandler(slog.New(slog.DiscardHandler))

	rr := httptest.NewRecorder()
	h.resumePaymentsHandler(rr, adminReq(http.MethodPost, "/api/admin/system/payments/resume"))
	if rr.Code != http.StatusForbidden {
		t.Fatalf("status %d, want 403", rr.Code)
	}
	if !models.PaymentsPaused() {
		t.Fatal("a course admin must not be able to resume payments")
	}
	rr = httptest.NewRecorder()
	h.paymentsStatusHandler(rr, adminReq(http.MethodGet, "/api/admin/system/payments"))
	if rr.Code != http.StatusForbidden {
		t.Fatalf("status %d, want 403", rr.Code)
	}
}
