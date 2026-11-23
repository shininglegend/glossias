package auth

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"glossias/src/pkg/database"
	"glossias/src/pkg/models"

	"github.com/clerk/clerk-sdk-go/v2"
	"github.com/jackc/pgx/v5/pgtype"
)

// stubClerk replaces the Clerk profile fetch for one test and counts calls.
func stubClerk(t *testing.T, fail bool) *atomic.Int32 {
	t.Helper()
	var calls atomic.Int32
	orig := fetchClerkUser
	fetchClerkUser = func(_ context.Context, id string) (*clerk.User, error) {
		calls.Add(1)
		if fail {
			return nil, errors.New("clerk unavailable")
		}
		emailID := "email-1"
		first, last := "Ada", "Lovelace"
		return &clerk.User{
			ID:                    id,
			PrimaryEmailAddressID: &emailID,
			EmailAddresses:        []*clerk.EmailAddress{{ID: emailID, EmailAddress: "ada@example.com"}},
			FirstName:             &first,
			LastName:              &last,
		}, nil
	}
	t.Cleanup(func() { fetchClerkUser = orig })
	return &calls
}

// setupSyncDB gives models a fresh cache (so no sync marks carry over) and a
// mock DB whose UpsertUser returns a row.
func setupSyncDB(t *testing.T, userID string) *database.MockDBTX {
	t.Helper()
	if err := models.SetCache(); err != nil {
		t.Fatal(err)
	}
	mockDB := database.NewMockDBTX()
	mockDB.StubQuery("name: UpsertUser :one", [][]any{{
		userID, "ada@example.com", "Ada Lovelace", pgtype.Bool{Valid: true}, pgtype.Timestamp{}, pgtype.Timestamp{},
	}}, nil)
	models.SetDB(mockDB)
	t.Cleanup(func() { models.SetDB(struct{}{}) })
	return mockDB
}

// serveDevUser sends one authenticated request (DEV_USER bypass) through
// Middleware and fails unless the handler saw the user.
func serveDevUser(t *testing.T, userID string) {
	t.Helper()
	var seen string
	h := Middleware(slog.New(slog.DiscardHandler))(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen = GetUserID(r)
	}))
	req := httptest.NewRequest(http.MethodGet, "/api/stories", nil)
	req.Header.Set("dev_auth", "12345678")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK || seen != userID {
		t.Fatalf("status %d, user %q; want 200 and %q", rr.Code, seen, userID)
	}
}

func TestAuthSyncsOncePerTTL(t *testing.T) {
	const id = "user_dev"
	t.Setenv("DEV_USER", id)
	clerkCalls := stubClerk(t, false)
	mockDB := setupSyncDB(t, id)

	serveDevUser(t, id)
	serveDevUser(t, id)

	if n := clerkCalls.Load(); n != 1 {
		t.Errorf("Clerk calls = %d, want 1", n)
	}
	if n := len(mockDB.Calls("name: UpsertUser")); n != 1 {
		t.Errorf("upserts = %d, want 1", n)
	}
}

func TestAuthResyncsAfterTTL(t *testing.T) {
	const id = "user_dev"
	t.Setenv("DEV_USER", id)
	clerkCalls := stubClerk(t, false)
	mockDB := setupSyncDB(t, id)

	serveDevUser(t, id)
	models.MarkUserSyncedAt(id, time.Now().Add(-11*time.Minute))
	serveDevUser(t, id)

	if n := clerkCalls.Load(); n != 2 {
		t.Errorf("Clerk calls = %d, want 2", n)
	}
	if n := len(mockDB.Calls("name: UpsertUser")); n != 2 {
		t.Errorf("upserts = %d, want 2", n)
	}
	if !models.UserRecentlySynced(id) {
		t.Error("re-sync was not recorded")
	}
}

func TestSyncUserClerkFailureKeepsProfile(t *testing.T) {
	const id = "user_jwt"
	clerkCalls := stubClerk(t, true)
	mockDB := setupSyncDB(t, id)
	logger := slog.New(slog.DiscardHandler)

	syncUser(context.Background(), id, logger)
	syncUser(context.Background(), id, logger)

	if n := clerkCalls.Load(); n != 2 {
		t.Errorf("Clerk calls = %d, want 2 (a failure must not count as a sync)", n)
	}
	if models.UserRecentlySynced(id) {
		t.Error("Clerk failure marked the user as synced")
	}
	if n := len(mockDB.Calls("name: UpsertUser")); n != 0 {
		t.Errorf("upserts = %d, want 0: an upsert with empty values blanks the stored email", n)
	}
	inserts := mockDB.Calls("name: InsertUserIfMissing")
	if len(inserts) != 2 {
		t.Fatalf("insert-if-missing calls = %d, want 2", len(inserts))
	}
	if !strings.Contains(inserts[0].SQL, "ON CONFLICT DO NOTHING") {
		t.Errorf("fallback insert may overwrite an existing row: %s", inserts[0].SQL)
	}
}

func TestSyncUserUpsertFailureNotMarked(t *testing.T) {
	const id = "user_jwt"
	stubClerk(t, false)
	if err := models.SetCache(); err != nil {
		t.Fatal(err)
	}
	mockDB := database.NewMockDBTX()
	mockDB.StubQuery("name: UpsertUser :one", nil, errors.New("unique violation"))
	models.SetDB(mockDB)
	t.Cleanup(func() { models.SetDB(struct{}{}) })

	var logs strings.Builder
	syncUser(context.Background(), id, slog.New(slog.NewTextHandler(&logs, nil)))

	if models.UserRecentlySynced(id) {
		t.Error("failed upsert marked the user as synced")
	}
	out := logs.String()
	if !strings.Contains(out, "user_id="+id) {
		t.Errorf("sync failure log lacks the user id: %s", out)
	}
	for _, pii := range []string{"ada@example.com", "Ada", "Lovelace"} {
		if strings.Contains(out, pii) {
			t.Errorf("sync failure log contains %q: %s", pii, out)
		}
	}
}

func TestAuthFailureLogsWarn(t *testing.T) {
	t.Setenv("DEV_USER", "")
	var logs strings.Builder
	logger := slog.New(slog.NewTextHandler(&logs, &slog.HandlerOptions{Level: slog.LevelDebug}))
	h := Middleware(logger)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler ran for an unauthenticated request")
	}))
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/api/stories", nil))

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("status %d, want 401", rr.Code)
	}
	out := logs.String()
	if !strings.Contains(out, "level=WARN") || !strings.Contains(out, "auth failed") {
		t.Errorf("want a WARN auth failed record, got: %s", out)
	}
	if strings.Contains(out, "level=ERROR") {
		t.Errorf("auth failure logged at ERROR: %s", out)
	}
}
