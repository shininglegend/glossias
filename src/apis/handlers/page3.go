package handlers

import (
	"encoding/json"
	"fmt"
	"glossias/src/apis/types"
	"glossias/src/pkg/models"
	"net/http"
	"os"
	"slices"
	"strconv"

	"github.com/gorilla/mux"
)

// GetPage3 returns JSON data for story page 3 (grammar)
func (h *Handler) GetPage3(w http.ResponseWriter, r *http.Request) {
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

	lines := h.processLinesForPage3(*story, id)

	data := types.Page3Data{
		PageData: types.PageData{
			StoryID:    storyID,
			StoryTitle: story.Metadata.Title["en"],
			Lines:      lines,
		},
		GrammarPoint: story.Metadata.GrammarPoint,
	}

	response := types.APIResponse{
		Success: true,
		Data:    data,
	}

	json.NewEncoder(w).Encode(response)
}

// processLinesForPage3 prepares lines with grammar highlights
func (h *Handler) processLinesForPage3(story models.Story, id int) []types.Line {
	lines := make([]types.Line, len(story.Content.Lines))

	folderPath := fmt.Sprintf(storiesDir+"stories_audio/%v_%v%v",
		story.Metadata.Description.Language,
		story.Metadata.WeekNumber,
		story.Metadata.DayLetter)
	audioDir, err := os.ReadDir(folderPath)

	for i, line := range story.Content.Lines {
		series := []string{}
		runes := []rune(line.Text)
		lastEnd := 0

		// Sort grammar points by position
		slices.SortFunc(line.Grammar, func(a, b models.GrammarItem) int {
			if a.Position[0] < b.Position[0] {
				return -1
			}
			if a.Position[0] > b.Position[0] {
				return 1
			}
			return 0
		})

		for _, grammar := range line.Grammar {
			start := grammar.Position[0]
			if start >= lastEnd {
				series = append(series, string(runes[lastEnd:start]))
			}
			series = append(series, "%", grammar.Text, "&")
			lastEnd = grammar.Position[1]
		}

		if lastEnd < len(runes) {
			series = append(series, string(runes[lastEnd:]))
		}

		var audioFile *string
		if err == nil && i < len(audioDir) {
			temp := fmt.Sprintf("/%v/%v", folderPath, audioDir[i].Name())
			audioFile = &temp
		}

		lines[i] = types.Line{
			Text:              series,
			AudioURL:          audioFile,
			HasVocabOrGrammar: len(line.Grammar) > 0,
		}
	}

	return lines
}
