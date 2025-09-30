package apis

import (
	"context"
	"glossias/src/auth"
	"glossias/src/pkg/models"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
)

type contextKey string

const TrackingIDKey contextKey = "tracking_id"

var allowedHosts = map[string]bool{
	// "http://localhost:3000": true,
	"http://localhost:5173":    true,
	"https://glossias.org":     true,
	"https://www.glossias.org": true,
}

// CORSMiddleware adds CORS headers to API responses
func CORSMiddleware() mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")
			// Check if the origin is allowed
			if allowedHosts[origin] {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Set("Access-Control-Allow-Methods", "GET,POST,PUT,PATCH,DELETE,OPTIONS")
				reqHdrs := r.Header.Get("Access-Control-Request-Headers")
				if reqHdrs == "" {
					reqHdrs = "content-type,authorization"
				}
				w.Header().Set("Access-Control-Allow-Headers", reqHdrs)
				w.Header().Set("Access-Control-Allow-Credentials", "true")
				w.Header().Set("Access-Control-Max-Age", "600")
				w.Header().Set("Access-Control-Expose-Headers", "X-Tracking-ID")
			}

			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusOK)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// TimeTrackingMiddleware tracks API request starts
func TimeTrackingMiddleware(logger *slog.Logger) mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Get user ID from context (set by auth middleware)
			userID, ok := auth.GetUserIDWithOk(r)

			// Skip tracking if no user ID available
			if !ok || userID == "" {
				next.ServeHTTP(w, r)
				return
			}

			route := r.URL.Path
			storyID := models.ExtractStoryIDFromRoute(route)

			clientIP := r.Header.Get("X-Forwarded-For")
			if clientIP == "" {
				clientIP = r.RemoteAddr
			}

			// Start time tracking
			trackingID, err := models.StartTimeTracking(context.Background(), userID, route, storyID, clientIP)
			if err != nil {
				logger.Error("failed to create time entry", "error", err)
				next.ServeHTTP(w, r)
				return
			}

			// Add tracking ID to both header and context for response body
			w.Header().Set("X-Tracking-ID", strconv.Itoa(int(trackingID)))
			ctx := context.WithValue(r.Context(), TrackingIDKey, trackingID)
			r = r.WithContext(ctx)

			next.ServeHTTP(w, r)
		})
	}
}
