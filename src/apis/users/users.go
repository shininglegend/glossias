package users

import (
	"encoding/json"
	"glossias/src/auth"
	"glossias/src/pkg/models"
	"log/slog"
	"net/http"

	"github.com/gorilla/mux"
)

type Handler struct {
	log *slog.Logger
}

func NewHandler(logger *slog.Logger) *Handler {
	return &Handler{
		log: logger,
	}
}

type UserResponse struct {
	UserID            string                    `json:"user_id"`
	Email             string                    `json:"email"`
	Name              string                    `json:"name"`
	IsSuperAdmin      bool                      `json:"is_super_admin"`
	CourseAdminRights []models.CourseAdminRight `json:"course_admin_rights"`
	EnrolledCourses   []models.UserCourse       `json:"enrolled_courses"`
}

type APIResponse struct {
	Success bool   `json:"success"`
	Data    any    `json:"data,omitempty"`
	Error   string `json:"error,omitempty"`
}

func (h *Handler) RegisterRoutes(router *mux.Router) {
	router.HandleFunc("/me", h.GetCurrentUser).Methods("GET", "OPTIONS")
}

func (h *Handler) GetCurrentUser(w http.ResponseWriter, r *http.Request) {
	// Get user ID from request context (set by auth middleware)
	userID, ok := auth.GetUserIDWithOk(r)
	if !ok {
		h.log.Warn("user info requested without user ID")
		h.sendError(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Get user from database
	user, err := models.GetUser(r.Context(), userID)
	if err != nil {
		h.log.Error("Failed to fetch user from database", "user_id", userID, "error", err)
		h.sendError(w, "Failed to fetch user information", http.StatusInternalServerError)
		return
	}

	// Get user's course admin rights
	courseRights, err := models.GetUserCourseAdminRights(r.Context(), userID)
	if err != nil {
		h.log.Error("Failed to fetch user course rights", "user_id", userID, "error", err)
		// Continue without course rights rather than failing completely
		courseRights = []models.CourseAdminRight{}
	}

	// Get user's enrolled courses
	enrolledCourses, err := models.GetCoursesForUser(r.Context(), userID)
	if err != nil {
		h.log.Error("Failed to fetch user enrolled courses", "user_id", userID, "error", err)
		// Continue without enrolled courses rather than failing completely
		enrolledCourses = []models.UserCourse{}
	}

	response := APIResponse{
		Success: true,
		Data: UserResponse{
			UserID:            user.UserID,
			Email:             user.Email,
			Name:              user.Name,
			IsSuperAdmin:      user.IsSuperAdmin,
			CourseAdminRights: courseRights,
			EnrolledCourses:   enrolledCourses,
		},
	}

	json.NewEncoder(w).Encode(response)
}

func (h *Handler) sendError(w http.ResponseWriter, message string, status int) {
	w.WriteHeader(status)
	response := APIResponse{
		Success: false,
		Error:   message,
	}
	json.NewEncoder(w).Encode(response)
}
