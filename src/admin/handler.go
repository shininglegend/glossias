// glossias/src/admin/admin.go
package admin

import (
	"glossias/src/auth"
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
		// Get user ID from request context (set by auth middleware)
		userID, ok := auth.GetUserID(r)
		if !ok {
			h.log.Warn("admin access attempted without user ID", "path", r.URL.Path)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// Check if user is admin (super admin or course admin)
		if !auth.IsAdmin(r.Context(), userID) {
			h.log.Warn("admin access denied", "user_id", userID, "path", r.URL.Path)
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}

		next.ServeHTTP(w, r)
	})
}
