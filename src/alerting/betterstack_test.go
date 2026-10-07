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

// uptimeStub answers /incidents with a fresh id per open and accepts resolves.
func uptimeStub(c *captured) *httptest.Server {
	var mu sync.Mutex
	n := 0
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/incidents" {
			mu.Lock()
			n++
			id := "inc_" + string(rune('0'+n))
			mu.Unlock()
			c.handler(http.StatusCreated, `{"data":{"id":"`+id+`","type":"incident"}}`)(w, r)
			return
		}
		c.handler(http.StatusOK, `{"data":{}}`)(w, r)
	}))
}

var paymentsIncident = Incident{
	Key: "payments", Title: "Payments paused (grant_failed)",
	Summary: "Payments paused: grant_failed. Paywall lifted; checkout refused.", Detail: "insert failed",
}

func TestUptimeAPI_OpensThenResolvesIncident(t *testing.T) {
	c := &captured{}
	srv := uptimeStub(c)
	defer srv.Close()

	a := &BetterStack{
		log: slog.New(slog.DiscardHandler), client: srv.Client(),
		uptimeAPI: srv.URL, token: "tok", requester: "ops@example.com", service: "svc",
	}
	a.Raise(context.Background(), paymentsIncident)
	a.Wait()
	// A second raise of the same key while open must not open a second incident.
	again := paymentsIncident
	again.Detail = "again"
	a.Raise(context.Background(), again)
	a.Wait()
	a.Resolve(context.Background(), "payments", "grant ok")
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
	if open.Body["name"] != "svc: Payments paused (grant_failed)" || open.Body["summary"] != paymentsIncident.Summary || open.Body["description"] != "insert failed" {
		t.Fatalf("open body = %v", open.Body)
	}
	if c.reqs[1].Path != "/incidents/inc_1/resolve" {
		t.Fatalf("resolve path = %q", c.reqs[1].Path)
	}
	if a.openIncidentID("payments") != "" {
		t.Fatal("incident id should clear after resolve")
	}
}

func TestUptimeAPI_KeysAreIndependent(t *testing.T) {
	c := &captured{}
	srv := uptimeStub(c)
	defer srv.Close()

	a := &BetterStack{
		log: slog.New(slog.DiscardHandler), client: srv.Client(),
		uptimeAPI: srv.URL, token: "tok", requester: "ops@example.com", service: "svc",
	}
	a.Raise(context.Background(), paymentsIncident)
	a.Wait()
	a.Raise(context.Background(), Incident{Key: "ai_grading", Title: "AI grading failing", Summary: "5 calls failed", Detail: "401"})
	a.Wait()
	if len(c.reqs) != 2 {
		t.Fatalf("requests = %d, want two opens for two keys", len(c.reqs))
	}
	// Resolving one key leaves the other open.
	a.Resolve(context.Background(), "ai_grading", "a call succeeded")
	a.Wait()
	if len(c.reqs) != 3 || c.reqs[2].Path != "/incidents/inc_2/resolve" {
		t.Fatalf("requests = %+v", c.reqs)
	}
	if a.openIncidentID("payments") != "inc_1" || a.openIncidentID("ai_grading") != "" {
		t.Fatalf("payments=%q grading=%q", a.openIncidentID("payments"), a.openIncidentID("ai_grading"))
	}
	// Resolving a key with nothing open is a logged no-op, not a request.
	a.Resolve(context.Background(), "ai_grading", "again")
	a.Wait()
	if len(c.reqs) != 3 {
		t.Fatalf("requests = %d, want no request for an idle key", len(c.reqs))
	}
}

func TestWebhookURL_PostsRaiseAndResolve(t *testing.T) {
	c := &captured{}
	srv := httptest.NewServer(c.handler(http.StatusOK, `ok`))
	defer srv.Close()

	a := &BetterStack{log: slog.New(slog.DiscardHandler), client: srv.Client(), webhookURL: srv.URL, service: "svc"}
	a.Raise(context.Background(), Incident{Key: "payments", Title: "Payments paused (webhook_rejected)", Summary: "3 bad sigs"})
	a.Resolve(context.Background(), "payments", "verified")
	a.Wait()

	if len(c.reqs) != 2 {
		t.Fatalf("requests = %d", len(c.reqs))
	}
	events := map[any]bool{}
	for _, r := range c.reqs {
		events[r.Body["event"]] = true
		if r.Body["key"] != "payments" || r.Body["service"] != "svc" {
			t.Fatalf("payload = %v", r.Body)
		}
	}
	if !events["incident_raised"] || !events["incident_resolved"] {
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
	a.Raise(context.Background(), Incident{Key: "payments", Title: "Payments paused (gateway_down)", Detail: "stripe down"})
	a.Wait() // returns: failures are logged, not propagated
}
