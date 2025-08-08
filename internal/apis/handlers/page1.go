package handlers

import (
	"encoding/json"
	"fmt"
	"glossias/internal/apis/types"
	"glossias/internal/pkg/models"
	"net/http"
	"os"
	"strconv"

	"github.com/gorilla/mux"
)

// GetPage1 returns JSON data for story page 1 (reading)
func (h *Handler) GetPage1(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	storyID := mux.Vars(r)["id"]
	id, err := strconv.Atoi(storyID)
	if err != nil {
		h.sendError(w, "Invalid story ID format", http.StatusBadRequest)
		return
	}

	story, err := models.GetStoryData(id)
	if err == models.ErrNotFound {
		h.sendError(w, "Story not found", http.StatusNotFound)
		return
	}
	if err != nil {
		h.log.Error("Failed to fetch story data", "error", err)
		h.sendError(w, "Failed to fetch story data", http.StatusInternalServerError)
		return
	}

	lines := h.processLinesForPage1(*story, id)

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
func (h *Handler) processLinesForPage1(story models.Story, id int) []types.Line {
	lines := make([]types.Line, 0, len(story.Content.Lines))

	folderPath := fmt.Sprintf(storiesDir+"stories_audio/%v_%v%v",
		story.Metadata.Description.Language,
		story.Metadata.WeekNumber,
		story.Metadata.DayLetter)
	audioDir, err := os.ReadDir(folderPath)

	for i, dbLine := range story.Content.Lines {
		var audioFile *string
		if err == nil && i < len(audioDir) {
			temp := fmt.Sprintf("/%v/%v", folderPath, audioDir[i].Name())
			audioFile = &temp
		}

		lines = append(lines, types.Line{
			Text:     []string{dbLine.Text},
			AudioURL: audioFile,
		})
	}

	return lines
}
