package alerting

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

// Environment:
//
//	BETTERSTACK_UPTIME_TOKEN     Uptime API token (Better Stack → Uptime →
//	                             Integrations → API tokens). Opens an incident
//	                             on Raise and resolves it on Resolve.
//	BETTERSTACK_REQUESTER_EMAIL  Who the incident is "requested by"; the Uptime
//	                             API requires it. Must be a team member.
//	BETTERSTACK_WEBHOOK_URL      Alternative or addition: any URL (such as an
//	                             Uptime "Incoming webhook" integration) that
//	                             receives a JSON POST on Raise and Resolve.
//
// With neither variable set, NewBetterStackFromEnv returns ok=false and the
// caller logs a startup warning; incidents are then visible only in logs.
const (
	envUptimeToken    = "BETTERSTACK_UPTIME_TOKEN"
	envRequesterEmail = "BETTERSTACK_REQUESTER_EMAIL"
	envWebhookURL     = "BETTERSTACK_WEBHOOK_URL"

	defaultUptimeAPI = "https://uptime.betterstack.com/api/v2"
	requestTimeout   = 10 * time.Second
)

// BetterStack implements Alerter against Better Stack Uptime. Network calls
// run in the background so a request path (the Stripe webhook, say) is never
// delayed by alerting.
type BetterStack struct {
	log        *slog.Logger
	client     *http.Client
	uptimeAPI  string
	token      string
	requester  string
	webhookURL string
	service    string

	mu sync.Mutex
	// incidents maps an incident key to the Uptime incident id that is open
	// for it. In memory only: an incident opened before a restart is not
	// auto-resolved by the new process (it logs and leaves it to the
	// operator), and a key opens at most one incident per process lifetime
	// of the outage.
	incidents map[string]string
	// wg lets tests (and a graceful shutdown) wait for in-flight sends.
	wg sync.WaitGroup
}

// NewBetterStackFromEnv builds the alerter from the environment. ok is false
// when no destination is configured.
func NewBetterStackFromEnv(logger *slog.Logger) (a *BetterStack, ok bool) {
	token := strings.TrimSpace(os.Getenv(envUptimeToken))
	requester := strings.TrimSpace(os.Getenv(envRequesterEmail))
	hook := strings.TrimSpace(os.Getenv(envWebhookURL))
	if token != "" && requester == "" {
		logger.Warn(envUptimeToken + " is set without " + envRequesterEmail + "; the Uptime API rejects incidents without a requester, so only the webhook URL (if any) will be used")
		token = ""
	}
	if token == "" && hook == "" {
		return nil, false
	}
	return &BetterStack{
		log:        logger,
		client:     &http.Client{Timeout: requestTimeout},
		uptimeAPI:  defaultUptimeAPI,
		token:      token,
		requester:  requester,
		webhookURL: hook,
		service:    "Logos Stories",
	}, true
}

// Destinations describes where alerts go, for the startup log line.
func (a *BetterStack) Destinations() string {
	var out []string
	if a.token != "" {
		out = append(out, "uptime-api")
	}
	if a.webhookURL != "" {
		out = append(out, "webhook")
	}
	return strings.Join(out, ",")
}

// Raise opens an incident for inc.Key (and/or posts the webhook) in the
// background. A key that already has an open incident is not opened again.
func (a *BetterStack) Raise(ctx context.Context, inc Incident) {
	a.async(func(ctx context.Context) {
		if a.token != "" {
			a.openIncident(ctx, inc)
		}
		if a.webhookURL != "" {
			a.postWebhook(ctx, map[string]any{
				"event":   "incident_raised",
				"service": a.service,
				"key":     inc.Key,
				"title":   inc.Title,
				"summary": inc.Summary,
				"detail":  inc.Detail,
				"message": fmt.Sprintf("%s: %s. %s %s", a.service, inc.Title, inc.Summary, inc.Detail),
				"at":      time.Now().UTC().Format(time.RFC3339),
			})
		}
	})
}

// Resolve closes the incident open for key (and/or posts the webhook) in the
// background. proof says what showed the condition has cleared.
func (a *BetterStack) Resolve(ctx context.Context, key, proof string) {
	a.async(func(ctx context.Context) {
		if a.token != "" {
			a.resolveIncident(ctx, key, proof)
		}
		if a.webhookURL != "" {
			a.postWebhook(ctx, map[string]any{
				"event":   "incident_resolved",
				"service": a.service,
				"key":     key,
				"proof":   proof,
				"message": fmt.Sprintf("%s: %s incident resolved. %s", a.service, key, proof),
				"at":      time.Now().UTC().Format(time.RFC3339),
			})
		}
	})
}

// Wait blocks until background sends finish. For tests and shutdown.
func (a *BetterStack) Wait() { a.wg.Wait() }

func (a *BetterStack) async(fn func(context.Context)) {
	a.wg.Add(1)
	go func() {
		defer a.wg.Done()
		defer func() {
			if r := recover(); r != nil {
				a.log.Error("alerting panic recovered", "panic", r)
			}
		}()
		// Detached from the caller's context: an HTTP response may already
		// be written by the time this runs.
		ctx, cancel := context.WithTimeout(context.Background(), requestTimeout)
		defer cancel()
		fn(ctx)
	}()
}

// openIncidentID returns the open incident id for key, or "".
func (a *BetterStack) openIncidentID(key string) string {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.incidents[key]
}

func (a *BetterStack) setIncidentID(key, id string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if id == "" {
		delete(a.incidents, key)
		return
	}
	if a.incidents == nil {
		a.incidents = make(map[string]string)
	}
	a.incidents[key] = id
}

func (a *BetterStack) openIncident(ctx context.Context, inc Incident) {
	if already := a.openIncidentID(inc.Key); already != "" {
		a.log.Info("better stack incident already open", "key", inc.Key, "incident", already)
		return
	}
	body := map[string]any{
		"requester_email": a.requester,
		"name":            fmt.Sprintf("%s: %s", a.service, inc.Title),
		"summary":         inc.Summary,
		"description":     inc.Detail,
		"call":            true,
		"sms":             true,
		"email":           true,
		"push":            true,
	}
	status, resp, err := a.uptime(ctx, http.MethodPost, "/incidents", body)
	if err != nil {
		a.log.Error("better stack: could not open incident", "key", inc.Key, "error", err)
		return
	}
	if status < 200 || status >= 300 {
		a.log.Error("better stack: incident rejected", "key", inc.Key, "status", status, "body", truncate(resp, 500))
		return
	}
	var parsed struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(resp, &parsed); err != nil || parsed.Data.ID == "" {
		a.log.Warn("better stack: incident opened but id not parsed; it will not auto-resolve", "key", inc.Key, "body", truncate(resp, 500))
		return
	}
	a.setIncidentID(inc.Key, parsed.Data.ID)
	a.log.Info("better stack incident opened", "key", inc.Key, "incident", parsed.Data.ID)
}

func (a *BetterStack) resolveIncident(ctx context.Context, key, proof string) {
	id := a.openIncidentID(key)
	if id == "" {
		// Opened by a previous process (before a restart) or never opened:
		// nothing to resolve from here.
		a.log.Info("better stack: no incident id on record to resolve", "key", key, "proof", proof)
		return
	}
	a.setIncidentID(key, "")
	status, resp, err := a.uptime(ctx, http.MethodPost, "/incidents/"+id+"/resolve", map[string]any{
		"resolved_by": a.requester,
	})
	if err != nil {
		a.log.Error("better stack: could not resolve incident", "key", key, "incident", id, "error", err)
		return
	}
	if status < 200 || status >= 300 {
		a.log.Error("better stack: resolve rejected", "key", key, "incident", id, "status", status, "body", truncate(resp, 500))
		return
	}
	a.log.Info("better stack incident resolved", "key", key, "incident", id)
}

func (a *BetterStack) uptime(ctx context.Context, method, path string, body any) (int, []byte, error) {
	buf, err := json.Marshal(body)
	if err != nil {
		return 0, nil, err
	}
	req, err := http.NewRequestWithContext(ctx, method, a.uptimeAPI+path, bytes.NewReader(buf))
	if err != nil {
		return 0, nil, err
	}
	req.Header.Set("Authorization", "Bearer "+a.token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := a.client.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer resp.Body.Close()
	out, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<16))
	return resp.StatusCode, out, nil
}

func (a *BetterStack) postWebhook(ctx context.Context, payload map[string]any) {
	buf, err := json.Marshal(payload)
	if err != nil {
		a.log.Error("better stack webhook: marshal", "error", err)
		return
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, a.webhookURL, bytes.NewReader(buf))
	if err != nil {
		a.log.Error("better stack webhook: build request", "error", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := a.client.Do(req)
	if err != nil {
		a.log.Error("better stack webhook: send failed", "error", err)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		out, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<16))
		a.log.Error("better stack webhook: rejected", "status", resp.StatusCode, "body", truncate(out, 500))
		return
	}
	a.log.Info("better stack webhook delivered", "event", payload["event"], "key", payload["key"])
}

func truncate(b []byte, n int) string {
	if len(b) <= n {
		return string(b)
	}
	return string(b[:n]) + "…"
}
