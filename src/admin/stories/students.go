// glossias/src/admin/stories/students.go
package stories

import (
	"encoding/json"
	"errors"
	"net/http"
	"slices"

	"glossias/src/apis/types"
	"glossias/src/auth"
	"glossias/src/pkg/models"

	"github.com/gorilla/mux"
)

// storyStudentsHandler serves GET /api/admin/stories/{id}/students: one row per
// enrolled student with their performance on this story.
// ?status=active|future|past filters by enrollment status; empty means all.
func (h *Handler) storyStudentsHandler(w http.ResponseWriter, r *http.Request) {
	storyID, ok := h.authorizeStoryEdit(w, r)
	if !ok {
		return
	}

	status := r.URL.Query().Get("status")
	if !slices.Contains([]string{"", "active", "future", "past"}, status) {
		writeJSONError(w, "Invalid status parameter. Must be: active, future, past, or empty", http.StatusBadRequest)
		return
	}

	performanceData, err := models.GetStoryStudentPerformance(r.Context(), int32(storyID), status)
	if err != nil {
		h.log.Error("Failed to fetch story student performance", "error", err, "storyID", storyID)
		writeJSONError(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(types.APIResponse{Success: true, Data: performanceData})
}

// storyStudentDrilldownHandler serves GET /api/admin/stories/{id}/students/{userId}:
// the actual answers and submissions behind one row of the performance table —
// every Identify pick, the requested translation lines, every Produce
// submission with its AI score and feedback, every Recall ordering attempt,
// and the per-phase time.
func (h *Handler) storyStudentDrilldownHandler(w http.ResponseWriter, r *http.Request) {
	storyID, ok := h.authorizeStoryEdit(w, r)
	if !ok {
		return
	}

	studentID := mux.Vars(r)["userId"]
	if studentID == "" {
		writeJSONError(w, "Missing student ID", http.StatusBadRequest)
		return
	}

	drilldown, err := models.GetStudentStoryDrilldown(r.Context(), int32(storyID), studentID)
	if err != nil {
		if errors.Is(err, models.ErrNotFound) {
			writeJSONError(w, "Student not found", http.StatusNotFound)
			return
		}
		h.log.Error("Failed to build student drill-down", "error", err, "storyID", storyID, "studentID", studentID)
		writeJSONError(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(types.APIResponse{Success: true, Data: drilldown})
}

// resetStudentProgressHandler serves
// DELETE /api/admin/stories/{id}/students/{userId}/progress?phase=all|video|identify|translate|produce|recall|vocab|grammar
// and wipes that student's answers, submissions and time rows for the story
// (or for one phase) so they can redo it. phase defaults to "all".
func (h *Handler) resetStudentProgressHandler(w http.ResponseWriter, r *http.Request) {
	storyID, ok := h.authorizeStoryEdit(w, r)
	if !ok {
		return
	}
	adminID, _ := auth.GetUserIDWithOk(r)

	studentID := mux.Vars(r)["userId"]
	if studentID == "" {
		writeJSONError(w, "Missing student ID", http.StatusBadRequest)
		return
	}

	phase := models.ResetPhase(r.URL.Query().Get("phase"))
	if phase == "" {
		phase = models.ResetAll
	}
	if !slices.Contains(models.ResetPhases, phase) {
		writeJSONError(w, "Invalid phase parameter. Must be one of: all, video, identify, translate, produce, recall, vocab, grammar", http.StatusBadRequest)
		return
	}

	result, err := models.ResetUserStoryProgress(r.Context(), studentID, int32(storyID), phase)
	if err != nil {
		if errors.Is(err, models.ErrInvalidResetPhase) {
			writeJSONError(w, err.Error(), http.StatusBadRequest)
			return
		}
		h.log.Error("Failed to reset student progress", "error", err,
			"adminID", adminID, "studentID", studentID, "storyID", storyID, "phase", phase)
		writeJSONError(w, "Failed to reset student progress", http.StatusInternalServerError)
		return
	}

	// Audit trail: destructive action against student data.
	h.log.Info("Student progress reset",
		"adminID", adminID, "studentID", studentID, "storyID", storyID, "phase", phase, "deleted", result.Deleted)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(types.APIResponse{Success: true, Data: result})
}
