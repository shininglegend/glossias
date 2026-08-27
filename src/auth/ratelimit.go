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

// Rate limiting
var (
	rateLimiters     = make(map[string]*rate.Limiter)
	rateLimiterMutex sync.RWMutex
	tokensPerSecond  = 15

	// Every admin story editor page fetches the story's metadata to show its
	// title in the header, on top of the page's own requests. That read is
	// cheap and cached, so it doesn't count against the per-IP budget.
	rateLimitExempt = regexp.MustCompile(`^/api/admin/stories/\d+/metadata$`)
)

func getRateLimiter(ip string) *rate.Limiter {
	rateLimiterMutex.RLock()
	limiter, exists := rateLimiters[ip]
	rateLimiterMutex.RUnlock()

	if !exists {
		rateLimiterMutex.Lock()
		limiter = rate.NewLimiter(rate.Every(time.Second), tokensPerSecond) // burst of tokensPerSecond, refilling 1/sec
		rateLimiters[ip] = limiter
		rateLimiterMutex.Unlock()
	}

	return limiter
}

// RateLimitMiddleware returns a middleware that rate limits requests by IP
func RateLimitMiddleware(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodGet && rateLimitExempt.MatchString(r.URL.Path) {
				next.ServeHTTP(w, r)
				return
			}

			clientIP := r.Header.Get("X-Forwarded-For")
			if clientIP == "" {
				clientIP = r.RemoteAddr
			}

			// Remove port if present
			if host, _, err := net.SplitHostPort(clientIP); err == nil {
				clientIP = host
			}

			limiter := getRateLimiter(clientIP)

			if !limiter.Allow() {
				logger.Warn("rate limit exceeded", "ip", clientIP, "path", r.URL.Path)
				http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
