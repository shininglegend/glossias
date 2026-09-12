// glossias/src/admin/stories/students.go
package stories

import (
	"encoding/json"
	"errors"
	"net/http"
	"slices"
	"strconv"

	"glossias/src/apis/types"
	"glossias/src/auth"
	"glossias/src/pkg/models"

	"github.com/gorilla/mux"
)

// storyStudentsHandler serves GET /api/admin/stories/{id}/students: one row per
// enrolled student with their performance on this story.
// ?course_id= is required and scopes the roster to that linked course.
// ?status=active|future|past filters by enrollment status; empty means all.
func (h *Handler) storyStudentsHandler(w http.ResponseWriter, r *http.Request) {
	storyID, err := strconv.Atoi(mux.Vars(r)["id"])
	if err != nil {
		writeJSONError(w, "Invalid story ID", http.StatusBadRequest)
		return
	}

	courseID, err := strconv.Atoi(r.URL.Query().Get("course_id"))
	if err != nil || courseID == 0 {
		writeJSONError(w, "course_id is required", http.StatusBadRequest)
		return
	}

	userID, ok := auth.GetUserIDWithOk(r)
	if !ok || !auth.IsCourseOrSuperAdmin(r.Context(), userID, int32(courseID)) {
		writeJSONError(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	linked, err := models.StoryLinkedToCourse(r.Context(), int32(storyID), int32(courseID))
	if err != nil {
		h.log.Error("Failed to check story course link", "error", err, "storyID", storyID, "courseID", courseID)
		writeJSONError(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	if !linked {
		writeJSONError(w, "Story is not linked to this course", http.StatusBadRequest)
		return
	}

	status := r.URL.Query().Get("status")
	if !slices.Contains([]string{"", "active", "future", "past"}, status) {
		writeJSONError(w, "Invalid status parameter. Must be: active, future, past, or empty", http.StatusBadRequest)
		return
	}

	performanceData, err := models.GetStoryStudentPerformanceForCourse(r.Context(), int32(storyID), int32(courseID), status, r.URL.Query().Get("section"))
	if err != nil {
		if errors.Is(err, models.ErrInvalidSectionFilter) {
			writeJSONError(w, "Invalid section parameter", http.StatusBadRequest)
			return
		}
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

	attemptNumber := 1
	if raw := r.URL.Query().Get("attempt"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n < 1 {
			writeJSONError(w, "Invalid attempt parameter", http.StatusBadRequest)
			return
		}
		attemptNumber = n
	}

	drilldown, err := models.GetStudentStoryDrilldown(r.Context(), int32(storyID), studentID, attemptNumber)
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

// studentAttemptsHandler serves GET /api/admin/stories/{id}/students/{userId}/attempts:
// every archived attempt with its frozen score, then the live current attempt
// with the phases that hold data. Backs the instructor's Manage Attempts dialog.
func (h *Handler) studentAttemptsHandler(w http.ResponseWriter, r *http.Request) {
	storyID, ok := h.authorizeStoryEdit(w, r)
	if !ok {
		return
	}
	studentID := mux.Vars(r)["userId"]
	if studentID == "" {
		writeJSONError(w, "Missing student ID", http.StatusBadRequest)
		return
	}

	header, err := models.GetStudentStoryHeader(r.Context(), int32(storyID), studentID)
	if err != nil {
		if errors.Is(err, models.ErrNotFound) {
			writeJSONError(w, "Student not found", http.StatusNotFound)
			return
		}
		h.log.Error("Failed to load student story header", "error", err, "storyID", storyID, "studentID", studentID)
		writeJSONError(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	attempts, err := models.ListUserStoryAttemptSummaries(r.Context(), studentID, int32(storyID), header.StoryTitle)
	if err != nil {
		h.log.Error("Failed to list student attempts", "error", err, "storyID", storyID, "studentID", studentID)
		writeJSONError(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(types.APIResponse{Success: true, Data: attempts})
}

// deleteStudentAttemptHandler serves DELETE /api/admin/stories/{id}/students/{userId}/attempts/{attempt}:
// drops one archived attempt (score snapshot included) and renumbers the
// later ones so the sequence stays contiguous. Archived attempts are
// all-or-nothing; the live attempt is trimmed phase by phase through
// /progress instead.
func (h *Handler) deleteStudentAttemptHandler(w http.ResponseWriter, r *http.Request) {
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
	attempt, err := strconv.Atoi(mux.Vars(r)["attempt"])
	if err != nil || attempt < 1 {
		writeJSONError(w, "Invalid attempt number", http.StatusBadRequest)
		return
	}

	if err := models.DeleteArchivedAttempt(r.Context(), studentID, int32(storyID), attempt); err != nil {
		if errors.Is(err, models.ErrAttemptNotFound) {
			writeJSONError(w, "Attempt not found", http.StatusNotFound)
			return
		}
		h.log.Error("Failed to delete student attempt", "error", err,
			"adminID", adminID, "studentID", studentID, "storyID", storyID, "attempt", attempt)
		writeJSONError(w, "Failed to delete attempt", http.StatusInternalServerError)
		return
	}

	// Audit trail: destructive action against student data.
	h.log.Info("Student attempt deleted",
		"adminID", adminID, "studentID", studentID, "storyID", storyID, "attempt", attempt)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(types.APIResponse{Success: true, Data: map[string]int{"attempt": attempt}})
}

// resetStudentProgressHandler serves
// DELETE /api/admin/stories/{id}/students/{userId}/progress?phase=all|exercises|video|identify|translate|produce|recall|vocab|grammar
// and wipes that student's answers, submissions and time rows for the story
// so they can redo it. Live rows are the student's current attempt, so every
// phase but "all" leaves archived attempts (the official score included)
// alone; "all" also deletes the archive. phase defaults to "all".
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
		writeJSONError(w, "Invalid phase parameter. Must be one of: all, exercises, video, identify, translate, produce, recall, vocab, grammar", http.StatusBadRequest)
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
