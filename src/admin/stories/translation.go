package stories

import (
	"encoding/json"
	"net/http"
	"strconv"

	"glossias/src/pkg/models"

	"github.com/gorilla/mux"
)

// LineTranslationRequest represents the request to get or insert a line translation
type LineTranslationRequest struct {
	LineNumber   int    `json:"lineNumber"`
	LanguageCode string `json:"languageCode"`
	Translation  string `json:"translation,omitempty"`
}

// GetTranslationsByLanguageRequest represents the request to get translations by language
type GetTranslationsByLanguageRequest struct {
	StoryID      int    `json:"storyId"`
	LanguageCode string `json:"languageCode"`
}

// BulkTranslationUpdate represents a single translation update
type BulkTranslationUpdate struct {
	LineNumber  int    `json:"lineNumber"`
	Translation string `json:"translation"`
}

// BulkTranslationRequest represents the request to update multiple translations
type BulkTranslationRequest struct {
	LanguageCode string                  `json:"languageCode"`
	Translations []BulkTranslationUpdate `json:"translations"`
}

// translationsHandler handles GET/PUT/DELETE /stories/{id}/translations
func (h *Handler) translationsHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		h.getAllTranslations(w, r)
	case "PUT":
		h.bulkUpdateTranslations(w, r)
	case "DELETE":
		h.deleteStoryTranslations(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// lineTranslationHandler handles GET/PUT/DELETE /stories/{id}/translations/line
func (h *Handler) lineTranslationHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		h.getLineTranslation(w, r)
	case "PUT":
		h.upsertLineTranslation(w, r)
	case "DELETE":
		h.deleteLineTranslation(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// translationsByLanguageHandler handles GET /stories/{id}/translations/lang/{lang}
func (h *Handler) translationsByLanguageHandler(w http.ResponseWriter, r *http.Request) {
	h.getTranslationsByLanguage(w, r)
}

// getLineTranslation handles GET /stories/{id}/translations/line
func (h *Handler) getLineTranslation(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	storyID, err := strconv.Atoi(vars["id"])
	if err != nil {
		http.Error(w, "Invalid story ID", http.StatusBadRequest)
		return
	}

	lineNumber, err := strconv.Atoi(r.URL.Query().Get("line"))
	if err != nil {
		http.Error(w, "Invalid line number", http.StatusBadRequest)
		return
	}

	languageCode := r.URL.Query().Get("lang")
	if languageCode == "" {
		http.Error(w, "Language code required", http.StatusBadRequest)
		return
	}

	translation, err := models.GetLineTranslation(r.Context(), storyID, lineNumber, languageCode)
	if err != nil {
		if err == models.ErrNotFound {
			http.Error(w, "Translation not found", http.StatusNotFound)
			return
		}
		h.log.Error("Failed to get line translation", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]string{"translation": translation})
}

// upsertLineTranslation handles PUT /stories/{id}/translations/line
func (h *Handler) upsertLineTranslation(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	storyID, err := strconv.Atoi(vars["id"])
	if err != nil {
		http.Error(w, "Invalid story ID", http.StatusBadRequest)
		return
	}

	var req LineTranslationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	err = models.UpsertLineTranslation(r.Context(), storyID, req.LineNumber, req.LanguageCode, req.Translation)
	if err != nil {
		h.log.Error("Failed to upsert line translation", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]bool{"success": true})
}

// getAllTranslations handles GET /stories/{id}/translations
func (h *Handler) getAllTranslations(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	storyID, err := strconv.Atoi(vars["id"])
	if err != nil {
		http.Error(w, "Invalid story ID", http.StatusBadRequest)
		return
	}

	translations, err := models.GetAllTranslationsForStory(r.Context(), storyID)
	if err != nil {
		h.log.Error("Failed to get story translations", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(translations)
}

// getTranslationsByLanguage handles GET /stories/{id}/translations/lang/{lang}
func (h *Handler) getTranslationsByLanguage(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	storyID, err := strconv.Atoi(vars["id"])
	if err != nil {
		http.Error(w, "Invalid story ID", http.StatusBadRequest)
		return
	}

	languageCode := vars["lang"]
	if languageCode == "" {
		http.Error(w, "Language code required", http.StatusBadRequest)
		return
	}

	translations, err := models.GetTranslationsByLanguage(r.Context(), storyID, languageCode)
	if err != nil {
		h.log.Error("Failed to get translations by language", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(translations)
}

// deleteLineTranslation handles DELETE /stories/{id}/translations/line
func (h *Handler) deleteLineTranslation(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	storyID, err := strconv.Atoi(vars["id"])
	if err != nil {
		http.Error(w, "Invalid story ID", http.StatusBadRequest)
		return
	}

	lineNumber, err := strconv.Atoi(r.URL.Query().Get("line"))
	if err != nil {
		http.Error(w, "Invalid line number", http.StatusBadRequest)
		return
	}

	languageCode := r.URL.Query().Get("lang")
	if languageCode == "" {
		http.Error(w, "Language code required", http.StatusBadRequest)
		return
	}

	err = models.DeleteLineTranslation(r.Context(), storyID, lineNumber, languageCode)
	if err != nil {
		h.log.Error("Failed to delete line translation", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]bool{"success": true})
}

// deleteStoryTranslations handles DELETE /stories/{id}/translations
func (h *Handler) deleteStoryTranslations(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	storyID, err := strconv.Atoi(vars["id"])
	if err != nil {
		http.Error(w, "Invalid story ID", http.StatusBadRequest)
		return
	}

	err = models.DeleteStoryTranslations(r.Context(), storyID)
	if err != nil {
		h.log.Error("Failed to delete story translations", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]bool{"success": true})
}

// bulkUpdateTranslations handles PUT /stories/{id}/translations
func (h *Handler) bulkUpdateTranslations(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	storyID, err := strconv.Atoi(vars["id"])
	if err != nil {
		http.Error(w, "Invalid story ID", http.StatusBadRequest)
		return
	}

	var req BulkTranslationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Update each translation
	for _, translation := range req.Translations {
		err = models.UpsertLineTranslation(r.Context(), storyID, translation.LineNumber, req.LanguageCode, translation.Translation)
		if err != nil {
			h.log.Error("Failed to upsert line translation", "error", err, "storyID", storyID, "lineNumber", translation.LineNumber)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]any{
		"success": true,
		"updated": len(req.Translations),
	})
}
