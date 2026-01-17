package courses

import (
	"encoding/json"
	"glossias/src/apis/types"
	"glossias/src/auth"
	"glossias/src/pkg/models"
	"net/http"
	"slices"
	"strconv"

	"github.com/gorilla/mux"
)

func (h *Handler) studentPerformanceHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	storyIDStr := vars["id"]
	storyID, err := strconv.Atoi(storyIDStr)
	if err != nil {
		h.log.Error("Invalid story ID", "id", storyIDStr, "error", err)
		http.Error(w, "Invalid story ID", http.StatusBadRequest)
		return
	}

	// Get status filter from query params (active, future, past, or "" for all)
	status := r.URL.Query().Get("status")
	if !slices.Contains([]string{"", "active", "future", "past"}, status) {
		h.log.Error("Invalid status parameter", "status", status)
		http.Error(w, "Invalid status parameter. Must be: active, future, past, or empty", http.StatusBadRequest)
		return
	}

	userID, ok := auth.GetUserIDWithOk(r)
	if !ok {
		h.log.Warn("student performance access attempted without user ID")
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Get story course ID for permission check
	courseID, err := models.GetStoryCourseID(r.Context(), int32(storyID))
	if err != nil {
		h.log.Error("Failed to get story course ID", "error", err, "story_id", storyID)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Check if user is course admin for this specific course
	if !auth.IsCourseOrSuperAdmin(r.Context(), userID, courseID) {
		h.log.Warn("student performance access denied", "user_id", userID, "story_id", storyID, "course_id", courseID)
		http.Error(w, "Forbidden - course admin access required", http.StatusForbidden)
		return
	}

	// Get student performance data
	performanceData, err := models.GetStoryStudentPerformance(r.Context(), int32(storyID), status)
	if err != nil {
		h.log.Error("Failed to fetch story student performance", "error", err, "story_id", storyID)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	response := types.APIResponse{
		Success: true,
		Data:    performanceData,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		h.log.Error("Failed to encode response", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
}
