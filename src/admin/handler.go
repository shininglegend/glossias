// glossias/internal/admin/admin.go
package admin

import (
	"log/slog"
	"net/http"

	"glossias/src/admin/stories"

	"github.com/gorilla/mux"
)

type Handler struct {
	log     *slog.Logger
	stories *stories.Handler
}

func NewHandler(log *slog.Logger) *Handler {
	return &Handler{
		log:     log,
		stories: stories.NewHandler(log),
	}
}

func (h *Handler) RegisterRoutes(r *mux.Router) {
	// Admin routes are now mounted by the caller under /api/admin
	// Apply admin-specific middleware at this level
	r.Use(h.adminAuthMiddleware)

	// Register all admin story routes beneath the provided base router
	h.stories.RegisterRoutes(r)
}

func (h *Handler) adminAuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// TODO: Admin authentication logic here
		next.ServeHTTP(w, r)
	})
}
