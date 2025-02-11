package main

import (
	"encoding/json"
	"html/template"
	"log"
	"log/slog"
	"logos-stories/internal/admin"
	"logos-stories/internal/logging"
	"logos-stories/internal/pkg/database"
	"logos-stories/internal/pkg/models"
	"logos-stories/internal/pkg/templates"
	"logos-stories/internal/stories"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/gorilla/mux"
)

func main() {
	logger := slog.New(logging.New(os.Stdout, &logging.Options{
		Level:     slog.LevelDebug,
		UseColors: true,
	}))

	// Initialize template engine
	templateEngine := templates.New("src/templates")
	templateEngine.AddFunc("json", func(v interface{}) template.JS {
		b, _ := json.Marshal(v)
		return template.JS(string(b))
	})

	// Initialize database
	dbPath := filepath.Join("data", "stories.db")
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
	// Setup static file serving
	r.PathPrefix("/static/").Handler(http.StripPrefix("/static/",
		http.FileServer(http.Dir("static"))))

	// Error handler
	r.NotFoundHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		logger.Warn("404", "path", r.URL.Path, "ip", r.RemoteAddr)
		// Load from 404.html
		http.ServeFile(w, r, "static/html/404.html")
	})

	// Other handlers
	adminHandler := admin.NewHandler(logger, templateEngine)
	adminHandler.RegisterRoutes(r)

	storiesHandler := stories.NewHandler(logger, templateEngine)
	storiesHandler.RegisterRoutes(r)

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
			start := time.Now()
			next.ServeHTTP(w, r)
			logger.Info("request completed",
				"method", r.Method,
				"path", r.URL.Path,
				"requester", r.RemoteAddr,
				"duration", time.Since(start))
		})
	}
}
