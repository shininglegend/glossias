package main

import (
	"context"
	"fmt"
	"glossias/src/admin"
	"glossias/src/apis"
	"glossias/src/auth"
	"glossias/src/logging"
	"glossias/src/pkg/database"
	"glossias/src/pkg/models"
	"log"
	"log/slog"
	"net/http"
	"os"
	"runtime/debug"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"

	"github.com/clerk/clerk-sdk-go/v2"
	clerkhttp "github.com/clerk/clerk-sdk-go/v2/http"
)

func main() {
	// Use JSON logging in production for Betterstack, colored logs in dev
	useJSON := false // JSON LOGGING
	logger := slog.New(logging.New(os.Stdout, &logging.Options{
		Level:     slog.LevelDebug,
		UseColors: !useJSON,
		UseJSON:   useJSON,
	}))

	// Load environment variables from .env file if present
	err := godotenv.Load()
	if err != nil {
		slog.WarnContext(context.Background(), "No .env file found, relying on environment variables")
		err = nil
	}

	// Initialize database with automatic reconnection support
	// USE_POOL=true uses pgxpool, USE_POOL=false uses database/sql, no DATABASE_URL uses mock
	dbPath := "" // Not used for PostgreSQL
	db, err := database.InitDBWithReconnect(dbPath)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	// Set the DB for the models package
	models.SetDB(db.RawConn())
	// Set the storage client for the models package
	storageUrl := os.Getenv("STORAGE_URL")
	storageKey := os.Getenv("STORAGE_API_KEY")
	if storageUrl == "" || storageKey == "" {
		logger.Warn("STORAGE_URL or STORAGE_API_KEY environment variable not set, storage operations will fail")
	}
	models.SetStorageClient(storageUrl, storageKey)
	// Initialize cache
	if err := models.SetCache(); err != nil {
		logger.Error("Failed to initialize cache", "error", err)
		os.Exit(1)
	}

	// Clerk stuff
	clerk_key := os.Getenv("CLERK_SECRET_KEY")
	if clerk_key == "" {
		logger.Error("CLERK_SECRET_KEY environment variable not set. All auth will fail.")
	}
	clerk.SetKey(clerk_key)

	// All routing below here.
	r := mux.NewRouter()

	// Setup middleware if needed
	r.Use(requestIDMiddleware())
	r.Use(auth.RateLimitMiddleware(logger))
	r.Use(auth.Middleware(logger))
	r.Use(loggingMiddleware(logger))

	// Health check endpoint (no auth required)
	r.HandleFunc("/api/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status": "healthy"}`))
	}).Methods("GET", "OPTIONS")

	// Database health check endpoint (no auth required, rate-limited to 1 request per 5 minutes)
	r.HandleFunc("/api/db-health", apis.DBHealthHandler(logger)).Methods("GET", "OPTIONS")

	// Time tracking API (no auth required)
	timeTrackingHandler := apis.NewTimeTrackingHandler(logger)
	timeTrackingRouter := r.PathPrefix("/api").Subrouter()
	timeTrackingRouter.Use(jsonMiddleware())
	timeTrackingHandler.RegisterRoutes(timeTrackingRouter)

	// API handlers
	apiHandler := apis.NewHandler(logger)
	apiRouter := r.PathPrefix("/api").Subrouter()

	// Clerk: require Authorization: Bearer <token> on every request (unless dev auth bypass)
	authorizedParty := os.Getenv("AUTHORIZED_PARTY")
	devUser := os.Getenv("DEV_USER")

	// Skip Clerk middleware if DEV_USER is set
	if devUser == "" {
		if authorizedParty == "" {
			logger.Warn("AUTHORIZED_PARTY environment variable not set")
			// It's not actually needed, but can cause problems if missing.
			apiRouter.Use(clerkhttp.RequireHeaderAuthorization())
		} else {
			apiRouter.Use(clerkhttp.RequireHeaderAuthorization(
				clerkhttp.AuthorizedPartyMatches(authorizedParty),
			))
		}
	}
	apiRouter.Use(jsonMiddleware())
	apiHandler.RegisterRoutes(apiRouter)

	// Admin API mounted under /api/admin/*
	adminHandler := admin.NewHandler(logger)
	adminApiRouter := apiRouter.PathPrefix("/admin").Subrouter()
	adminHandler.RegisterRoutes(adminApiRouter)

	// Select correct port and start the server
	port := os.Getenv("PORT")
	if port == "" {
		logger.Error("PORT environment variable not set")
		os.Exit(1)
	}

	srv := &http.Server{
		Handler:      r,
		Addr:         ":" + port,
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}

	logger.Info("starting server", "addr", srv.Addr, "use_json_logging", useJSON)
	if err := srv.ListenAndServe(); err != nil {
		logger.Error("server error", "error", err)
		os.Exit(1)
	}
}

func requestIDMiddleware() mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requestID := uuid.New().String()
			ctx := context.WithValue(r.Context(), "request_id", requestID)
			w.Header().Set("X-Request-ID", requestID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func loggingMiddleware(logger *slog.Logger) mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			requestID := r.Context().Value("request_id").(string)
			userID := auth.GetUserID(r)

			// Wrap ResponseWriter to capture status code
			ww := &responseWriter{ResponseWriter: w, status: 200}

			// Recover from panics
			defer func() {
				if err := recover(); err != nil {
					logger.Error("PANIC RECOVERED", "error", fmt.Sprintf("%v", err), "error.stack_trace", string(debug.Stack()),
						"http.request_id", requestID, "http.method", r.Method, "http.url", r.URL.Path, "http.status_code", 500,
						"user.id", userID, "client.address", r.RemoteAddr, "client.user_agent", r.UserAgent())
					ww.WriteHeader(http.StatusInternalServerError)
					fmt.Fprintf(ww, `{"success":false,"error":"Internal Server Error, the developer has been notified."}`)
				}
			}()

			next.ServeHTTP(ww, r)

			// Skip health checks, minimal logging for success, detailed for errors
			if r.URL.Path != "/api/health" {
				if ww.status >= 500 {
					logger.Error("HTTP 5xx error", "http.method", r.Method, "http.url", r.URL.Path, "http.status_code", ww.status,
						"http.request_id", requestID, "duration_ms", time.Since(start).Milliseconds(), "user.id", userID,
						"client.address", r.RemoteAddr, "client.user_agent", r.UserAgent(),
						"http.query", r.URL.RawQuery, "http.host", r.Host)
				} else if ww.status >= 400 {
					logger.Warn("HTTP 4xx error", "http.method", r.Method, "http.url", r.URL.Path,
						"http.status_code", ww.status, "http.request_id", requestID, "http.query", r.URL.RawQuery,  "duration_ms", time.Since(start).Milliseconds(), "http.host", r.Host)
				} else {
					logger.Debug("HTTP request", "http.method", r.Method, "http.url", r.URL.Path, "http.status_code", ww.status, "duration_ms", time.Since(start).Milliseconds(), "http.host", r.Host)
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
