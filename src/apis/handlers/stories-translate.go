package handlers

import (
	"context"
	"encoding/json"
	"glossias/src/apis/types"
	"glossias/src/auth"
	"glossias/src/pkg/models"
	"net/http"
	"slices"
	"strconv"

	"github.com/gorilla/mux"
)

var ErrMissingTranslation = "Translation not available. Contact the story author."

// GetTranslateData returns JSON data for story page 4 (translation) for selected lines
func (h *Handler) GetTranslateData(w http.ResponseWriter, r *http.Request) {
	// If GET, send all lines, otherwise, save request
	storyIDstr := mux.Vars(r)["id"]
	storyID, err := strconv.Atoi(storyIDstr)
	if err != nil {
		h.sendError(w, "Invalid story ID format", http.StatusBadRequest)
		return
	}

	userID := auth.GetUserID(r)
	if userID == "" {
		h.sendError(w, "Authentication required", http.StatusUnauthorized)
		return
	}

	switch r.Method {
	case http.MethodGet:
		h.getTranslationRequest(w, r, userID, storyID)
	case http.MethodPut:
		h.saveTranslationRequest(w, r, userID, storyID)
	case http.MethodOptions:
		// TODO: Is this needed?
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (h *Handler) getTranslationRequest(w http.ResponseWriter, r *http.Request, userID string, storyID int) {
	ctx := r.Context()
	story, err := models.GetStoryData(ctx, storyID, userID)
	if err == models.ErrNotFound {
		h.sendError(w, "Story not found", http.StatusNotFound)
		return
	}
	if err != nil {
		h.log.Error("Failed to fetch story data", "error", err)
		h.sendError(w, "Failed to fetch story data", http.StatusInternalServerError)
		return
	}

	lines, err := h.processLinesForTranslation(ctx, *story, storyID)
	if err != nil {
		h.log.Error("Failed to process lines for translation", "error", err)
		h.sendError(w, "Failed to process lines for translation", http.StatusInternalServerError)
		return
	}

	hasTranslated, err := models.TranslationRequestExists(ctx, userID, storyID)

	data := types.TranslationPageData{
		PageData: types.PageData{
			StoryID:    strconv.Itoa(storyID),
			StoryTitle: story.Metadata.Title["en"],
			Language:   story.Metadata.Language,
		},
		Lines:         lines,
		HasTranslated: hasTranslated,
	}

	response := types.APIResponse{
		Success: true,
		Data:    data,
	}

	json.NewEncoder(w).Encode(response)
}

// saveTranslationRequest saves which lines the student translated
func (h *Handler) saveTranslationRequest(w http.ResponseWriter, r *http.Request, userID string, storyID int) {
	ctx := r.Context()

	// Get the previous request, if any
	prevTranslationRequest, err := models.GetTranslationRequest(ctx, userID, storyID)
	if err != nil && err != models.ErrNotFound {
		h.log.Error("Failed to save translation request", "error", err)
		h.sendError(w, "Failed to save translation request", http.StatusInternalServerError)
		return
	}

	// Parse line numbers from query parameters for new request
	lineNumbersStr := r.URL.Query().Get("lines")

	lineNumbers := []int{}
	err = json.Unmarshal([]byte(lineNumbersStr), &lineNumbers)
	if err != nil {
		h.sendError(w, "Invalid line numbers format", http.StatusBadRequest)
		return
	}
	// Combine the two requests if present
	if prevTranslationRequest == nil {
		// Create translation request to show this has been done
		_, err = models.CreateTranslationRequest(ctx, userID, storyID, lineNumbers)
		if err != nil {
			h.log.Error("Failed to create translation request", "error", err)
			h.sendError(w, "Failed to create translation request", http.StatusInternalServerError)
			return
		}
	} else {
		// Update the previous req in the db
		prevLineNumbers := prevTranslationRequest.RequestedLines

		// Combine and deduplicate line numbers using a map
		lineSet := make(map[int]bool)
		for _, line := range prevLineNumbers {
			lineSet[int(line)] = true
		}
		for _, line := range lineNumbers {
			lineSet[line] = true
		}

		// Convert to sorted slice
		combinedLines := make([]int32, 0, len(lineSet))
		for line := range lineSet {
			combinedLines = append(combinedLines, int32(line))
		}
		slices.Sort(combinedLines)

		// Save in DB
		err := models.UpdateTranslationRequest(ctx, userID, storyID, combinedLines)
		if err != nil {
			h.log.Error("Failed to update translation request", "error", err)
			h.sendError(w, "Failed to update translation request", http.StatusInternalServerError)
			return
		}
	}
	w.WriteHeader(http.StatusOK)
}

// processLinesForTranslation prepares lines for translation page
func (h *Handler) processLinesForTranslation(ctx context.Context, story models.Story, id int) (lines []types.LineTranslation, err error) {
	lines = make([]types.LineTranslation, 0, len(story.Content.Lines))

	translations, err := models.GetTranslationsByLanguage(ctx, id, "en")
	if err != nil {
		return nil, err
	}

	// Create a map of line number to translation for efficient lookup
	translationMap := make(map[int32]string)
	for _, trans := range translations {
		translationMap[trans.LineNumber] = trans.TranslationText
	}

	for lineIndex, lineContent := range story.Content.Lines {
		if lineIndex < len(story.Content.Lines) {
			lineTranslation := types.LineTranslation{
				LineText: types.LineText{
					Text: lineContent.Text,
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
		}
	}

	return lines, nil
}
