package auth

import (
	"log/slog"
	"net"
	"net/http"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// Rate limiting
var (
	rateLimiters     = make(map[string]*rate.Limiter)
	rateLimiterMutex sync.RWMutex
	tokensPerSecond = 15
)

func getRateLimiter(ip string) *rate.Limiter {
	rateLimiterMutex.RLock()
	limiter, exists := rateLimiters[ip]
	rateLimiterMutex.RUnlock()

	if !exists {
		rateLimiterMutex.Lock()
		limiter = rate.NewLimiter(rate.Every(time.Second), tokensPerSecond) // 10 requests per second
		rateLimiters[ip] = limiter
		rateLimiterMutex.Unlock()
	}

	return limiter
}

// RateLimitMiddleware returns a middleware that rate limits requests by IP
func RateLimitMiddleware(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
