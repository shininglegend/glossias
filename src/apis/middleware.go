package apis

import (
	"net/http"

	"github.com/gorilla/mux"
)

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
