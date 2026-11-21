// Package server builds the HTTP router shared by main and the integration
// tests, so the routes and middleware order the tests measure are the ones
// production serves.
package server

import (
	"log/slog"
	"net/http"

	"glossias/src/admin"
	"glossias/src/apis"
	"glossias/src/auth"
	"glossias/src/logging"
	"glossias/src/pkg/database"
	"glossias/src/pkg/models"

	"github.com/gorilla/mux"
)

// Config is what NewRouter needs from its caller.
type Config struct {
	Logger *slog.Logger
	// ProduceGrading may be nil to run without AI grading.
	ProduceGrading *models.ProduceGradingService
	// RateLimit runs first on every request. Nil skips it (tests).
	RateLimit mux.MiddlewareFunc
	// Auth identifies the user and sets auth.UserIDKey. main passes
	// auth.Middleware; tests pass a fake that reads a header.
	Auth mux.MiddlewareFunc
	// APIAuth is applied to the /api subrouter only (Clerk's
	// RequireHeaderAuthorization). Nil skips it (DEV_USER, tests).
	APIAuth mux.MiddlewareFunc
}

// NewRouter registers every route and middleware.
func NewRouter(cfg Config) *mux.Router {
	logger := cfg.Logger
	r := mux.NewRouter()

	if cfg.RateLimit != nil {
		r.Use(cfg.RateLimit)
	}
	r.Use(cfg.Auth)
	r.Use(QueryCountMiddleware())
	r.Use(loggingMiddleware(logger))

	// Health check endpoint (no auth required)
	r.HandleFunc("/api/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status": "healthy"}`))
	}).Methods("GET", "OPTIONS")

	// Database health check endpoint (no auth required, rate-limited to 1 request per 5 minutes)
	r.HandleFunc("/api/db-health", apis.DBHealthHandler(logger)).Methods("GET", "OPTIONS")

	apiHandler := apis.NewHandler(logger, cfg.ProduceGrading)
	r.HandleFunc(auth.StripeWebhookPath, apiHandler.BillingWebhook()).Methods("POST")

	// Time tracking API (no auth required)
	timeTrackingHandler := apis.NewTimeTrackingHandler(logger)
	timeTrackingRouter := r.PathPrefix("/api").Subrouter()
	timeTrackingRouter.Use(jsonMiddleware())
	timeTrackingHandler.RegisterRoutes(timeTrackingRouter)

	apiRouter := r.PathPrefix("/api").Subrouter()
	if cfg.APIAuth != nil {
		apiRouter.Use(cfg.APIAuth)
	}
	apiRouter.Use(jsonMiddleware())
	apiHandler.RegisterRoutes(apiRouter)

	// Admin API mounted under /api/admin/*
	adminHandler := admin.NewHandler(logger)
	adminApiRouter := apiRouter.PathPrefix("/admin").Subrouter()
	adminHandler.RegisterRoutes(adminApiRouter)

	return r
}

// dbQueryWarnThreshold is the per-request DB call count above which a request
// is logged as suspicious. Legitimate pages need a handful of queries; well
// above that almost always means a model function is being called in a loop
// (N+1). Fix by adding a batch query (WHERE id = ANY($1)), not by raising this.
const dbQueryWarnThreshold = 15

// QueryCountMiddleware attaches a DB query counter to every request so the
// models layer can record each call (see database.CountQuery). Must run
// before loggingMiddleware, which reads the total. A counter already on the
// context (a test measuring the whole request) is reused, not shadowed.
func QueryCountMiddleware() mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := database.EnsureQueryCounter(r.Context())
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func loggingMiddleware(logger *slog.Logger) mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Wrap ResponseWriter to capture status code
			ww := &responseWriter{ResponseWriter: w, status: 200}
			next.ServeHTTP(ww, r)
			if r.URL.Path != "/api/health" {
				dbQueries := database.QueryCount(r.Context())
				level, msg := logging.HTTPRequestLog(ww.status)
				logger.Log(r.Context(), level, msg,
					"method", r.Method,
					"path", r.URL.Path,
					"status", ww.status,
					"db_queries", dbQueries,
					"requester", r.RemoteAddr)
				if dbQueries > dbQueryWarnThreshold {
					logger.Warn("high DB query count (possible N+1)",
						"method", r.Method,
						"path", r.URL.Path,
						"db_queries", dbQueries,
						"threshold", dbQueryWarnThreshold)
				}
			}
		})
	}
}

type responseWriter struct {
	http.ResponseWriter
	status int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.status = code
	rw.ResponseWriter.WriteHeader(code)
}

func jsonMiddleware() mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			next.ServeHTTP(w, r)
		})
	}
}
