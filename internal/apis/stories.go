package apis

import (
	"encoding/json"
	"glossias/internal/apis/handlers"
	"log/slog"
	"net/http"

	"github.com/gorilla/mux"
)

// Handler is a wrapper that delegates to the handlers package
type Handler struct {
	*handlers.Handler
}

// NewHandler creates a new API handler
func NewHandler(logger *slog.Logger) *Handler {
	return &Handler{
		Handler: handlers.NewHandler(logger),
	}
}

// RegisterRoutes registers all public story API routes under /api/stories
func (h *Handler) RegisterRoutes(router *mux.Router) {
	// Base is /api/stories
	storiesRouter := router.PathPrefix("/stories").Subrouter()
	h.Handler.RegisterRoutes(storiesRouter)
	// Add a test hello route
	router.HandleFunc("/hello", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]string{"message": "Hello from the api!"})
	})
}
