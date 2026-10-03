package billing

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"glossias/src/auth"
	"glossias/src/pkg/database"
	"glossias/src/pkg/models"

	"github.com/jackc/pgx/v5/pgtype"
)

type mockGateway struct {
	price       *PriceInfo
	checkout    *CheckoutResult
	event       *WebhookEvent
	parseErr    error
	retrieved   *CompletedCheckout
	retrieveErr error
}

func (m *mockGateway) CreateCustomer(context.Context, string, string, string) (string, error) {
	return "cus_test", nil
}
func (m *mockGateway) CreateCheckoutSession(context.Context, CheckoutParams) (*CheckoutResult, error) {
	if m.checkout == nil {
		return &CheckoutResult{URL: "https://checkout.example/session"}, nil
	}
	return m.checkout, nil
}
func (m *mockGateway) GetPrice(context.Context) (*PriceInfo, error) {
	if m.price == nil {
		return &PriceInfo{AmountCents: 1, Currency: "usd", Name: "Access"}, nil
	}
	return m.price, nil
}
func (m *mockGateway) RetrieveCheckout(context.Context, string) (*CompletedCheckout, error) {
	if m.retrieveErr != nil {
		return nil, m.retrieveErr
	}
	return m.retrieved, nil
}
func (m *mockGateway) ParseWebhook([]byte, string) (*WebhookEvent, error) {
	if m.parseErr != nil {
		return nil, m.parseErr
	}
	return m.event, nil
}

func openCheckout(g Gateway) *Handler {
	return NewHandler(slog.New(slog.DiscardHandler), g)
}

func authReq(method, path, body string) *http.Request {
	var r *http.Request
	if body == "" {
		r = httptest.NewRequest(method, path, nil)
	} else {
		r = httptest.NewRequest(method, path, strings.NewReader(body))
	}
	return r.WithContext(context.WithValue(r.Context(), auth.UserIDKey, "user-1"))
}

func stubCourse(mock *database.MockDBTX, id int32, trial bool) {
	mock.StubQuery("name: GetCourse :one", [][]any{{
		id, "LAT 101", "Latin", pgtype.Text{}, trial, pgtype.Timestamp{}, pgtype.Timestamp{},
	}}, nil)
}

func TestCreateCheckout_RejectsTrial(t *testing.T) {
	mockDB := database.NewMockDBTX()
	stubCourse(mockDB, 4, true)
	models.SetDB(mockDB)
	t.Cleanup(func() { models.SetDB(struct{}{}) })

	h := openCheckout(&mockGateway{})
	rr := httptest.NewRecorder()
	h.CreateCheckout(rr, authReq("POST", "/api/checkout", `{"course_id":4}`))
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status %d, want 400: %s", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), "Trial") {
		t.Fatalf("body %q", rr.Body.String())
	}
}

func TestCreateCheckout_RejectsNotEnrolled(t *testing.T) {
	mockDB := database.NewMockDBTX()
	stubCourse(mockDB, 5, false)
	mockDB.StubQuery("IsUserEnrolledInCourse", [][]any{{false}}, nil)
	models.SetDB(mockDB)
	t.Cleanup(func() { models.SetDB(struct{}{}) })

	h := openCheckout(&mockGateway{})
	rr := httptest.NewRecorder()
	h.CreateCheckout(rr, authReq("POST", "/api/checkout", `{"course_id":5}`))
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status %d, want 400: %s", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), "Not enrolled") {
		t.Fatalf("body %q", rr.Body.String())
	}
}

func TestCreateCheckout_AlreadyActive(t *testing.T) {
	mockDB := database.NewMockDBTX()
	stubCourse(mockDB, 6, false)
	mockDB.StubQuery("IsUserEnrolledInCourse", [][]any{{true}}, nil)
	exp := pgtype.Timestamptz{Time: time.Now().Add(24 * time.Hour), Valid: true}
	mockDB.StubQuery("GetActiveEntitlementForUserCourse", [][]any{{
		int64(1), "user-1", int32(6), "purchase",
		pgtype.Timestamptz{Time: time.Now(), Valid: true},
		exp, pgtype.Int4{}, pgtype.Text{}, pgtype.Text{}, pgtype.Text{},
	}}, nil)
	models.SetDB(mockDB)
	t.Cleanup(func() { models.SetDB(struct{}{}) })

	h := openCheckout(&mockGateway{})
	rr := httptest.NewRecorder()
	h.CreateCheckout(rr, authReq("POST", "/api/checkout", `{"course_id":6}`))
	if rr.Code != http.StatusOK {
		t.Fatalf("status %d, want 200: %s", rr.Code, rr.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body["already_active"] != true {
		t.Fatalf("body %v", body)
	}
}

func TestHandleWebhook_GrantsOnce(t *testing.T) {
	mockDB := database.NewMockDBTX()
	mockDB.StubQuery("GetLatestEntitlementExpiryForUserCourse", nil, nil)
	mockDB.StubExecRows("InsertAccessEntitlement", 1)
	models.SetDB(mockDB)
	t.Cleanup(func() { models.SetDB(struct{}{}) })

	h := NewHandler(slog.New(slog.DiscardHandler), &mockGateway{
		event: &WebhookEvent{
			Type:            "checkout.session.completed",
			SessionID:       "cs_new",
			UserID:          "user-1",
			CourseID:        7,
			PaymentIntentID: "pi_new",
			AmountCents:     1,
			Source:          "purchase",
		},
	})
	rr := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/webhooks/stripe", bytes.NewReader([]byte(`{}`)))
	h.HandleWebhook(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status %d: %s", rr.Code, rr.Body.String())
	}
	if n := len(mockDB.Calls("InsertAccessEntitlement")); n != 1 {
		t.Fatalf("inserts = %d, want 1", n)
	}
}

// A session the confirm endpoint already granted (or a Stripe retry) hits
// ON CONFLICT DO NOTHING: no error, and checkout must stay enabled.
func TestHandleWebhook_IdempotentRepeat(t *testing.T) {
	mockDB := database.NewMockDBTX()
	mockDB.StubQuery("GetLatestEntitlementExpiryForUserCourse", nil, nil)
	mockDB.StubExecRows("InsertAccessEntitlement", 0)
	models.SetDB(mockDB)
	t.Cleanup(func() { models.SetDB(struct{}{}) })

	h := NewHandler(slog.New(slog.DiscardHandler), &mockGateway{
		event: &WebhookEvent{
			Type:            "checkout.session.completed",
			SessionID:       "cs_new",
			UserID:          "user-1",
			CourseID:        7,
			PaymentIntentID: "pi_new",
			AmountCents:     1,
			Source:          "purchase",
		},
	})
	rr := httptest.NewRecorder()
	h.HandleWebhook(rr, httptest.NewRequest("POST", "/api/webhooks/stripe", bytes.NewReader([]byte(`{}`))))
	if rr.Code != http.StatusOK {
		t.Fatalf("status %d: %s", rr.Code, rr.Body.String())
	}
	if !h.checkoutEnabled() {
		t.Fatal("an already-granted session must not pause checkout")
	}
}

func TestHandleWebhook_GrantFailurePausesCheckout(t *testing.T) {
	t.Cleanup(func() { models.SetPaymentsPaused(false) })
	mockDB := database.NewMockDBTX()
	mockDB.StubQuery("GetLatestEntitlementExpiryForUserCourse", nil, nil)
	mockDB.StubExec("InsertAccessEntitlement", errors.New("connection reset"))
	models.SetDB(mockDB)
	t.Cleanup(func() { models.SetDB(struct{}{}) })

	h := NewHandler(slog.New(slog.DiscardHandler), &mockGateway{
		event: &WebhookEvent{
			Type: "checkout.session.completed", SessionID: "cs_fail",
			UserID: "user-1", CourseID: 7, Source: "purchase",
		},
	})
	rr := httptest.NewRecorder()
	h.HandleWebhook(rr, httptest.NewRequest("POST", "/api/webhooks/stripe", bytes.NewReader([]byte(`{}`))))
	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("status %d, want 500: %s", rr.Code, rr.Body.String())
	}
	if h.checkoutEnabled() {
		t.Fatal("checkout should pause when a webhook cannot grant access")
	}
	if !models.PaymentsPaused() {
		t.Fatal("a failed grant should pause payments for the paywall too")
	}
}

func TestConfirmCheckout_GrantsPaidSession(t *testing.T) {
	mockDB := database.NewMockDBTX()
	mockDB.StubQuery("GetLatestEntitlementExpiryForUserCourse", nil, nil)
	exp := time.Now().Add(time.Hour)
	mockDB.StubExecRows("InsertAccessEntitlement", 1)
	mockDB.StubQuery("GetActiveEntitlementForUserCourse", [][]any{{
		int64(1), "user-1", int32(7), "purchase",
		pgtype.Timestamptz{Time: time.Now(), Valid: true},
		pgtype.Timestamptz{Time: exp, Valid: true},
		pgtype.Int4{}, pgtype.Text{String: "cs_paid", Valid: true},
		pgtype.Text{String: "pi_paid", Valid: true}, pgtype.Text{},
	}}, nil)
	models.SetDB(mockDB)
	t.Cleanup(func() { models.SetDB(struct{}{}) })

	h := NewHandler(slog.New(slog.DiscardHandler), &mockGateway{
		retrieved: &CompletedCheckout{
			SessionID: "cs_paid", UserID: "user-1", CourseID: 7,
			PaymentIntentID: "pi_paid", AmountCents: 1, Source: "purchase", Paid: true,
		},
	})
	models.SetPaymentsPaused(true)
	t.Cleanup(func() { models.SetPaymentsPaused(false) })
	rr := httptest.NewRecorder()
	h.ConfirmCheckout(rr, authReq("POST", "/api/checkout/confirm", `{"session_id":"cs_paid"}`))
	if rr.Code != http.StatusOK {
		t.Fatalf("status %d: %s", rr.Code, rr.Body.String())
	}
	if !h.checkoutEnabled() {
		t.Fatal("a successful confirm grant should re-enable checkout")
	}
	var body map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body["ok"] != true {
		t.Fatalf("body %v", body)
	}
	if n := len(mockDB.Calls("InsertAccessEntitlement")); n != 1 {
		t.Fatalf("inserts = %d, want 1", n)
	}
}

func TestConfirmCheckout_RejectsOtherUser(t *testing.T) {
	h := NewHandler(slog.New(slog.DiscardHandler), &mockGateway{
		retrieved: &CompletedCheckout{
			SessionID: "cs_other", UserID: "someone-else", CourseID: 7, Paid: true, Source: "purchase",
		},
	})
	rr := httptest.NewRecorder()
	h.ConfirmCheckout(rr, authReq("POST", "/api/checkout/confirm", `{"session_id":"cs_other"}`))
	if rr.Code != http.StatusForbidden {
		t.Fatalf("status %d, want 403: %s", rr.Code, rr.Body.String())
	}
}
