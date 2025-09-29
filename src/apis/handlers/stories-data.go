package handlers

import (
	"encoding/json"
	"glossias/src/apis/types"
	"glossias/src/auth"
	"glossias/src/pkg/models"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
)

// GetStoryMetadata returns metadata for a specific story
func (h *Handler) GetStoryMetadata(w http.ResponseWriter, r *http.Request) {
	storyID := mux.Vars(r)["id"]
	id, err := strconv.Atoi(storyID)
	if err != nil {
		h.sendError(w, "Invalid story ID format", http.StatusBadRequest)
		return
	}

	// Get story data from database
	story, err := models.GetStoryData(r.Context(), id, auth.GetUserID(r))
	if err == models.ErrNotFound {
		h.sendError(w, "Story not found", http.StatusNotFound)
		return
	}
	if err != nil {
		h.log.Error("Failed to fetch story metadata", "error", err, "storyID", id)
		h.sendError(w, "Failed to fetch story metadata", http.StatusInternalServerError)
		return
	}

	response := types.APIResponse{
		Success: true,
		Data:    story.Metadata,
	}

	json.NewEncoder(w).Encode(response)
}
