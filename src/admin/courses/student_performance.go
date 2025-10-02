package courses

import (
	"encoding/json"
	"glossias/src/apis/types"
	"glossias/src/auth"
	"glossias/src/pkg/models"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
)

func (h *Handler) studentPerformanceHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	courseIDStr := vars["id"]
	courseID, err := strconv.Atoi(courseIDStr)
	if err != nil {
		h.log.Error("Invalid course ID", "course_id", courseIDStr, "error", err)
		http.Error(w, "Invalid course ID", http.StatusBadRequest)
		return
	}

	userID, ok := auth.GetUserIDWithOk(r)
	if !ok {
		h.log.Warn("student performance access attempted without user ID")
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Check if user is course admin for this specific course
	if !auth.IsCourseOrSuperAdmin(r.Context(), userID, int32(courseID)) {
		h.log.Warn("student performance access denied", "user_id", userID, "course_id", courseID)
		http.Error(w, "Forbidden - course admin access required", http.StatusForbidden)
		return
	}

	// Get student performance data
	performanceData, err := models.GetCourseStudentPerformance(r.Context(), int32(courseID))
	if err != nil {
		h.log.Error("Failed to fetch course student performance", "error", err, "course_id", courseID)
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
