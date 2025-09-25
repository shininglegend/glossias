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

// GetTranslateData returns JSON data for story page 4 (translation)
func (h *Handler) GetTranslateData(w http.ResponseWriter, r *http.Request) {
	storyID := mux.Vars(r)["id"]
	id, err := strconv.Atoi(storyID)
	if err != nil {
		h.sendError(w, "Invalid story ID format", http.StatusBadRequest)
		return
	}

	story, err := models.GetStoryData(r.Context(), id, auth.GetUserID(r))
	if err == models.ErrNotFound {
		h.sendError(w, "Story not found", http.StatusNotFound)
		return
	}
	if err != nil {
		h.log.Error("Failed to fetch story data", "error", err)
		h.sendError(w, "Failed to fetch story data", http.StatusInternalServerError)
		return
	}

	lines := h.processLinesForTranslation(*story, id)

	// TODO: Translation field not yet implemented in StoryMetadata
	// Return empty translation for now
	translation := ""

	data := types.TranslationPageData{
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

// processLinesForTranslation prepares lines for translation page (plain text with audio)
func (h *Handler) processLinesForTranslation(story models.Story, id int) []types.Line {
	lines := make([]types.Line, 0, len(story.Content.Lines))

	for _, dbLine := range story.Content.Lines {
		lines = append(lines, types.Line{
			Text:       []string{dbLine.Text},
			AudioFiles: []types.AudioFile{}, // No audio files for page 4
		})
	}

	return lines
}
