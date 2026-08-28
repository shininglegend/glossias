package stories

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"glossias/src/pkg/models"

	"github.com/gorilla/mux"
)

// produceResponse is what the Produce editor loads: both ordered segments, the
// contrastive explanation, the story's grammar points to choose from, and the
// phase's readiness report.
type produceResponse struct {
	Segments      []models.ProduceSegment `json:"segments"`
	Explanation   string                  `json:"explanation"`
	GrammarPoints []models.GrammarPoint   `json:"grammarPoints"`
	Readiness     models.PhaseReadiness   `json:"readiness"`
	Required      int                     `json:"required"`
}

type produceSegmentRequest struct {
	EnglishText     string `json:"englishText"`
	ReferenceHebrew string `json:"referenceHebrew"`
	GrammarPointID  *int   `json:"grammarPointId,omitempty"`
	// LineStart and LineEnd place the segment in the story text (1-based,
	// inclusive). Both optional, but must be given together.
	LineStart *int `json:"lineStart,omitempty"`
	LineEnd   *int `json:"lineEnd,omitempty"`
}

type produceExplanationRequest struct {
	Explanation string `json:"explanation"`
}

func (h *Handler) produceHandler(w http.ResponseWriter, r *http.Request) {
	storyID, ok := h.authorizeStoryEdit(w, r)
	if !ok {
		return
	}

	if r.Method != http.MethodGet {
		writeJSONError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ctx := r.Context()

	segments, err := models.GetStoryProduceSegments(ctx, storyID)
	if err != nil {
		h.log.Error("Failed to fetch produce segments", "error", err, "storyID", storyID)
		writeJSONError(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	explanation, err := models.GetStoryProduceExplanation(ctx, storyID)
	if err != nil && !errors.Is(err, models.ErrNotFound) {
		h.log.Error("Failed to fetch produce explanation", "error", err, "storyID", storyID)
		writeJSONError(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	grammarPoints, err := models.GetStoryGrammarPoints(ctx, storyID)
	if err != nil {
		h.log.Error("Failed to fetch story grammar points", "error", err, "storyID", storyID)
		writeJSONError(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(produceResponse{
		Segments:      segments,
		Explanation:   explanation,
		GrammarPoints: grammarPoints,
		Readiness:     models.ValidateProduceContent(segments, explanation),
		Required:      models.ProduceSegmentsPerStory,
	})
}

// produceSegmentHandler upserts or removes the segment at a given order slot.
// Addressing by order rather than by row ID matches how the phase presents them
// and keeps the editor's two slots stable across saves.
func (h *Handler) produceSegmentHandler(w http.ResponseWriter, r *http.Request) {
	storyID, ok := h.authorizeStoryEdit(w, r)
	if !ok {
		return
	}

	order, err := strconv.Atoi(mux.Vars(r)["order"])
	if err != nil || order < 1 || order > models.ProduceSegmentsPerStory {
		writeJSONError(w, "Segment order must be between 1 and "+
			strconv.Itoa(models.ProduceSegmentsPerStory), http.StatusBadRequest)
		return
	}

	switch r.Method {
	case http.MethodPut:
		h.saveProduceSegment(w, r, storyID, order)
	case http.MethodDelete:
		h.deleteProduceSegment(w, r, storyID, order)
	default:
		writeJSONError(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *Handler) saveProduceSegment(w http.ResponseWriter, r *http.Request, storyID, order int) {
	ctx := r.Context()

	var req produceSegmentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	englishText := strings.TrimSpace(req.EnglishText)
	referenceHebrew := strings.TrimSpace(req.ReferenceHebrew)
	if englishText == "" {
		writeJSONError(w, "englishText is required", http.StatusBadRequest)
		return
	}
	if referenceHebrew == "" {
		writeJSONError(w, "referenceHebrew is required", http.StatusBadRequest)
		return
	}

	// A grammar point from another story would put the wrong context into the
	// grading prompt and the explanation popup.
	if req.GrammarPointID != nil {
		grammarPoint, err := models.GetGrammarPoint(ctx, *req.GrammarPointID)
		if errors.Is(err, models.ErrNotFound) {
			writeJSONError(w, "Grammar point not found", http.StatusBadRequest)
			return
		}
		if err != nil {
			h.log.Error("Failed to fetch grammar point", "error", err, "grammarPointID", *req.GrammarPointID)
			writeJSONError(w, "Internal server error", http.StatusInternalServerError)
			return
		}
		if grammarPoint.StoryID != storyID {
			writeJSONError(w, "Grammar point does not belong to this story", http.StatusBadRequest)
			return
		}
	}

	// The line range drives where the student page marks the segment's slot;
	// a line that doesn't exist would leave the marker nowhere.
	if (req.LineStart == nil) != (req.LineEnd == nil) {
		writeJSONError(w, "lineStart and lineEnd must be provided together", http.StatusBadRequest)
		return
	}
	if req.LineStart != nil {
		lineCount, err := models.CountStoryLines(ctx, storyID)
		if err != nil {
			h.log.Error("Failed to count story lines", "error", err, "storyID", storyID)
			writeJSONError(w, "Internal server error", http.StatusInternalServerError)
			return
		}
		if *req.LineStart < 1 || *req.LineEnd > lineCount || *req.LineStart > *req.LineEnd {
			writeJSONError(w, "lineStart/lineEnd must be a valid range between 1 and "+strconv.Itoa(lineCount), http.StatusBadRequest)
			return
		}
	}

	segment, err := models.UpsertProduceSegment(ctx, models.ProduceSegment{
		StoryID:         storyID,
		SegmentOrder:    order,
		EnglishText:     englishText,
		ReferenceHebrew: referenceHebrew,
		GrammarPointID:  req.GrammarPointID,
		LineStart:       req.LineStart,
		LineEnd:         req.LineEnd,
	})
	if err != nil {
		h.log.Error("Failed to save produce segment", "error", err, "storyID", storyID, "order", order)
		writeJSONError(w, "Failed to save segment", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(segment)
}

func (h *Handler) deleteProduceSegment(w http.ResponseWriter, r *http.Request, storyID, order int) {
	ctx := r.Context()

	segments, err := models.GetStoryProduceSegments(ctx, storyID)
	if err != nil {
		h.log.Error("Failed to fetch produce segments", "error", err, "storyID", storyID)
		writeJSONError(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	for _, segment := range segments {
		if segment.SegmentOrder != order {
			continue
		}
		// produce_submissions.segment_id cascades, so this discards any student
		// attempts at the segment along with it.
		if err := models.DeleteProduceSegment(ctx, storyID, segment.ID); err != nil {
			h.log.Error("Failed to delete produce segment", "error", err, "segmentID", segment.ID)
			writeJSONError(w, "Failed to delete segment", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)
		return
	}

	writeJSONError(w, "No segment at position "+strconv.Itoa(order), http.StatusNotFound)
}

func (h *Handler) produceExplanationHandler(w http.ResponseWriter, r *http.Request) {
	storyID, ok := h.authorizeStoryEdit(w, r)
	if !ok {
		return
	}

	ctx := r.Context()

	switch r.Method {
	case http.MethodPut:
		var req produceExplanationRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSONError(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		explanation := strings.TrimSpace(req.Explanation)
		if explanation == "" {
			writeJSONError(w, "explanation is required; use DELETE to remove it", http.StatusBadRequest)
			return
		}

		if err := models.UpsertStoryProduceExplanation(ctx, storyID, explanation); err != nil {
			h.log.Error("Failed to save produce explanation", "error", err, "storyID", storyID)
			writeJSONError(w, "Failed to save explanation", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"explanation": explanation})

	case http.MethodDelete:
		if err := models.DeleteStoryProduceExplanation(ctx, storyID); err != nil {
			h.log.Error("Failed to delete produce explanation", "error", err, "storyID", storyID)
			writeJSONError(w, "Failed to delete explanation", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)

	default:
		writeJSONError(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}
