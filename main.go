package main

import (
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
	logger := slog.New(logging.New(os.Stdout, &logging.Options{
		Level:     slog.LevelDebug,
		UseColors: true,
	}))

	// Load environment variables from .env file if present
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	// Initialize database based on POSTGRES_DB environment variable
	// USE_POOL=true uses pgxpool, USE_POOL=false uses database/sql, no DATABASE_URL uses mock
	dbPath := "" // Not used for PostgreSQL
	db, err := database.InitDB(dbPath)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	// Set the DB for the models package
	models.SetDB(db.RawConn())

	// Clerk stuff
	clerk_key := os.Getenv("CLERK_SECRET_KEY")
	if clerk_key == "" {
		logger.Error("CLERK_SECRET_KEY environment variable not set")
		os.Exit(1)
	}
	clerk.SetKey(clerk_key)

	// All routing below here.
	r := mux.NewRouter()

	// Setup middleware if needed
	r.Use(auth.Middleware(logger))
	r.Use(loggingMiddleware(logger))

	// Initialize handlers
	// Serve static files (robots.txt, etc) that aren't handled by the frontend service
	r.PathPrefix("/static/").Handler(http.StripPrefix("/static/",
		http.FileServer(http.Dir("static"))))

	// Robots.txt
	r.HandleFunc("/robots.txt", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "static/robots.txt")
	})

	// Health check endpoint (no auth required)
	r.HandleFunc("/api/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status": "healthy"}`))
	}).Methods("GET", "OPTIONS")

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
			// It's not acually needed, but can cause problems if missing.
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

func loggingMiddleware(logger *slog.Logger) mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Wrap ResponseWriter to capture status code
			ww := &responseWriter{ResponseWriter: w, status: 200}
			next.ServeHTTP(ww, r)
			logger.Info("request completed",
				"method", r.Method,
				"path", r.URL.Path,
				"status", ww.status,
				"requester", r.RemoteAddr)
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
