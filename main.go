package main

import (
	"glossias/src/admin"
	"glossias/src/apis"
	"glossias/src/logging"
	"glossias/src/pkg/database"
	"glossias/src/pkg/models"
	"log"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
)

func main() {
	logger := slog.New(logging.New(os.Stdout, &logging.Options{
		Level:     slog.LevelDebug,
		UseColors: true,
	}))

	// Initialize database based on POSTGRES_DB environment variable
	// USE_POOL=true uses pgxpool, USE_POOL=false uses database/sql, no DATABASE_URL uses mock
	dbPath := "" // Not used for PostgreSQL
	db, err := database.InitDB(dbPath)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	// Set the DB for the models package
	models.SetDB(db)

	r := mux.NewRouter()

	// Setup middleware if needed
	r.Use(loggingMiddleware(logger))

	// Initialize handlers
	// Serve static files (robots.txt, etc) that aren't handled by the frontend service
	r.PathPrefix("/static/").Handler(http.StripPrefix("/static/",
		http.FileServer(http.Dir("static"))))

	// Robots.txt
	r.HandleFunc("/robots.txt", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "static/robots.txt")
	})

	// API handlers
	apiHandler := apis.NewHandler(logger)
	apiRouter := r.PathPrefix("/api").Subrouter()
	apiRouter.Use(jsonMiddleware())
	// Mount public story API under /api/*
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
			next.ServeHTTP(w, r)
			logger.Info("request completed",
				"method", r.Method,
				"path", r.URL.Path,
				"requester", r.RemoteAddr)
		})
	}
}

func jsonMiddleware() mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			next.ServeHTTP(w, r)
		})
	}
}
