package apis

import (
	"glossias/src/apis/handlers"
	"glossias/src/apis/users"
	"log/slog"

	"github.com/gorilla/mux"
)

// Handler is a wrapper that delegates to the handlers package
type Handler struct {
	*handlers.Handler
	users *users.Handler
}

// NewHandler creates a new API handler
func NewHandler(logger *slog.Logger) *Handler {
	return &Handler{
		Handler: handlers.NewHandler(logger),
		users:   users.NewHandler(logger),
	}
}

// RegisterRoutes registers all public story API routes under /api/stories
func (h *Handler) RegisterRoutes(router *mux.Router) {
	// Base is /api/stories
	storiesRouter := router.PathPrefix("/stories").Subrouter()
	h.Handler.RegisterRoutes(storiesRouter)
	h.users.RegisterRoutes(router)
}
