package apis

import (
	"context"
	"encoding/json"
	"log/slog"
	"net"
	"net/http"
	"sync"
	"time"

	"glossias/src/pkg/models"
)

// Rate limiter for DB health check (1 request per 5 minutes per IP)
var (
	dbHealthLimiters     = make(map[string]time.Time)
	dbHealthLimiterMutex sync.RWMutex
	dbHealthRateLimit    = 5 * time.Minute
)

func canMakeDBHealthRequest(ip string) bool {
	dbHealthLimiterMutex.Lock()
	defer dbHealthLimiterMutex.Unlock()

	lastRequest, exists := dbHealthLimiters[ip]
	if !exists || time.Since(lastRequest) >= dbHealthRateLimit {
		dbHealthLimiters[ip] = time.Now()
		return true
	}
	return false
}

// DBHealthHandler checks database connectivity
func DBHealthHandler(logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		clientIP := r.Header.Get("X-Forwarded-For")
		if clientIP == "" {
			clientIP = r.RemoteAddr
		}

		// Remove port if present
		if host, _, err := net.SplitHostPort(clientIP); err == nil {
			clientIP = host
		}

		// Check rate limit
		if !canMakeDBHealthRequest(clientIP) {
			logger.Warn("db health check rate limit exceeded", "ip", clientIP)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusTooManyRequests)
			json.NewEncoder(w).Encode(map[string]string{
				"status": "error",
				"error":  "Rate limit exceeded. Please wait 5 minutes between requests.",
			})
			return
		}

		// Test database connection
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		err := models.TestDBConnection(ctx)
		if err != nil {
			logger.Error("database health check failed", "error", err)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusServiceUnavailable)
			json.NewEncoder(w).Encode(map[string]string{
				"status": "unhealthy",
				"error":  "Database connection failed",
			})
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{
			"status": "healthy",
		})
	}
}
