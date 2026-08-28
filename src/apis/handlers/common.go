package handlers

import (
	"encoding/json"
	"glossias/src/apis/types"
	"glossias/src/pkg/models"
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
	// produceGrading grades Produce submissions in the background. Nil when
	// grading is disabled (no ANTHROPIC_API_KEY); submissions then stay
	// ungraded, which the rest of the app tolerates.
	produceGrading *models.ProduceGradingService
}

// NewHandler creates a new API handler with the given logger. produceGrading
// may be nil to disable AI grading.
func NewHandler(logger *slog.Logger, produceGrading *models.ProduceGradingService) *Handler {
	return &Handler{
		log:            logger,
		produceGrading: produceGrading,
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
