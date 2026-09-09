package stories

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"glossias/src/auth"
	"glossias/src/pkg/models"

	"github.com/gorilla/mux"
)

type linkCourseRequest struct {
	CourseID int `json:"courseId"`
}

func (h *Handler) storyCoursesHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.listStoryCourses(w, r)
	case http.MethodPost:
		h.linkStoryCourse(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *Handler) listStoryCourses(w http.ResponseWriter, r *http.Request) {
	storyID, ok := h.authorizeStoryLinkedAdmin(w, r)
	if !ok {
		return
	}

	courses, err := models.ListStoryCourses(r.Context(), int32(storyID))
	if err != nil {
		h.log.Error("failed to list story courses", "error", err, "storyID", storyID)
		writeJSONError(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"success": true, "courses": courses})
}

func (h *Handler) linkStoryCourse(w http.ResponseWriter, r *http.Request) {
	storyID, err := strconv.Atoi(mux.Vars(r)["id"])
	if err != nil {
		writeJSONError(w, "Invalid story ID", http.StatusBadRequest)
		return
	}

	userID, ok := auth.GetUserIDWithOk(r)
	if !ok {
		writeJSONError(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var req linkCourseRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.CourseID == 0 {
		writeJSONError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if !auth.IsCourseOrSuperAdmin(r.Context(), userID, int32(req.CourseID)) {
		writeJSONError(w, "Forbidden: not a course admin", http.StatusForbidden)
		return
	}
	if !models.CanUserAdminLinkedStory(r.Context(), userID, int32(storyID)) &&
		!models.CanUserEditStory(r.Context(), userID, int32(storyID)) {
		writeJSONError(w, "Forbidden: cannot link this story", http.StatusForbidden)
		return
	}

	if _, err := models.GetCourse(r.Context(), int32(req.CourseID)); err != nil {
		if errors.Is(err, models.ErrNotFound) {
			writeJSONError(w, "Course not found", http.StatusBadRequest)
			return
		}
		h.log.Error("failed to verify course", "error", err, "course_id", req.CourseID)
		writeJSONError(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	if err := models.LinkStoryCourse(r.Context(), int32(storyID), int32(req.CourseID)); err != nil {
		h.log.Error("failed to link story", "error", err, "storyID", storyID, "courseID", req.CourseID)
		writeJSONError(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"success": true})
}

func (h *Handler) unlinkStoryCourse(w http.ResponseWriter, r *http.Request) {
	storyID, err := strconv.Atoi(mux.Vars(r)["id"])
	if err != nil {
		writeJSONError(w, "Invalid story ID", http.StatusBadRequest)
		return
	}
	courseID, err := strconv.Atoi(mux.Vars(r)["courseId"])
	if err != nil {
		writeJSONError(w, "Invalid course ID", http.StatusBadRequest)
		return
	}

	userID, ok := auth.GetUserIDWithOk(r)
	if !ok {
		writeJSONError(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	if !auth.IsCourseOrSuperAdmin(r.Context(), userID, int32(courseID)) {
		writeJSONError(w, "Forbidden: not a course admin", http.StatusForbidden)
		return
	}

	if err := models.UnlinkStoryCourse(r.Context(), int32(storyID), int32(courseID)); err != nil {
		if errors.Is(err, models.ErrUnlinkOwner) {
			writeJSONError(w, "Cannot unlink the owner course", http.StatusBadRequest)
			return
		}
		h.log.Error("failed to unlink story", "error", err, "storyID", storyID, "courseID", courseID)
		writeJSONError(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"success": true})
}

// authorizeStoryLinkedAdmin allows the owner-course admin or any admin of a
// linked course (and super admins).
func (h *Handler) authorizeStoryLinkedAdmin(w http.ResponseWriter, r *http.Request) (int, bool) {
	storyID, err := strconv.Atoi(mux.Vars(r)["id"])
	if err != nil {
		writeJSONError(w, "Invalid story ID", http.StatusBadRequest)
		return 0, false
	}

	userID, ok := auth.GetUserIDWithOk(r)
	if !ok || !models.CanUserAdminLinkedStory(r.Context(), userID, int32(storyID)) {
		writeJSONError(w, "Unauthorized", http.StatusUnauthorized)
		return 0, false
	}

	return storyID, true
}
