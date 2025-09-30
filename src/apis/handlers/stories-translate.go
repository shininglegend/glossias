package handlers

import (
	"context"
	"encoding/json"
	"glossias/src/apis/types"
	"glossias/src/auth"
	"glossias/src/pkg/models"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
)

// PostTranslateData returns JSON data for story page 4 (translation) for selected lines
func (h *Handler) GetTranslateData(w http.ResponseWriter, r *http.Request) {
	storyID := mux.Vars(r)["id"]
	id, err := strconv.Atoi(storyID)
	if err != nil {
		h.sendError(w, "Invalid story ID format", http.StatusBadRequest)
		return
	}

	// Parse line numbers from query parameters
	lineNumbersStr := r.URL.Query().Get("lines")
	if lineNumbersStr == "" {
		h.sendError(w, "Line numbers required", http.StatusBadRequest)
		return
	}

	var lineNumbers []int
	err = json.Unmarshal([]byte(lineNumbersStr), &lineNumbers)
	if err != nil {
		h.sendError(w, "Invalid line numbers format", http.StatusBadRequest)
		return
	}

	// Validate line numbers
	if len(lineNumbers) == 0 || len(lineNumbers) > 5 {
		h.sendError(w, "Must specify 1-5 line numbers", http.StatusBadRequest)
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

	// Convert 1-indexed to 0-indexed and validate bounds
	zeroIndexedLines := make([]int, len(lineNumbers))
	for i, lineNum := range lineNumbers {
		if lineNum < 1 || lineNum > len(story.Content.Lines) {
			h.sendError(w, "Line number out of bounds", http.StatusBadRequest)
			return
		}
		zeroIndexedLines[i] = lineNum - 1
	}

	lines, err := h.processLinesForTranslation(r.Context(), *story, id, zeroIndexedLines)
	if err != nil {
		h.log.Error("Failed to process lines for translation", "error", err)
		h.sendError(w, "Failed to process lines for translation", http.StatusInternalServerError)
		return
	}

	data := types.TranslationPageData{
		PageData: types.PageData{
			StoryID:    storyID,
			StoryTitle: story.Metadata.Title["en"],
			Language:   story.Metadata.Language,
		},
		Lines: lines,
	}

	response := types.APIResponse{
		Success: true,
		Data:    data,
	}

	json.NewEncoder(w).Encode(response)
}

// processLinesForTranslation prepares lines for translation page
func (h *Handler) processLinesForTranslation(ctx context.Context, story models.Story, id int, linesToTranslate []int) ([]types.LineTranslation, error) {
	lines := make([]types.LineTranslation, 0, len(linesToTranslate))

	translation, err := models.GetTranslationsByLanguage(ctx, id, "en")
	if err != nil {
		return nil, err
	}

	for _, lineIndex := range linesToTranslate {
		if lineIndex < len(story.Content.Lines) {
			dbLine := story.Content.Lines[lineIndex]
			lineTranslation := types.LineTranslation{
				LineText: types.LineText{
					Text: dbLine.Text,
				},
			}

			if translation != nil && lineIndex < len(translation) {
				lineTranslation.Translation = &translation[lineIndex].TranslationText
			}

			lines = append(lines, lineTranslation)
		}
	}

	return lines, nil
}
