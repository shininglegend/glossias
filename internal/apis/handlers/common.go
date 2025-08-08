package handlers

import (
	"encoding/json"
	"glossias/internal/apis/types"
	"glossias/internal/pkg/models"
	"log/slog"
	"net/http"
)

const (
	storiesDir = "static/stories/"
	vocabBlank = "%"
)

// Handler contains shared dependencies for all API handlers
type Handler struct {
	log *slog.Logger
}

// NewHandler creates a new API handler with the given logger
func NewHandler(logger *slog.Logger) *Handler {
	return &Handler{
		log: logger,
	}
}

// sendError sends a standard error response
func (h *Handler) sendError(w http.ResponseWriter, message string, status int) {
	w.WriteHeader(status)
	response := types.APIResponse{
		Success: false,
		Error:   message,
	}
	json.NewEncoder(w).Encode(response)
}

// sendValidationError sends a validation error with expected answer counts
func (h *Handler) sendValidationError(w http.ResponseWriter, message string, expectedAnswers map[int]int) {
	w.WriteHeader(http.StatusBadRequest)
	response := types.APIResponse{
		Success: false,
		Error:   message,
		Data: types.LineValidationError{
			Message:         message,
			ExpectedAnswers: expectedAnswers,
		},
	}
	json.NewEncoder(w).Encode(response)
}

// sortVocab sorts vocabulary items by position
func (h *Handler) sortVocab(a, b models.VocabularyItem) int {
	if a.Position[0] < b.Position[0] {
		return -1
	}
	if a.Position[0] > b.Position[0] {
		return 1
	}
	return 0
}
