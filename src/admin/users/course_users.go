package users

import (
	"encoding/json"
	"glossias/src/auth"
	"glossias/src/pkg/models"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
)

type Handler struct {
	log *slog.Logger
}

func NewHandler(log *slog.Logger) *Handler {
	return &Handler{
		log: log,
	}
}

func (h *Handler) RegisterRoutes(r *mux.Router) {
	// Base: /api/admin/course-users
	courseUsers := r.PathPrefix("/course-users").Subrouter()
	courseUsers.HandleFunc("/{courseId:[0-9]+}", h.GetUsersForCourse).Methods("GET")
	courseUsers.HandleFunc("/{courseId:[0-9]+}", h.AddUserToCourse).Methods("POST")
	courseUsers.HandleFunc("/{courseId:[0-9]+}/users/{userId}", h.RemoveUserFromCourse).Methods("DELETE")
}

type UserResponse struct {
	ID         string `json:"id"`
	Email      string `json:"email"`
	Name       string `json:"name"`
	Role       string `json:"role"`
	EnrolledAt string `json:"enrolled_at"`
}

type AddUserRequest struct {
	Email string `json:"email"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}

func (h *Handler) GetUsersForCourse(w http.ResponseWriter, r *http.Request) {
	userID := auth.GetUserID(r)
	if userID == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	vars := mux.Vars(r)
	courseIDStr := vars["courseId"]
	courseID, err := strconv.Atoi(courseIDStr)
	if err != nil {
		http.Error(w, "Invalid course ID", http.StatusBadRequest)
		return
	}

	ctx := r.Context()

	// Check if user is admin of this course or super admin
	if !models.IsUserCourseOrSuperAdmin(ctx, userID, int32(courseID)) {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	// Get users for course
	courseUsers, err := models.GetUsersForCourse(ctx, courseID)
	if err != nil {
		h.log.Error("failed to get users for course", "error", err, "course_id", courseID)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Convert to response format
	users := make([]UserResponse, len(courseUsers))
	for i, user := range courseUsers {
		role := "student"
		if models.IsUserSuperAdmin(r.Context(), user.UserID) {
			role = "super_admin"
		} else if models.IsUserOnlyCourseAdmin(r.Context(), user.UserID, int32(courseID)) {
			role = "course_admin"
		}

		users[i] = UserResponse{
			ID:         user.UserID,
			Email:      user.Email,
			Name:       user.Name,
			Role:       role,
			EnrolledAt: user.EnrolledAt.Format("2006-01-02T15:04:05Z07:00"),
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"users": users,
	})
}

func (h *Handler) AddUserToCourse(w http.ResponseWriter, r *http.Request) {
	userID := auth.GetUserID(r)
	if userID == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	vars := mux.Vars(r)
	courseIDStr := vars["courseId"]
	courseID, err := strconv.Atoi(courseIDStr)
	if err != nil {
		http.Error(w, "Invalid course ID", http.StatusBadRequest)
		return
	}

	ctx := r.Context()

	// Check if user is admin of this course or super admin
	if !models.IsUserCourseOrSuperAdmin(ctx, userID, int32(courseID)) {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	var req AddUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Email == "" {
		http.Error(w, "Email is required", http.StatusBadRequest)
		return
	}

	// Add user to course by email
	err = models.AddUserToCourseByEmail(ctx, req.Email, courseID)
	if err != nil {
		h.log.Error("failed to add user to course", "error", err, "email", req.Email, "course_id", courseID)

		// Check if it's a user not found error
		if err == models.ErrNotFound {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(ErrorResponse{Error: "User with this email not found"})
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(ErrorResponse{Error: "Failed to add user to course"})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"message": "User added to course successfully",
	})
}

func (h *Handler) RemoveUserFromCourse(w http.ResponseWriter, r *http.Request) {
	userID := auth.GetUserID(r)
	if userID == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	vars := mux.Vars(r)
	courseIDStr := vars["courseId"]
	courseID, err := strconv.Atoi(courseIDStr)
	if err != nil {
		http.Error(w, "Invalid course ID", http.StatusBadRequest)
		return
	}

	targetUserID := vars["userId"]
	if targetUserID == "" {
		http.Error(w, "User ID is required", http.StatusBadRequest)
		return
	}

	ctx := r.Context()

	// Check if user is admin of this course or super admin
	if !models.IsUserCourseOrSuperAdmin(ctx, userID, int32(courseID)) {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	// Remove user from course
	err = models.RemoveUserFromCourse(ctx, courseID, targetUserID)
	if err != nil {
		h.log.Error("failed to remove user from course", "error", err, "user_id", targetUserID, "course_id", courseID)

		// Check if it's a user not found error
		if err == models.ErrNotFound {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(ErrorResponse{Error: "User not found or not enrolled in this course"})
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(ErrorResponse{Error: "Failed to remove user from course"})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"message": "User removed from course successfully",
	})
}
