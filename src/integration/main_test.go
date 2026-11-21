//go:build integration

// Package integration measures DB queries per request for every endpoint
// against a real, seeded Postgres (see src/pkg/testdb). Requests go through
// the production router (src/server) with a fake auth middleware in place of
// Clerk, so the counts match what production logs as db_queries on a request
// that syncs the user (fakeAuth upserts every time; production skips the
// upsert while a sync is less than 10 minutes old).
//
// Run: make test-integration (or go test -tags integration ./src/integration).
// models keeps its DB, cache and payments state in package globals, so no
// test here may call t.Parallel.
package integration

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"glossias/src/auth"
	"glossias/src/pkg/models"
	"glossias/src/pkg/testdb"
	"glossias/src/server"
)

// testUserHeader names the user a request is made as.
const testUserHeader = "X-Test-User"

var router http.Handler

// seedUsers maps a seeded user id to its email and name, filled from the
// dataset before any request is served.
var seedUsers = map[string][2]string{}

func TestMain(m *testing.M) {
	// Keep the run independent of the developer's .env and of external services.
	for _, k := range []string{
		"PAYWALL_ENABLED", "DEV_USER", "ANTHROPIC_API_KEY",
		"STRIPE_SECRET_KEY", "STRIPE_WEBHOOK_SECRET", "STRIPE_PRICE_ACCESS",
		"BETTERSTACK_UPTIME_TOKEN", "BETTERSTACK_REQUESTER_EMAIL", "BETTERSTACK_WEBHOOK_URL",
		"STORAGE_URL", "STORAGE_API_KEY",
	} {
		os.Unsetenv(k)
	}
	// INTEGRATION_LOG=1 shows the server's logs (to see why a case failed).
	var logOut io.Writer = io.Discard
	if os.Getenv("INTEGRATION_LOG") != "" {
		logOut = os.Stderr
	}
	logger := slog.New(slog.NewTextHandler(logOut, nil))
	slog.SetDefault(logger)
	if err := models.SetCache(); err != nil {
		panic(err)
	}
	// Pages that sign asset URLs need a storage server; a stub signs anything.
	storage := httptest.NewServer(http.HandlerFunc(fakeStorage))
	defer storage.Close()
	models.SetStorageClient(storage.URL, "test-key")
	// Built with PAYWALL_ENABLED unset: with it set and no Stripe key the
	// handler would pause payments, which lifts the paywall for every case.
	router = server.NewRouter(server.Config{
		Logger: logger,
		Auth:   fakeAuth,
	})
	code := m.Run()
	testdb.Terminate()
	os.Exit(code)
}

// fakeStorage answers the Supabase Storage calls models makes: listing
// buckets (the startup check) and signing object URLs.
func fakeStorage(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	switch {
	case r.Method == http.MethodGet && r.URL.Path == "/bucket":
		w.Write([]byte("[]"))
	case r.Method == http.MethodPost && strings.HasPrefix(r.URL.Path, "/object/sign/"):
		fmt.Fprintf(w, `{"signedURL":%q}`, r.URL.Path+"?token=test")
	default:
		http.NotFound(w, r)
	}
}

// fakeAuth stands in for auth.Middleware: it trusts the X-Test-User header
// instead of a Clerk JWT, but still upserts the user as production does on
// every request.
func fakeAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if id := r.Header.Get(testUserHeader); id != "" {
			// Production takes email and name from Clerk; here, from the seed.
			u, ok := seedUsers[id]
			if !ok {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}
			if _, err := models.UpsertUser(r.Context(), id, u[0], u[1]); err != nil {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}
			r = r.WithContext(context.WithValue(r.Context(), auth.UserIDKey, id))
		}
		next.ServeHTTP(w, r)
	})
}
