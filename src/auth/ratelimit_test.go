package auth

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"golang.org/x/time/rate"
)

func TestRateLimitMiddleware(t *testing.T) {
	// Reinitialize rateLimiters map for clean test isolation
	rateLimiterMutex.Lock()
	rateLimiters = make(map[string]*rate.Limiter)
	// Lower tokensPerSecond temporarily for easier testing limit hit
	oldTokensPerSecond := tokensPerSecond
	tokensPerSecond = 2 // 2 tokens per second
	rateLimiterMutex.Unlock()

	defer func() {
		rateLimiterMutex.Lock()
		tokensPerSecond = oldTokensPerSecond
		rateLimiters = make(map[string]*rate.Limiter)
		rateLimiterMutex.Unlock()
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

	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
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
		}()
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
