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

// PostTranslateData returns JSON data for story page 4 (translation) for selected lines
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

	lines, err := h.processLinesForTranslation(r.Context(), *story, id, nil)
	if err != nil {
		h.log.Error("Failed to process lines for translation", "error", err)
		h.sendError(w, "Failed to process lines for translation", http.StatusInternalServerError)
		return
	}

	data := types.TranslationPageData{
		PageData: types.PageData{
			StoryID:    storyID,
			StoryTitle: story.Metadata.Title["en"],
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
	lines := make([]types.LineTranslation, 0, len(story.Content.Lines))

	translation, err := models.GetTranslationsByLanguage(ctx, id, "en")
	if err != nil {
		return nil, err
	}

	for i, dbLine := range story.Content.Lines {
		if translation != nil && i < len(translation) {
			if slices.Contains(linesToTranslate, i) {
				lines = append(lines, types.LineTranslation{
					LineText: types.LineText{
						Text: dbLine.Text,
					},
					Translation: &translation[i].TranslationText,
				})
			}
		}
	}

	return lines, nil
}
