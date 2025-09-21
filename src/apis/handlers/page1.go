package handlers

import (
	"encoding/json"
	"glossias/src/apis/types"
	"glossias/src/pkg/models"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
)

// GetPage1 returns JSON data for story page 1 (reading)
func (h *Handler) GetPage1(w http.ResponseWriter, r *http.Request) {
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

	lines := h.processLinesForPage1(*story)

	data := types.PageData{
		StoryID:    storyID,
		StoryTitle: story.Metadata.Title["en"],
		Lines:      lines,
	}

	response := types.APIResponse{
		Success: true,
		Data:    data,
	}

	json.NewEncoder(w).Encode(response)
}

// processLinesForPage1 prepares lines with audio for reading page
func (h *Handler) processLinesForPage1(story models.Story) []types.Line {
	lines := make([]types.Line, 0, len(story.Content.Lines))

	for _, dbLine := range story.Content.Lines {
		// Convert audio files to API format
		audioFiles := make([]types.AudioFile, 0, len(dbLine.AudioFiles))
		for _, audio := range dbLine.AudioFiles {
			audioFiles = append(audioFiles, types.AudioFile{
				ID:         audio.ID,
				FilePath:   audio.FilePath,
				FileBucket: audio.FileBucket,
				Label:      audio.Label,
			})
		}

		lines = append(lines, types.Line{
			Text:               []string{dbLine.Text},
			AudioFiles:         audioFiles,
		})
	}

	return lines
}
