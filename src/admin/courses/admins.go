// logos-stories/src/admin/courses/admins.go
package courses

import (
	"encoding/json"
	"glossias/src/auth"
	"glossias/src/pkg/models"
	"net/http"
)

// CourseAdmin represents a user assigned as admin to a course
type CourseAdmin struct {
	UserID     string `json:"user_id"`
	Email      string `json:"email"`
	Name       string `json:"name"`
	AssignedAt string `json:"assigned_at"`
}

// AddCourseAdminRequest represents the request body for adding a course admin
type AddCourseAdminRequest struct {
	UserID string `json:"user_id"`
}

// handleCourseAdminsList returns all admins for a specific course
func (h *Handler) handleCourseAdminsList(w http.ResponseWriter, r *http.Request, courseID int32) {
	userID, ok := auth.GetUserIDWithOk(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Check if user has access to this course
	if !auth.HasPermission(r.Context(), userID, courseID) {
		http.Error(w, "Access denied", http.StatusForbidden)
		return
	}

	admins, err := models.GetCourseAdmins(r.Context(), courseID)
	if err != nil {
		h.log.Error("failed to get course admins", "error", err, "course_id", courseID)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(struct {
		Admins []models.CourseAdmin `json:"admins"`
	}{admins})
}

// handleCourseAdminAdd adds a user as admin to a course (super admin only)
func (h *Handler) handleCourseAdminAdd(w http.ResponseWriter, r *http.Request, courseID int32) {
	userID, ok := auth.GetUserIDWithOk(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Only super admins can assign course admins
	if !models.IsUserSuperAdmin(r.Context(), userID) {
		http.Error(w, "Super admin access required", http.StatusForbidden)
		return
	}

	var req AddCourseAdminRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if req.UserID == "" {
		http.Error(w, "User ID is required", http.StatusBadRequest)
		return
	}

	// Check if user exists
	_, err := models.GetUser(r.Context(), req.UserID)
	if err != nil {
		if err == models.ErrNotFound {
			http.Error(w, "User not found", http.StatusNotFound)
			return
		}
		h.log.Error("failed to check user existence", "error", err, "user_id", req.UserID)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Check if course exists
	_, err = models.GetCourse(r.Context(), courseID)
	if err != nil {
		if err == models.ErrNotFound {
			http.Error(w, "Course not found", http.StatusNotFound)
			return
		}
		h.log.Error("failed to check course existence", "error", err, "course_id", courseID)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Add course admin assignment
	assignment, err := models.AddCourseAdmin(r.Context(), courseID, req.UserID)
	if err != nil {
		h.log.Error("failed to add course admin", "error", err, "course_id", courseID, "user_id", req.UserID)
		http.Error(w, "Failed to add course admin", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(assignment)
}

// handleCourseAdminRemove removes a user as admin from a course (super admin only)
func (h *Handler) handleCourseAdminRemove(w http.ResponseWriter, r *http.Request, courseID int32, targetUserID string) {
	userID, ok := auth.GetUserIDWithOk(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Only super admins can remove course admins
	if !models.IsUserSuperAdmin(r.Context(), userID) {
		http.Error(w, "Super admin access required", http.StatusForbidden)
		return
	}

	// Check if course exists
	_, err := models.GetCourse(r.Context(), courseID)
	if err != nil {
		if err == models.ErrNotFound {
			http.Error(w, "Course not found", http.StatusNotFound)
			return
		}
		h.log.Error("failed to check course existence", "error", err, "course_id", courseID)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Check if user is actually a course admin
	if !models.IsUserCourseAdmin(r.Context(), targetUserID, courseID) {
		http.Error(w, "User is not an admin of this course", http.StatusNotFound)
		return
	}

	// Remove course admin assignment
	err = models.RemoveCourseAdmin(r.Context(), courseID, targetUserID)
	if err != nil {
		h.log.Error("failed to remove course admin", "error", err, "course_id", courseID, "user_id", targetUserID)
		http.Error(w, "Failed to remove course admin", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
