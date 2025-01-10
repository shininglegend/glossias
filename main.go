package main

import (
	"fmt"
	"log/slog"
	"logos-stories/internal/logging"
	"logos-stories/internal/stories"
	"net/http"
	"os"
)

func main() {
	// logging
	logger := slog.New(os.Stdout, &logging.Options{})

	// Define routes
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Hello, World!")
	})
	http.HandleFunc("/page1", stories.ServePage1)

	// Serve static files
	fs := http.FileServer(http.Dir("static"))
	http.Handle("/static/", http.StripPrefix("/static/", fs))

	// Start server
	fmt.Println("Server running on http://localhost:8080")
	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		logger.Error("Failed to start server", "error", err)
	}
}
