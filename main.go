package main

import (
	"context"
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
	"time"

	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"

	"github.com/clerk/clerk-sdk-go/v2"
	clerkhttp "github.com/clerk/clerk-sdk-go/v2/http"
)

func main() {
	// The level is a LevelVar so it can be set after .env is loaded: the
	// handler reads it on every record, so the logger can exist before we
	// know the configured level.
	var level slog.LevelVar
	level.Set(slog.LevelDebug) // default until LOG_LEVEL is read
	logger := slog.New(logging.New(os.Stdout, &logging.Options{
		Level:     &level,
		UseColors: true,
	}))

	// Load environment variables from .env file if present
	if err := godotenv.Load(); err != nil {
		logger.Warn("No .env file found, relying on environment variables")
	}
	// LOG_LEVEL: DEBUG (default), INFO, WARN, ERROR.
	if v := os.Getenv("LOG_LEVEL"); v != "" {
		if err := level.UnmarshalText([]byte(v)); err != nil {
			logger.Warn("invalid LOG_LEVEL, keeping DEBUG", "value", v)
		}
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
	r.Use(auth.RateLimitMiddleware(logger))
	r.Use(auth.Middleware(logger))
	r.Use(queryCountMiddleware())
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
	// AI grading of Produce submissions runs in the background and is optional:
	// without an API key submissions are stored ungraded.
	// The grader's system prompt is versioned in the database and edited on the
	// admin System page; make sure the first version is on record.
	if err := models.EnsureProduceGradingPrompt(context.Background()); err != nil {
		logger.Warn("Could not seed the Produce grading prompt; grading will use the built-in default", "error", err)
	}
	var produceGrading *models.ProduceGradingService
	if grader, ok := models.NewAnthropicGraderFromEnv(); ok {
		produceGrading = models.NewProduceGradingService(grader, logger)
		logger.Info("AI grading enabled", "model", models.GradingModel)
	} else {
		logger.Warn("ANTHROPIC_API_KEY not set; Produce submissions will not be AI-graded")
	}
	apiHandler := apis.NewHandler(logger, produceGrading)
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

	logger.Info("starting server", "addr", srv.Addr)
	if err := srv.ListenAndServe(); err != nil {
		logger.Error("server error", "error", err)
		os.Exit(1)
	}
}

// dbQueryWarnThreshold is the per-request DB call count above which a request
// is logged as suspicious. Legitimate pages need a handful of queries; well
// above that almost always means a model function is being called in a loop
// (N+1). Fix by adding a batch query (WHERE id = ANY($1)), not by raising this.
const dbQueryWarnThreshold = 15

// queryCountMiddleware attaches a DB query counter to every request so the
// models layer can record each call (see database.CountQuery). Must run
// before loggingMiddleware, which reads the total.
func queryCountMiddleware() mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx, _ := database.WithQueryCounter(r.Context())
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
				logger.Info("request completed",
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
