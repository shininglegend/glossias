package handlers

import (
	"encoding/json"
	"errors"
	"glossias/src/apis/types"
	"glossias/src/auth"
	"glossias/src/pkg/models"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
)

// ResetOwnProgress serves DELETE /api/stories/{id}/progress. It freezes the
// current score as a completed attempt, then wipes live exercise answers so
// the student can redo Identify → Translate → Produce → Recall. Video time
// is left intact; the official (first) score stays on the Score page.
func (h *Handler) ResetOwnProgress(w http.ResponseWriter, r *http.Request) {
	storyIDStr := mux.Vars(r)["id"]
	storyID, err := strconv.Atoi(storyIDStr)
	if err != nil {
		h.sendError(w, "Invalid story ID format", http.StatusBadRequest)
		return
	}

	userID := auth.GetUserID(r)
	if userID == "" {
		h.sendError(w, "User not authenticated", http.StatusUnauthorized)
		return
	}

	story, err := models.GetStoryData(r.Context(), storyID, userID)
	if err == models.ErrNotFound {
		h.sendError(w, "Story not found", http.StatusNotFound)
		return
	}
	if err != nil {
		h.log.Error("Failed to fetch story data", "error", err, "storyID", storyID)
		h.sendError(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	result, err := models.StartStudentStoryRedo(r.Context(), userID, int32(storyID), story.Metadata.Title["en"])
	if errors.Is(err, models.ErrStoryNotComplete) {
		h.sendError(w, "Finish the story before starting a new attempt", http.StatusBadRequest)
		return
	}
	if err != nil {
		h.log.Error("Failed to reset own story progress", "error", err, "userID", userID, "storyID", storyID)
		h.sendError(w, "Failed to restart the story", http.StatusInternalServerError)
		return
	}

	h.log.Info("Student restarted story exercises", "userID", userID, "storyID", storyID, "deleted", result.Deleted)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(types.APIResponse{Success: true, Data: result})
}
