// glossias/src/admin/admin.go
package admin

import (
	"glossias/src/auth"
	"glossias/src/pkg/models"
	"log/slog"
	"net/http"
	"strconv"

	"glossias/src/admin/courses"
	"glossias/src/admin/stories"
	adminusers "glossias/src/admin/users"

	"github.com/gorilla/mux"
)

type Handler struct {
	log     *slog.Logger
	stories *stories.Handler
	courses *courses.Handler
	users   *adminusers.Handler
}

func NewHandler(log *slog.Logger) *Handler {
	return &Handler{
		log:     log,
		stories: stories.NewHandler(log),
		courses: courses.NewHandler(log),
		users:   adminusers.NewHandler(log),
	}
}

func (h *Handler) RegisterRoutes(r *mux.Router) {
	// Admin routes are now mounted by the caller under /api/admin
	// Apply admin-specific middleware at this level
	r.Use(h.adminAuthMiddleware)

	// Register all admin routes beneath the provided base router
	h.stories.RegisterRoutes(r)
	h.courses.RegisterRoutes(r)
	h.users.RegisterRoutes(r)

	// Cache management endpoint (super admin only)
	r.HandleFunc("/cache/clear", h.clearCache).Methods("POST")
}

func (h *Handler) adminAuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get user ID from request context (set by auth middleware)
		userID, ok := auth.GetUserIDWithOk(r)
		if !ok {
			h.log.Warn("admin access attempted without user ID", "path", r.URL.Path)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// Check if user is admin (super admin or course admin)
		if !auth.IsAnyAdmin(r.Context(), userID) {
			h.log.Warn("admin access denied", "user_id", userID, "path", r.URL.Path)
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}

		// For course-specific operations, check course access
		// Extract course_id from query parameters if present
		courseIDStr := r.URL.Query().Get("course_id")
		if courseIDStr != "" {
			courseID, err := strconv.Atoi(courseIDStr)
			if err != nil {
				h.log.Warn("invalid course_id parameter", "course_id", courseIDStr, "user_id", userID)
				http.Error(w, "Invalid course ID", http.StatusBadRequest)
				return
			}

			// Check if user has access to this specific course
			if !auth.IsCourseAdmin(r.Context(), userID, int32(courseID)) {
				h.log.Warn("course access denied", "user_id", userID, "course_id", courseID, "path", r.URL.Path)
				http.Error(w, "Course access forbidden", http.StatusForbidden)
				return
			}
		}

		next.ServeHTTP(w, r)
	})
}

func (h *Handler) clearCache(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.GetUserIDWithOk(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Restrict to super admins only
	if !models.IsUserSuperAdmin(r.Context(), userID) {
		h.log.Warn("cache clear denied - not super admin", "user_id", userID)
		http.Error(w, "Forbidden - super admin required", http.StatusForbidden)
		return
	}

	if err := models.ClearAllCache(); err != nil {
		h.log.Error("failed to clear cache", "error", err, "user_id", userID)
		http.Error(w, "Failed to clear cache", http.StatusInternalServerError)
		return
	}

	h.log.Info("cache cleared by admin", "user_id", userID)
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Cache cleared successfully"))
}
