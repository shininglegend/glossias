package apis

import (
	"glossias/src/apis/handlers"
	"glossias/src/apis/users"
	"glossias/src/pkg/models"
	"log/slog"

	"github.com/gorilla/mux"
)

// Handler is a wrapper that delegates to the handlers package
type Handler struct {
	*handlers.Handler
	users  *users.Handler
	logger *slog.Logger
}

// NewHandler creates a new API handler. produceGrading may be nil to run
// without AI grading.
func NewHandler(logger *slog.Logger, produceGrading *models.ProduceGradingService) *Handler {
	return &Handler{
		Handler: handlers.NewHandler(logger, produceGrading),
		users:   users.NewHandler(logger),
		logger:  logger,
	}
}

// RegisterRoutes registers all public story API routes under /api/stories
func (h *Handler) RegisterRoutes(router *mux.Router) {
	// Base is /api/stories
	storiesRouter := router.PathPrefix("/stories").Subrouter()
	h.Handler.RegisterRoutes(storiesRouter)
	h.users.RegisterRoutes(router)
}
