// logos-stories/internal/admin/stories/metadata.go
package stories

import (
	"encoding/json"
	"logos-stories/internal/pkg/models"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
)

func (h *Handler) metadataHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	storyID, err := strconv.Atoi(vars["id"])
	if err != nil {
		http.Error(w, "Invalid story ID", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case http.MethodGet:
		h.handleGetMetadata(w, r, storyID)
	case http.MethodPut:
		h.handleUpdateMetadata(w, r, storyID)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *Handler) handleGetMetadata(w http.ResponseWriter, r *http.Request, storyID int) {
	story, err := models.GetStoryData(storyID)
	if err != nil {
		if err == models.ErrNotFound {
			http.Error(w, "Story not found", http.StatusNotFound)
			return
		}
		h.log.Error("Failed to fetch story metadata", "error", err, "storyID", storyID)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// For regular GET requests, render the HTML template
	if !isJSONRequest(r) {
		if err := h.te.Render(w, "admin/metadataStory.html", story); err != nil {
			h.log.Error("Failed to render metadata template", "error", err)
			http.Error(w, "Failed to render page", http.StatusInternalServerError)
		}
		return
	}

	// For API requests, return JSON
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(EditStoryResponse{
		Story:   story,
		Success: true,
	})
}

func (h *Handler) handleUpdateMetadata(w http.ResponseWriter, r *http.Request, storyID int) {
	var metadata models.StoryMetadata
	if err := json.NewDecoder(r.Body).Decode(&metadata); err != nil {
		h.log.Debug("Invalid request body", "error", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if err := models.EditStoryMetadata(storyID, metadata); err != nil {
		h.log.Error("Failed to update metadata", "error", err, "storyID", storyID)
		http.Error(w, "Failed to update metadata", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"success": true})
}

func isJSONRequest(r *http.Request) bool {
	return r.Header.Get("Accept") == "application/json"
}
