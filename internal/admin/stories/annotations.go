// glossias/internal/admin/stories/annotations.go
package stories

import (
	"encoding/json"
	"glossias/internal/pkg/models"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
)

type AnnotationRequest struct {
	LineNumber int                    `json:"lineNumber"`
	Vocabulary *models.VocabularyItem `json:"vocabulary,omitempty"`
	Grammar    *models.GrammarItem    `json:"grammar,omitempty"`
	Footnote   *models.Footnote       `json:"footnote,omitempty"`
}

func (h *Handler) annotationsHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	storyID, err := strconv.Atoi(vars["id"])
	if err != nil {
		writeJSONError(w, "Invalid story ID", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case http.MethodGet:
		h.handleGetStoryContent(w, storyID)
	case http.MethodPut:
		h.handleUpdateAnnotations(w, r, storyID)
	case http.MethodDelete:
		h.handleClearAnnotations(w, storyID)
	default:
		writeJSONError(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *Handler) handleGetStoryContent(w http.ResponseWriter, storyID int) {
	story, err := models.GetStoryData(storyID)
	if err != nil {
		if err == models.ErrNotFound {
			writeJSONError(w, "Story not found", http.StatusNotFound)
			return
		}
		h.log.Error("Failed to fetch story data", "error", err)
		writeJSONError(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Return just the story content
	json.NewEncoder(w).Encode(map[string]interface{}{
		"content":  story.Content,
		"metadata": story.Metadata,
	})
}

func (h *Handler) handleUpdateAnnotations(w http.ResponseWriter, r *http.Request, storyID int) {
	var req AnnotationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Init a blank struct
	line := models.StoryLine{
		LineNumber: req.LineNumber,
	}

	// Add the appropriate annotation type
	switch {
	case req.Vocabulary != nil:
		line.Vocabulary = []models.VocabularyItem{*req.Vocabulary}
	case req.Grammar != nil:
		line.Grammar = []models.GrammarItem{*req.Grammar}
	case req.Footnote != nil:
		line.Footnotes = []models.Footnote{*req.Footnote}
	default:
		writeJSONError(w, "No annotation provided", http.StatusBadRequest)
		return
	}

	// Update in database (editline just adds)
	if err := models.AddLineAnnotations(storyID, req.LineNumber, line); err != nil {
		h.log.Error("Failed to update annotations",
			"error", err,
			"storyID", storyID,
			"lineNumber", req.LineNumber)
		writeJSONError(w, "Failed to update annotations", http.StatusInternalServerError)
		return
	}

	// Return success response
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]bool{"success": true})
}

func (h *Handler) handleClearAnnotations(w http.ResponseWriter, storyID int) {
	if err := models.ClearStoryAnnotations(storyID); err != nil {
		h.log.Error("Failed to clear annotations", "error", err, "storyID", storyID)
		writeJSONError(w, "Failed to clear annotations", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]bool{"success": true})
}

// Helper function for consistent error responses
func writeJSONError(w http.ResponseWriter, message string, status int) {
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": message})
}
