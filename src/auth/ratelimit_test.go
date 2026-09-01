package auth

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

func TestRateLimitMiddleware(t *testing.T) {
	resetRateLimiters()
	// Lower tokensPerSecond temporarily for easier testing limit hit
	rateLimiterMutex.Lock()
	oldTokensPerSecond := tokensPerSecond
	tokensPerSecond = 2 // 2 tokens per second
	rateLimiterMutex.Unlock()

	defer func() {
		rateLimiterMutex.Lock()
		tokensPerSecond = oldTokensPerSecond
		rateLimiterMutex.Unlock()
		resetRateLimiters()
	}()

	logger := slog.New(slog.DiscardHandler)
	middleware := RateLimitMiddleware(logger)

	// Create dummy handler
	dummyHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handlerToTest := middleware(dummyHandler)

	// Send 5 rapid requests from the same client IP
	clientIP := "192.168.1.100"
	okCount := 0
	limitCount := 0

	var wg sync.WaitGroup
	var mu sync.Mutex

	for range 5 {
		wg.Go(func() {
			req := httptest.NewRequest("GET", "/api/stories", nil)
			req.RemoteAddr = clientIP + ":1234"

			rr := httptest.NewRecorder()
			handlerToTest.ServeHTTP(rr, req)

			mu.Lock()
			if rr.Code == http.StatusOK {
				okCount++
			} else if rr.Code == http.StatusTooManyRequests {
				limitCount++
			} else {
				t.Errorf("unexpected status code: %d", rr.Code)
			}
			mu.Unlock()
		})
	}

	wg.Wait()

	// Since limit is 2 per second, at least some of the 5 concurrent requests should be rate limited
	if okCount > 2 {
		t.Errorf("expected at most 2 allowed requests, got %d", okCount)
	}
	if limitCount == 0 {
		t.Errorf("expected at least one request to be rate limited, got 0 limit hits")
	}
}

func TestRateLimiterEvictsIdleIPs(t *testing.T) {
	resetRateLimiters()
	defer resetRateLimiters()

	start := time.Date(2026, 8, 27, 12, 0, 0, 0, time.UTC)

	// Three clients seen at the start.
	for _, ip := range []string{"10.0.0.1", "10.0.0.2", "10.0.0.3"} {
		getRateLimiter(ip, start)
	}
	if got := rateLimiterCount(); got != 3 {
		t.Fatalf("tracked = %d, want 3", got)
	}

	// One client stays active just before the TTL; the others go quiet.
	active := start.Add(rateLimiterIdleTTL - time.Second)
	getRateLimiter("10.0.0.1", active)

	// Well past the TTL for the quiet ones, a new request triggers a sweep.
	later := start.Add(rateLimiterIdleTTL + rateLimiterSweepEvery + time.Second)
	getRateLimiter("10.0.0.9", later)

	if got := rateLimiterCount(); got != 2 {
		t.Errorf("tracked = %d, want 2 (the active client and the new one)", got)
	}
	rateLimiterMutex.Lock()
	_, hasActive := rateLimiters["10.0.0.1"]
	_, hasIdle := rateLimiters["10.0.0.2"]
	rateLimiterMutex.Unlock()
	if !hasActive {
		t.Error("recently active client should be kept")
	}
	if hasIdle {
		t.Error("idle client should have been evicted")
	}
}

func TestRateLimiterSweepIsThrottled(t *testing.T) {
	resetRateLimiters()
	defer resetRateLimiters()

	start := time.Date(2026, 8, 27, 12, 0, 0, 0, time.UTC)
	getRateLimiter("10.0.0.1", start)

	// Past the idle TTL but within a sweep interval of the last sweep: the
	// stale entry survives because we don't rescan the map on every request.
	rateLimiterMutex.Lock()
	rateLimiterLastSweep = start.Add(rateLimiterIdleTTL + time.Second)
	rateLimiterMutex.Unlock()
	getRateLimiter("10.0.0.2", start.Add(rateLimiterIdleTTL+2*time.Second))
	if got := rateLimiterCount(); got != 2 {
		t.Errorf("tracked = %d, want 2 (sweep throttled)", got)
	}
}

func TestRateLimiterKeepsLimiterForActiveIP(t *testing.T) {
	resetRateLimiters()
	defer resetRateLimiters()

	now := time.Now()
	first := getRateLimiter("10.0.0.1", now)
	second := getRateLimiter("10.0.0.1", now.Add(time.Second))
	if first != second {
		t.Error("an active IP should keep the same limiter (and its consumed tokens)")
	}
}

func TestClientIP(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "192.168.1.5:5555"
	if got := ClientIP(req); got != "192.168.1.5" {
		t.Errorf("ClientIP = %q, want RemoteAddr host", got)
	}
	req.Header.Set("X-Forwarded-For", "203.0.113.7")
	if got := ClientIP(req); got != "203.0.113.7" {
		t.Errorf("ClientIP = %q, want X-Forwarded-For", got)
	}
}
