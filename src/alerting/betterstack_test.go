package alerting

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"glossias/src/pkg/models"
)

type captured struct {
	mu   sync.Mutex
	reqs []struct {
		Path, Auth string
		Body       map[string]any
	}
}

func (c *captured) handler(status int, respBody string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		var body map[string]any
		_ = json.Unmarshal(raw, &body)
		c.mu.Lock()
		c.reqs = append(c.reqs, struct {
			Path, Auth string
			Body       map[string]any
		}{r.URL.Path, r.Header.Get("Authorization"), body})
		c.mu.Unlock()
		w.WriteHeader(status)
		_, _ = w.Write([]byte(respBody))
	}
}

func TestUptimeAPI_OpensThenResolvesIncident(t *testing.T) {
	c := &captured{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/incidents" {
			c.handler(http.StatusCreated, `{"data":{"id":"inc_42","type":"incident"}}`)(w, r)
			return
		}
		c.handler(http.StatusOK, `{"data":{"id":"inc_42"}}`)(w, r)
	}))
	defer srv.Close()

	a := &BetterStack{
		log: slog.New(slog.DiscardHandler), client: srv.Client(),
		uptimeAPI: srv.URL, token: "tok", requester: "ops@example.com", service: "svc",
	}
	a.PaymentsPaused(context.Background(), models.PauseReasonGrantFailed, "insert failed")
	a.Wait()
	// A second pause while open must not open a second incident.
	a.PaymentsPaused(context.Background(), models.PauseReasonGrantFailed, "again")
	a.Wait()
	a.PaymentsResumed(context.Background(), "grant ok")
	a.Wait()

	if len(c.reqs) != 2 {
		t.Fatalf("requests = %d, want open + resolve", len(c.reqs))
	}
	open := c.reqs[0]
	if open.Path != "/incidents" || open.Auth != "Bearer tok" {
		t.Fatalf("open = %+v", open)
	}
	if open.Body["requester_email"] != "ops@example.com" || open.Body["call"] != true {
		t.Fatalf("open body = %v", open.Body)
	}
	if c.reqs[1].Path != "/incidents/inc_42/resolve" {
		t.Fatalf("resolve path = %q", c.reqs[1].Path)
	}
	if a.incidentID != "" {
		t.Fatal("incident id should clear after resolve")
	}
}

func TestWebhookURL_PostsPauseAndResume(t *testing.T) {
	c := &captured{}
	srv := httptest.NewServer(c.handler(http.StatusOK, `ok`))
	defer srv.Close()

	a := &BetterStack{log: slog.New(slog.DiscardHandler), client: srv.Client(), webhookURL: srv.URL, service: "svc"}
	a.PaymentsPaused(context.Background(), models.PauseReasonWebhookRejected, "3 bad sigs")
	a.PaymentsResumed(context.Background(), "verified")
	a.Wait()

	if len(c.reqs) != 2 {
		t.Fatalf("requests = %d", len(c.reqs))
	}
	events := map[any]bool{}
	for _, r := range c.reqs {
		events[r.Body["event"]] = true
	}
	if !events["payments_paused"] || !events["payments_resumed"] {
		t.Fatalf("events = %v", events)
	}
}

func TestNewBetterStackFromEnv(t *testing.T) {
	t.Setenv(envUptimeToken, "")
	t.Setenv(envRequesterEmail, "")
	t.Setenv(envWebhookURL, "")
	if _, ok := NewBetterStackFromEnv(slog.New(slog.DiscardHandler)); ok {
		t.Fatal("no env should mean no alerter")
	}
	t.Setenv(envUptimeToken, "tok")
	if _, ok := NewBetterStackFromEnv(slog.New(slog.DiscardHandler)); ok {
		t.Fatal("a token without a requester email cannot open incidents and must not count as configured")
	}
	t.Setenv(envRequesterEmail, "ops@example.com")
	a, ok := NewBetterStackFromEnv(slog.New(slog.DiscardHandler))
	if !ok || a.Destinations() != "uptime-api" {
		t.Fatalf("ok=%v dest=%q", ok, a.Destinations())
	}
	t.Setenv(envWebhookURL, "https://example.invalid/hook")
	a, _ = NewBetterStackFromEnv(slog.New(slog.DiscardHandler))
	if a.Destinations() != "uptime-api,webhook" {
		t.Fatalf("dest=%q", a.Destinations())
	}
}

func TestUnreachableDestinationDoesNotPanicOrBlock(t *testing.T) {
	a := &BetterStack{log: slog.New(slog.DiscardHandler), client: &http.Client{}, uptimeAPI: "http://127.0.0.1:1", token: "t", requester: "x@y", webhookURL: "http://127.0.0.1:1"}
	a.PaymentsPaused(context.Background(), models.PauseReasonGatewayDown, "stripe down")
	a.Wait() // returns: failures are logged, not propagated
}
