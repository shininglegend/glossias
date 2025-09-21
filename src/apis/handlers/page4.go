package handlers

import (
	"encoding/json"
	"glossias/src/apis/types"
	"glossias/src/pkg/models"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
)

// GetPage4 returns JSON data for story page 4 (translation)
func (h *Handler) GetPage4(w http.ResponseWriter, r *http.Request) {
	storyID := mux.Vars(r)["id"]
	id, err := strconv.Atoi(storyID)
	if err != nil {
		h.sendError(w, "Invalid story ID format", http.StatusBadRequest)
		return
	}

	story, err := models.GetStoryData(r.Context(), id)
	if err == models.ErrNotFound {
		h.sendError(w, "Story not found", http.StatusNotFound)
		return
	}
	if err != nil {
		h.log.Error("Failed to fetch story data", "error", err)
		h.sendError(w, "Failed to fetch story data", http.StatusInternalServerError)
		return
	}

	lines := h.processLinesForPage4(*story, id)

	// TODO: Translation field not yet implemented in StoryMetadata
	// Return empty translation for now
	translation := ""

	data := types.Page4Data{
		PageData: types.PageData{
			StoryID:    storyID,
			StoryTitle: story.Metadata.Title["en"],
			Lines:      lines,
		},
		Translation: translation,
	}

	response := types.APIResponse{
		Success: true,
		Data:    data,
	}

	json.NewEncoder(w).Encode(response)
}

// processLinesForPage4 prepares lines for translation page (plain text with audio)
func (h *Handler) processLinesForPage4(story models.Story, id int) []types.Line {
	lines := make([]types.Line, 0, len(story.Content.Lines))

	for _, dbLine := range story.Content.Lines {
		lines = append(lines, types.Line{
			Text:       []string{dbLine.Text},
			AudioFiles: []types.AudioFile{}, // No audio files for page 4
		})
	}

	return lines
}
