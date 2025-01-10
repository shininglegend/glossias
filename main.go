package main

import (
	"log/slog"
	"logos-stories/internal/logging"
	"logos-stories/internal/stories"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/mux"
)

func main() {
	logger := slog.New(logging.New(os.Stdout, &logging.Options{
		Level: slog.LevelDebug,
	}))

	r := mux.NewRouter()

	// Setup middleware if needed
	r.Use(loggingMiddleware(logger))

	// Initialize handlers
	// Setup static file serving
	r.PathPrefix("/static/").Handler(http.StripPrefix("/static/",
		http.FileServer(http.Dir("static"))))
		
	// Other handlers
	storiesHandler := stories.NewHandler(logger)
	storiesHandler.RegisterRoutes(r)

	srv := &http.Server{
		Handler:      r,
		Addr:         ":8080",
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
				"duration", time.Since(start))
		})
	}
}
