package stories

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"glossias/src/pkg/models"

	"github.com/gorilla/mux"
)

// recallResponse is what the Recall editor loads: the five ordered sentences
// with signed image URLs, the story's target words to link them to, and the
// phase's readiness report.
type recallResponse struct {
	Sentences        []models.RecallSentence   `json:"sentences"`
	TargetVocabulary []models.TargetVocabulary `json:"targetVocabulary"`
	Readiness        models.PhaseReadiness     `json:"readiness"`
	Required         int                       `json:"required"`
}

type recallSentenceRequest struct {
	HebrewText    string `json:"hebrewText"`
	TargetVocabID *int   `json:"targetVocabId,omitempty"`
	// ImagePath is a path returned by the phase-asset upload endpoint. Omitting
	// it leaves the stored image alone; sending an empty string clears it and
	// deletes the stored file.
	ImagePath *string `json:"imagePath,omitempty"`
}

func (h *Handler) recallHandler(w http.ResponseWriter, r *http.Request) {
	storyID, ok := h.authorizeStoryEdit(w, r)
	if !ok {
		return
	}

	if r.Method != http.MethodGet {
		writeJSONError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ctx := r.Context()

	sentences, err := models.GetStoryRecallSentences(ctx, storyID)
	if err != nil {
		h.log.Error("Failed to fetch recall sentences", "error", err, "storyID", storyID)
		writeJSONError(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	words, err := models.GetStoryTargetVocabulary(ctx, storyID)
	if err != nil {
		h.log.Error("Failed to fetch target vocabulary", "error", err, "storyID", storyID)
		writeJSONError(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// A storage outage must not make the editor unusable: the author can still
	// see and fix the text side of the content, so log and serve unsigned.
	if err := models.SignRecallSentenceURLs(ctx, sentences, signedURLExpiry); err != nil {
		h.log.Warn("Failed to sign recall sentence images", "error", err, "storyID", storyID)
	}

	targetVocabIDs := make(map[int]bool, len(words))
	for _, word := range words {
		targetVocabIDs[word.ID] = true
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(recallResponse{
		Sentences:        sentences,
		TargetVocabulary: words,
		Readiness:        models.ValidateRecallSentences(sentences, targetVocabIDs),
		Required:         models.RecallSentencesPerStory,
	})
}

// recallSentenceHandler upserts or removes the sentence at a given sequence
// position. Addressing by position rather than by row ID matches the fixed
// five-slot shape of the exercise.
func (h *Handler) recallSentenceHandler(w http.ResponseWriter, r *http.Request) {
	storyID, ok := h.authorizeStoryEdit(w, r)
	if !ok {
		return
	}

	order, err := strconv.Atoi(mux.Vars(r)["order"])
	if err != nil || order < 1 || order > models.RecallSentencesPerStory {
		writeJSONError(w, "Sequence order must be between 1 and "+
			strconv.Itoa(models.RecallSentencesPerStory), http.StatusBadRequest)
		return
	}

	switch r.Method {
	case http.MethodPut:
		h.saveRecallSentence(w, r, storyID, order)
	case http.MethodDelete:
		h.deleteRecallSentence(w, r, storyID, order)
	default:
		writeJSONError(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *Handler) saveRecallSentence(w http.ResponseWriter, r *http.Request, storyID, order int) {
	ctx := r.Context()

	var req recallSentenceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	hebrewText := strings.TrimSpace(req.HebrewText)
	if hebrewText == "" {
		writeJSONError(w, "hebrewText is required", http.StatusBadRequest)
		return
	}

	// A target word from another story would show the student a picture that has
	// nothing to do with this exercise.
	if req.TargetVocabID != nil {
		word, err := models.GetTargetVocabulary(ctx, *req.TargetVocabID)
		if err != nil {
			h.writeOwnerError(w, err, assetRecallImage)
			return
		}
		if word.StoryID != storyID {
			writeJSONError(w, "Target word does not belong to this story", http.StatusBadRequest)
			return
		}
	}

	// The sentence may not exist yet, in which case there is no ID to derive an
	// image path from and no previous image to clean up.
	existing := h.existingRecallSentence(r, storyID, order)

	sentence := models.RecallSentence{
		StoryID:       storyID,
		SequenceOrder: order,
		HebrewText:    hebrewText,
		TargetVocabID: req.TargetVocabID,
	}
	if existing != nil {
		sentence.ImagePath = existing.ImagePath
		sentence.ImageBucket = existing.ImageBucket
	}

	if req.ImagePath != nil {
		if existing == nil && *req.ImagePath != "" {
			writeJSONError(w, "Save the sentence before uploading its picture", http.StatusBadRequest)
			return
		}
		ownerID := 0
		if existing != nil {
			ownerID = existing.ID
		}
		imageBucket, ok := validateAssetPath(assetRecallImage, storyID, ownerID, *req.ImagePath)
		if !ok {
			writeJSONError(w, "imagePath was not issued for this recall sentence", http.StatusBadRequest)
			return
		}
		sentence.ImagePath = *req.ImagePath
		sentence.ImageBucket = imageBucket
	}

	saved, err := models.UpsertRecallSentence(ctx, sentence)
	if err != nil {
		h.log.Error("Failed to save recall sentence", "error", err, "storyID", storyID, "order", order)
		writeJSONError(w, "Failed to save sentence", http.StatusInternalServerError)
		return
	}

	if existing != nil {
		h.removeSupersededAsset(r, existing.ImageBucket, existing.ImagePath, saved.ImagePath)
	}

	if err := models.SignRecallSentenceURLs(ctx, []models.RecallSentence{*saved}, signedURLExpiry); err != nil {
		h.log.Warn("Failed to sign recall sentence image", "error", err, "sentenceID", saved.ID)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(saved)
}

func (h *Handler) deleteRecallSentence(w http.ResponseWriter, r *http.Request, storyID, order int) {
	existing := h.existingRecallSentence(r, storyID, order)
	if existing == nil {
		writeJSONError(w, "No sentence at position "+strconv.Itoa(order), http.StatusNotFound)
		return
	}

	if err := models.DeleteRecallSentence(r.Context(), existing.ID); err != nil {
		h.log.Error("Failed to delete recall sentence", "error", err, "sentenceID", existing.ID)
		writeJSONError(w, "Failed to delete sentence", http.StatusInternalServerError)
		return
	}

	h.removeSupersededAsset(r, existing.ImageBucket, existing.ImagePath, "")

	w.WriteHeader(http.StatusNoContent)
}

// existingRecallSentence returns the story's sentence at a sequence position, or
// nil when the slot is empty or unreadable.
func (h *Handler) existingRecallSentence(r *http.Request, storyID, order int) *models.RecallSentence {
	sentences, err := models.GetStoryRecallSentences(r.Context(), storyID)
	if err != nil {
		h.log.Error("Failed to fetch recall sentences", "error", err, "storyID", storyID)
		return nil
	}
	for i := range sentences {
		if sentences[i].SequenceOrder == order {
			return &sentences[i]
		}
	}
	return nil
}
