// logos-stories/src/admin/courses/crud.go
package courses

import (
	"encoding/json"
	"glossias/src/auth"
	"glossias/src/pkg/models"
	"net/http"
)

// Course represents a course with its metadata
type Course struct {
	CourseID     int32  `json:"course_id"`
	CourseNumber string `json:"course_number"`
	Name         string `json:"name"`
	Description  string `json:"description"`
}

// CreateCourseRequest represents the request body for creating a course
type CreateCourseRequest struct {
	CourseNumber string `json:"course_number"`
	Name         string `json:"name"`
	Description  string `json:"description"`
}

// UpdateCourseRequest represents the request body for updating a course
type UpdateCourseRequest struct {
	CourseNumber string `json:"course_number"`
	Name         string `json:"name"`
	Description  string `json:"description"`
}

// handleCoursesList returns all courses for super admins, or courses user is admin of for regular admins
func (h *Handler) handleCoursesList(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.GetUserID(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var courses []models.Course
	var err error

	if models.IsUserSuperAdmin(r.Context(), userID) {
		// Super admins can list all courses
		courses, err = models.ListAllCourses(r.Context())
	} else {
		// Regular admins can only see courses they admin
		courses, err = models.GetCoursesForUser(r.Context(), userID)
	}

	if err != nil {
		h.log.Error("failed to list courses", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(struct {
		Courses []models.Course `json:"courses"`
	}{courses})
}

// handleCourseCreate creates a new course (super admin only)
func (h *Handler) handleCourseCreate(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.GetUserID(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Only super admins can create courses
	if !models.IsUserSuperAdmin(r.Context(), userID) {
		http.Error(w, "Super admin access required", http.StatusForbidden)
		return
	}

	var req CreateCourseRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if req.CourseNumber == "" || req.Name == "" {
		http.Error(w, "Course number and name are required", http.StatusBadRequest)
		return
	}

	course, err := models.CreateCourse(r.Context(), req.CourseNumber, req.Name, req.Description)
	if err != nil {
		h.log.Error("failed to create course", "error", err, "course_number", req.CourseNumber)
		http.Error(w, "Failed to create course", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(course)
}

// handleCourseGet returns a specific course
func (h *Handler) handleCourseGet(w http.ResponseWriter, r *http.Request, courseID int32) {
	userID, ok := auth.GetUserID(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Check if user has access to this course
	if !auth.HasPermission(r.Context(), userID, courseID) {
		http.Error(w, "Access denied", http.StatusForbidden)
		return
	}

	course, err := models.GetCourse(r.Context(), courseID)
	if err != nil {
		if err == models.ErrNotFound {
			http.Error(w, "Course not found", http.StatusNotFound)
			return
		}
		h.log.Error("failed to get course", "error", err, "course_id", courseID)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(course)
}

// handleCourseUpdate updates a course (super admin only)
func (h *Handler) handleCourseUpdate(w http.ResponseWriter, r *http.Request, courseID int32) {
	userID, ok := auth.GetUserID(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Only super admins can update courses
	if !models.IsUserSuperAdmin(r.Context(), userID) {
		http.Error(w, "Super admin access required", http.StatusForbidden)
		return
	}

	var req UpdateCourseRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if req.CourseNumber == "" || req.Name == "" {
		http.Error(w, "Course number and name are required", http.StatusBadRequest)
		return
	}

	course, err := models.UpdateCourse(r.Context(), courseID, req.CourseNumber, req.Name, req.Description)
	if err != nil {
		if err == models.ErrNotFound {
			http.Error(w, "Course not found", http.StatusNotFound)
			return
		}
		h.log.Error("failed to update course", "error", err, "course_id", courseID)
		http.Error(w, "Failed to update course", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(course)
}

// handleCourseDelete deletes a course (super admin only)
func (h *Handler) handleCourseDelete(w http.ResponseWriter, r *http.Request, courseID int32) {
	userID, ok := auth.GetUserID(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Only super admins can delete courses
	if !models.IsUserSuperAdmin(r.Context(), userID) {
		http.Error(w, "Super admin access required", http.StatusForbidden)
		return
	}

	err := models.DeleteCourse(r.Context(), courseID)
	if err != nil {
		if err == models.ErrNotFound {
			http.Error(w, "Course not found", http.StatusNotFound)
			return
		}
		h.log.Error("failed to delete course", "error", err, "course_id", courseID)
		http.Error(w, "Failed to delete course", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
