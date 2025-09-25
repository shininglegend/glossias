package handlers

import (
	"encoding/json"
	"glossias/src/apis/types"
	"glossias/src/auth"
	"glossias/src/pkg/models"
	"net/http"
	"slices"
	"strconv"

	"github.com/gorilla/mux"
)

// GetGrammarPage returns JSON data for story page 3 (grammar)
func (h *Handler) GetGrammarPage(w http.ResponseWriter, r *http.Request) {
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

	lines := h.createGrammarAnnotatedLines(*story, id)

	data := types.GrammarPageData{
		PageData: types.PageData{
			StoryID:    storyID,
			StoryTitle: story.Metadata.Title["en"],
			Lines:      lines,
		},
	}

	response := types.APIResponse{
		Success: true,
		Data:    data,
	}

	json.NewEncoder(w).Encode(response)
}

// createGrammarAnnotatedLines prepares lines with grammar highlights
func (h *Handler) createGrammarAnnotatedLines(story models.Story, id int) []types.Line {
	lines := make([]types.Line, len(story.Content.Lines))

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

		lines[i] = types.Line{
			Text:              series,
			AudioFiles:        []types.AudioFile{}, // Empty for now - could be populated from models
			HasVocabOrGrammar: len(line.Grammar) > 0,
		}
	}

	return lines
}
