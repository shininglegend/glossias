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

var ErrMissingTranslation = "Translation not available. Contact the story author."

// GetTranslateData returns JSON data for story page 4 (translation) for selected lines
func (h *Handler) GetTranslateData(w http.ResponseWriter, r *http.Request) {
	storyID := mux.Vars(r)["id"]
	id, err := strconv.Atoi(storyID)
	if err != nil {
		h.sendError(w, "Invalid story ID format", http.StatusBadRequest)
		return
	}

	userID := auth.GetUserID(r)
	if userID == "" {
		h.sendError(w, "Authentication required", http.StatusUnauthorized)
		return
	}

	// Check if user already has a translation request for this story
	existingRequest, err := models.GetTranslationRequest(r.Context(), userID, id)
	if err != nil && err != models.ErrNotFound {
		h.log.Error("Failed to check existing translation request", "error", err)
		h.sendError(w, "Failed to process request", http.StatusInternalServerError)
		return
	}

	var lineNumbers []int
	if existingRequest != nil {
		// Use existing requested lines
		for _, line := range existingRequest.RequestedLines {
			lineNumbers = append(lineNumbers, int(line))
		}
	} else {
		// Parse line numbers from query parameters for new request
		lineNumbersStr := r.URL.Query().Get("lines")
		if lineNumbersStr == "" {
			h.sendError(w, "Line numbers required", http.StatusBadRequest)
			return
		}

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

		// Create translation request to limit future requests
		_, err = models.CreateTranslationRequest(r.Context(), userID, id, lineNumbers)
		if err != nil {
			h.log.Error("Failed to create translation request", "error", err)
			h.sendError(w, "Failed to process request", http.StatusInternalServerError)
			return
		}
	}

	story, err := models.GetStoryData(r.Context(), id, userID)
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

	lines, returnedLines, err := h.processLinesForTranslation(r.Context(), *story, id, zeroIndexedLines)
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
		Lines:         lines,
		ReturnedLines: returnedLines,
	}

	response := types.APIResponse{
		Success: true,
		Data:    data,
	}

	json.NewEncoder(w).Encode(response)
}

// processLinesForTranslation prepares lines for translation page
func (h *Handler) processLinesForTranslation(ctx context.Context, story models.Story, id int, linesToTranslate []int) ([]types.LineTranslation, []int, error) {
	lines := make([]types.LineTranslation, 0, len(linesToTranslate))
	returnedLines := make([]int, 0, len(linesToTranslate))

	translations, err := models.GetTranslationsByLanguage(ctx, id, "en")
	if err != nil {
		return nil, nil, err
	}

	// Create a map of line number to translation for efficient lookup
	translationMap := make(map[int32]string)
	for _, trans := range translations {
		translationMap[trans.LineNumber] = trans.TranslationText
	}

	for _, lineIndex := range linesToTranslate {
		if lineIndex < len(story.Content.Lines) {
			dbLine := story.Content.Lines[lineIndex]
			lineTranslation := types.LineTranslation{
				LineText: types.LineText{
					Text: dbLine.Text,
				},
				LineNumber: lineIndex + 1, // Convert to 1-indexed
			}

			// Look up translation by line number
			if translationText, exists := translationMap[int32(lineIndex+1)]; exists && translationText != "" {
				lineTranslation.Translation = &translationText
			} else {
				lineTranslation.Translation = &ErrMissingTranslation
			}

			lines = append(lines, lineTranslation)
			returnedLines = append(returnedLines, lineIndex+1) // Convert to 1-indexed
		}
	}

	return lines, returnedLines, nil
}
