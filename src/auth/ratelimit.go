package auth

import (
	"log/slog"
	"net"
	"net/http"
	"regexp"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// Per-IP rate limiting.
//
// Limiters live in a map keyed by client IP. Entries record when they were
// last used and are swept out after rateLimiterIdleTTL so the map is bounded
// by the number of *recently active* clients rather than every IP ever seen
// (developer_review.md #2).
var (
	rateLimiters     = make(map[string]*rateLimiterEntry)
	rateLimiterMutex sync.Mutex
	tokensPerSecond  = 15

	// rateLimiterIdleTTL is how long an IP may be silent before its limiter is
	// dropped. A dropped limiter comes back full, which is what a fresh client
	// gets anyway, so eviction never tightens the limit.
	rateLimiterIdleTTL = 10 * time.Minute
	// rateLimiterSweepEvery bounds how often the map is scanned.
	rateLimiterSweepEvery = time.Minute
	rateLimiterLastSweep  time.Time

	// Every admin story editor page fetches the story's metadata (header title)
	// and content readiness (nav warnings) on top of the page's own requests.
	// Both are served from cache after the first hit, so they don't count
	// against the per-IP budget.
	rateLimitExempt = regexp.MustCompile(`^/api/admin/stories/\d+/(metadata|content-readiness)$`)
)

type rateLimiterEntry struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

// getRateLimiter returns the IP's limiter, creating it if needed, and sweeps
// idle entries opportunistically. The sweep is amortised over requests
// instead of a background goroutine so there is nothing to start or stop.
func getRateLimiter(ip string, now time.Time) *rate.Limiter {
	rateLimiterMutex.Lock()
	defer rateLimiterMutex.Unlock()

	if now.Sub(rateLimiterLastSweep) >= rateLimiterSweepEvery {
		rateLimiterLastSweep = now
		for key, entry := range rateLimiters {
			if now.Sub(entry.lastSeen) > rateLimiterIdleTTL {
				delete(rateLimiters, key)
			}
		}
	}

	entry, exists := rateLimiters[ip]
	if !exists {
		entry = &rateLimiterEntry{
			limiter: rate.NewLimiter(rate.Every(time.Second), tokensPerSecond), // burst of tokensPerSecond, refilling 1/sec
		}
		rateLimiters[ip] = entry
	}
	entry.lastSeen = now
	return entry.limiter
}

// rateLimiterCount reports tracked IPs, for tests.
func rateLimiterCount() int {
	rateLimiterMutex.Lock()
	defer rateLimiterMutex.Unlock()
	return len(rateLimiters)
}

// resetRateLimiters clears all state, for tests.
func resetRateLimiters() {
	rateLimiterMutex.Lock()
	defer rateLimiterMutex.Unlock()
	rateLimiters = make(map[string]*rateLimiterEntry)
	rateLimiterLastSweep = time.Time{}
}

// ClientIP extracts the caller's IP from X-Forwarded-For or RemoteAddr,
// without the port.
func ClientIP(r *http.Request) string {
	clientIP := r.Header.Get("X-Forwarded-For")
	if clientIP == "" {
		clientIP = r.RemoteAddr
	}
	if host, _, err := net.SplitHostPort(clientIP); err == nil {
		clientIP = host
	}
	return clientIP
}

// RateLimitMiddleware returns a middleware that rate limits requests by IP
func RateLimitMiddleware(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodGet && rateLimitExempt.MatchString(r.URL.Path) {
				next.ServeHTTP(w, r)
				return
			}

			clientIP := ClientIP(r)
			limiter := getRateLimiter(clientIP, time.Now())

			if !limiter.Allow() {
				logger.Warn("rate limit exceeded", "ip", clientIP, "path", r.URL.Path)
				http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
