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

	// Initialize handlers
	storiesHandler := stories.NewHandler(logger)
	storiesHandler.RegisterRoutes(r)

	// Setup middleware if needed
	r.Use( /* your middleware */ )

	// Setup static file serving
	r.PathPrefix("/static/").Handler(http.StripPrefix("/static/",
		http.FileServer(http.Dir("static"))))
		// http://localhost:8080/static/stories/stories_audio/hb_9b/hb_9b-01.mp3
		//http://localhost:8080/static/stories/stories_audio/hb_9b-01.mp3

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
