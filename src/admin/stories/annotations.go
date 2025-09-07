// glossias/src/admin/stories/annotations.go
package stories

import (
	"encoding/json"
	"glossias/src/pkg/models"
	"net/http"
	"strconv"
	"strings"

	"github.com/gorilla/mux"
)

type AnnotationRequest struct {
	LineNumber int                    `json:"lineNumber"`
	Vocabulary *models.VocabularyItem `json:"vocabulary,omitempty"`
	Grammar    *models.GrammarItem    `json:"grammar,omitempty"`
	Footnote   *models.Footnote       `json:"footnote,omitempty"`
	// For editing existing annotations
	VocabularyPosition *[2]int `json:"vocabularyPosition,omitempty"` // Position of existing vocabulary item to edit
	GrammarPosition    *[2]int `json:"grammarPosition,omitempty"`    // Position of existing grammar item to edit
	FootnoteID         *int    `json:"footnoteId,omitempty"`         // ID of existing footnote to edit
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
		h.handleGetStoryAnnotations(w, r, storyID)
	case http.MethodPost:
		h.handleAddAnnotations(w, r, storyID)
	case http.MethodPut:
		h.handleEditAnnotations(w, r, storyID)
	case http.MethodDelete:
		h.handleClearAnnotations(w, r, storyID)
	default:
		writeJSONError(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *Handler) handleGetStoryAnnotations(w http.ResponseWriter, r *http.Request, storyID int) {
	ctx := r.Context()
	lineNumberParam := r.URL.Query().Get("line")

	if lineNumberParam != "" {
		// Get annotations for specific line
		lineNumber, err := strconv.Atoi(lineNumberParam)
		if err != nil {
			writeJSONError(w, "Invalid line number", http.StatusBadRequest)
			return
		}

		line, err := models.GetLineAnnotations(ctx, storyID, lineNumber)
		if err != nil {
			if err == models.ErrNotFound {
				writeJSONError(w, "Story not found", http.StatusNotFound)
				return
			}
			h.log.Error("Failed to fetch line annotations", "error", err, "storyID", storyID, "lineNumber", lineNumber)
			writeJSONError(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		json.NewEncoder(w).Encode(line)
		return
	}

	// Get all annotations for story
	annotations, err := models.GetStoryAnnotations(ctx, storyID)
	if err != nil {
		if err == models.ErrNotFound {
			writeJSONError(w, "Story not found", http.StatusNotFound)
			return
		}
		h.log.Error("Failed to fetch story annotations", "error", err, "storyID", storyID)
		writeJSONError(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(annotations)
}

func (h *Handler) handleAddAnnotations(w http.ResponseWriter, r *http.Request, storyID int) {
	ctx := r.Context()

	var req AnnotationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Get line text for validation
	lineText, err := models.GetLineText(ctx, storyID, req.LineNumber)
	if err != nil {
		if err == models.ErrInvalidLineNumber {
			writeJSONError(w, "Line not found", http.StatusNotFound)
			return
		}
		h.log.Error("Failed to get line text", "error", err, "storyID", storyID, "lineNumber", req.LineNumber)
		writeJSONError(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Init a blank struct
	line := models.StoryLine{
		LineNumber: req.LineNumber,
	}

	// Add the appropriate annotation type with validation
	switch {
	case req.Vocabulary != nil:
		// Validate that the word exists in the line text
		if !strings.Contains(strings.ToLower(lineText), strings.ToLower(req.Vocabulary.Word)) {
			writeJSONError(w, "Word not found in line text", http.StatusBadRequest)
			return
		}
		line.Vocabulary = []models.VocabularyItem{*req.Vocabulary}
	case req.Grammar != nil:
		// Validate that the grammar text exists in the line text
		if !strings.Contains(strings.ToLower(lineText), strings.ToLower(req.Grammar.Text)) {
			writeJSONError(w, "Grammar text not found in line text", http.StatusBadRequest)
			return
		}
		line.Grammar = []models.GrammarItem{*req.Grammar}
	case req.Footnote != nil:
		line.Footnotes = []models.Footnote{*req.Footnote}
	default:
		writeJSONError(w, "No annotation provided", http.StatusBadRequest)
		return
	}

	// Update in database (editline just adds)
	if err := models.AddLineAnnotations(ctx, storyID, req.LineNumber, line); err != nil {
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

func (h *Handler) handleEditAnnotations(w http.ResponseWriter, r *http.Request, storyID int) {
	ctx := r.Context()

	var req AnnotationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate that we have both the annotation data and the identifier
	switch {
	case req.Grammar != nil && req.GrammarPosition != nil:
		if err := models.UpdateGrammarAnnotation(ctx, storyID, req.LineNumber, *req.GrammarPosition, *req.Grammar); err != nil {
			h.log.Error("Failed to update grammar annotation", "error", err, "storyID", storyID, "lineNumber", req.LineNumber)
			writeJSONError(w, "Failed to update grammar annotation", http.StatusInternalServerError)
			return
		}
	case req.Footnote != nil && req.FootnoteID != nil:
		if err := models.UpdateFootnoteAnnotation(ctx, storyID, *req.FootnoteID, *req.Footnote); err != nil {
			h.log.Error("Failed to update footnote annotation", "error", err, "storyID", storyID, "footnoteID", *req.FootnoteID)
			writeJSONError(w, "Failed to update footnote annotation", http.StatusInternalServerError)
			return
		}
	case req.Vocabulary != nil && req.VocabularyPosition != nil:
		if err := models.UpdateVocabularyAnnotation(ctx, storyID, req.LineNumber, *req.VocabularyPosition, *req.Vocabulary); err != nil {
			h.log.Error("Failed to update vocabulary annotation", "error", err, "storyID", storyID, "lineNumber", req.LineNumber)
			writeJSONError(w, "Failed to update vocabulary annotation", http.StatusInternalServerError)
			return
		}
	case req.Vocabulary != nil:
		if err := models.UpdateVocabularyByWord(ctx, storyID, req.LineNumber, req.Vocabulary.Word, req.Vocabulary.LexicalForm); err != nil {
			h.log.Error("Failed to update vocabulary lexical form", "error", err, "storyID", storyID, "lineNumber", req.LineNumber, "word", req.Vocabulary.Word)
			writeJSONError(w, "Failed to update vocabulary lexical form", http.StatusInternalServerError)
			return
		}
	default:
		writeJSONError(w, "Missing annotation data or identifier for editing", http.StatusBadRequest)
		return
	}

	// Return success response
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]bool{"success": true})
}

func (h *Handler) handleClearAnnotations(w http.ResponseWriter, r *http.Request, storyID int) {
	ctx := r.Context()

	lineNumberParam := r.URL.Query().Get("line")

	if lineNumberParam != "" {
		// Clear annotations for specific line
		lineNumber, err := strconv.Atoi(lineNumberParam)
		if err != nil {
			writeJSONError(w, "Invalid line number", http.StatusBadRequest)
			return
		}

		if err := models.ClearLineAnnotations(ctx, storyID, lineNumber); err != nil {
			h.log.Error("Failed to clear line annotations", "error", err, "storyID", storyID, "lineNumber", lineNumber)
			writeJSONError(w, "Failed to clear line annotations", http.StatusInternalServerError)
			return
		}
	} else {
		// Clear all annotations for story
		if err := models.ClearStoryAnnotations(ctx, storyID); err != nil {
			h.log.Error("Failed to clear annotations", "error", err, "storyID", storyID)
			writeJSONError(w, "Failed to clear annotations", http.StatusInternalServerError)
			return
		}
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]bool{"success": true})
}

// Helper function for consistent error responses
func writeJSONError(w http.ResponseWriter, message string, status int) {
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": message})
}
