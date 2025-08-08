package apis

import (
	"glossias/internal/apis/handlers"
	"log/slog"

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

// RegisterRoutes registers all API routes
func (h *Handler) RegisterRoutes(router *mux.Router) {
	h.Handler.RegisterRoutes(router)
}
