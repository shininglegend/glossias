package courses

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"glossias/src/auth"
	"glossias/src/pkg/models"

	"github.com/gorilla/mux"
)

type attachSectionRequest struct {
	SectionCourseID int32 `json:"section_course_id"`
}

type assignSectionStudentsRequest struct {
	UserIDs         []string `json:"user_ids"`
	SectionCourseID *int32   `json:"section_course_id"`
}

func (h *Handler) listCourseSectionsHandler(w http.ResponseWriter, r *http.Request) {
	courseID, ok := h.parseCourseID(w, r)
	if !ok {
		return
	}
	userID, ok := auth.GetUserIDWithOk(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	if !models.IsUserCourseOrSuperAdmin(r.Context(), userID, courseID) {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	sections, err := models.ListCourseSections(r.Context(), courseID)
	if err != nil {
		h.log.Error("failed to list course sections", "error", err, "course_id", courseID)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	if sections == nil {
		sections = []models.CourseSection{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"sections": sections})
}

func (h *Handler) attachCourseSectionHandler(w http.ResponseWriter, r *http.Request) {
	parentID, ok := h.parseCourseID(w, r)
	if !ok {
		return
	}
	userID, ok := auth.GetUserIDWithOk(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	if !models.IsUserSuperAdmin(r.Context(), userID) {
		http.Error(w, "Super admin access required", http.StatusForbidden)
		return
	}

	var req attachSectionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.SectionCourseID == 0 {
		http.Error(w, "section_course_id is required", http.StatusBadRequest)
		return
	}
	if err := models.AttachCourseSection(r.Context(), parentID, req.SectionCourseID); err != nil {
		h.writeSectionError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"success": true})
}

func (h *Handler) detachCourseSectionHandler(w http.ResponseWriter, r *http.Request) {
	parentID, ok := h.parseCourseID(w, r)
	if !ok {
		return
	}
	sectionID, err := strconv.Atoi(mux.Vars(r)["sectionId"])
	if err != nil || sectionID == 0 {
		http.Error(w, "Invalid section ID", http.StatusBadRequest)
		return
	}
	userID, ok := auth.GetUserIDWithOk(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	if !models.IsUserSuperAdmin(r.Context(), userID) {
		http.Error(w, "Super admin access required", http.StatusForbidden)
		return
	}

	if err := models.DetachCourseSection(r.Context(), parentID, int32(sectionID)); err != nil {
		h.writeSectionError(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"success": true})
}

func (h *Handler) assignSectionStudentsHandler(w http.ResponseWriter, r *http.Request) {
	parentID, ok := h.parseCourseID(w, r)
	if !ok {
		return
	}
	userID, ok := auth.GetUserIDWithOk(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	if !models.IsUserCourseOrSuperAdmin(r.Context(), userID, parentID) {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	var req assignSectionStudentsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}
	if len(req.UserIDs) == 0 {
		http.Error(w, "user_ids is required", http.StatusBadRequest)
		return
	}

	var sectionID int32
	if req.SectionCourseID != nil {
		sectionID = *req.SectionCourseID
	}
	if err := models.AssignUsersToSection(r.Context(), parentID, sectionID, req.UserIDs); err != nil {
		h.writeSectionError(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"success": true})
}

func (h *Handler) parseCourseID(w http.ResponseWriter, r *http.Request) (int32, bool) {
	id, err := strconv.Atoi(mux.Vars(r)["id"])
	if err != nil || id == 0 {
		http.Error(w, "Invalid course ID", http.StatusBadRequest)
		return 0, false
	}
	return int32(id), true
}

func (h *Handler) writeSectionError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, models.ErrNotFound):
		http.Error(w, "Course not found", http.StatusNotFound)
	case errors.Is(err, models.ErrInvalidSection),
		errors.Is(err, models.ErrSectionCycle),
		errors.Is(err, models.ErrAlreadySection),
		errors.Is(err, models.ErrSectionNotChild):
		http.Error(w, err.Error(), http.StatusBadRequest)
	default:
		h.log.Error("course section error", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}
