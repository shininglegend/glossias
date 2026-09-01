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

// targetVocabularyResponse is what the target-vocabulary editor loads: the
// story's chosen words with signed asset URLs for preview, every annotated
// lexical form as a candidate list, and the Identify phase's readiness report.
type targetVocabularyResponse struct {
	Words      []models.TargetVocabulary `json:"words"`
	Candidates []models.LexicalFormCount `json:"candidates"`
	Readiness  models.PhaseReadiness     `json:"readiness"`
	Required   int                       `json:"required"`
	MinOccurs  int                       `json:"minOccurrences"`
}

type targetVocabularyRequest struct {
	LexicalForm string `json:"lexicalForm"`
	// AudioPath and ImagePath are paths returned by the phase-asset upload
	// endpoint. Omitting a field leaves the stored asset alone; sending an empty
	// string clears it and deletes the stored file.
	AudioPath *string `json:"audioPath,omitempty"`
	ImagePath *string `json:"imagePath,omitempty"`
}

func (h *Handler) targetVocabularyHandler(w http.ResponseWriter, r *http.Request) {
	storyID, ok := h.authorizeStoryEdit(w, r)
	if !ok {
		return
	}

	switch r.Method {
	case http.MethodGet:
		h.listTargetVocabulary(w, r, storyID)
	case http.MethodPost:
		h.createTargetVocabulary(w, r, storyID)
	default:
		writeJSONError(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *Handler) targetVocabularyItemHandler(w http.ResponseWriter, r *http.Request) {
	storyID, ok := h.authorizeStoryEdit(w, r)
	if !ok {
		return
	}

	wordID, err := strconv.Atoi(mux.Vars(r)["wordId"])
	if err != nil {
		writeJSONError(w, "Invalid target word ID", http.StatusBadRequest)
		return
	}

	// Load first so every branch can confirm the word belongs to this story.
	word, err := models.GetTargetVocabulary(r.Context(), wordID)
	if err != nil {
		h.writeOwnerError(w, err, assetTargetVocabImage)
		return
	}
	if word.StoryID != storyID {
		writeJSONError(w, "Target word not found for this story", http.StatusNotFound)
		return
	}

	switch r.Method {
	case http.MethodPut:
		h.updateTargetVocabulary(w, r, storyID, word)
	case http.MethodDelete:
		h.deleteTargetVocabulary(w, r, storyID, word)
	default:
		writeJSONError(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *Handler) listTargetVocabulary(w http.ResponseWriter, r *http.Request, storyID int) {
	ctx := r.Context()

	words, err := models.GetStoryTargetVocabulary(ctx, storyID)
	if err != nil {
		h.log.Error("Failed to fetch target vocabulary", "error", err, "storyID", storyID)
		writeJSONError(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	candidates, err := models.GetStoryLexicalFormCounts(ctx, storyID)
	if err != nil {
		h.log.Error("Failed to fetch lexical form counts", "error", err, "storyID", storyID)
		writeJSONError(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// A storage outage must not make the editor unusable: the author can still
	// see and fix the text side of the content, so log and serve unsigned.
	if err := models.SignTargetVocabularyURLs(ctx, words, signedURLExpiry); err != nil {
		h.log.Warn("Failed to sign target vocabulary assets", "error", err, "storyID", storyID)
	}

	occurrences := make(map[string]int, len(candidates))
	for _, candidate := range candidates {
		occurrences[candidate.LexicalForm] = candidate.Occurrences
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(targetVocabularyResponse{
		Words:      words,
		Candidates: candidates,
		Readiness:  models.ValidateTargetVocabulary(words, occurrences),
		Required:   models.TargetWordsPerStory,
		MinOccurs:  models.MinTargetWordOccurrences,
	})
}

func (h *Handler) createTargetVocabulary(w http.ResponseWriter, r *http.Request, storyID int) {
	ctx := r.Context()

	var req targetVocabularyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	lexicalForm := strings.TrimSpace(req.LexicalForm)
	if lexicalForm == "" {
		writeJSONError(w, "lexicalForm is required", http.StatusBadRequest)
		return
	}

	// Refuse a sixth word: the Identify popup shows exactly five pictures, so an
	// extra word would silently change the exercise.
	count, err := models.CountStoryTargetVocabulary(ctx, storyID)
	if err != nil {
		h.log.Error("Failed to count target vocabulary", "error", err, "storyID", storyID)
		writeJSONError(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	if count >= models.TargetWordsPerStory {
		writeJSONError(w, "This story already has "+strconv.Itoa(models.TargetWordsPerStory)+
			" target words; remove one before adding another", http.StatusConflict)
		return
	}

	// The word must already be annotated often enough for the Identify phase to
	// pause on it more than once. Enforced here rather than at runtime, per spec.
	occurrences, err := h.lexicalFormOccurrences(r, storyID, lexicalForm)
	if err != nil {
		h.log.Error("Failed to count lexical form occurrences", "error", err, "storyID", storyID)
		writeJSONError(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	if occurrences < models.MinTargetWordOccurrences {
		writeJSONError(w, "\""+lexicalForm+"\" is annotated "+strconv.Itoa(occurrences)+
			" time(s) in this story; at least "+strconv.Itoa(models.MinTargetWordOccurrences)+
			" occurrences are required", http.StatusBadRequest)
		return
	}

	// Assets are attached after creation, once the word has an ID to derive an
	// upload path from.
	word, err := models.CreateTargetVocabulary(ctx, models.TargetVocabulary{
		StoryID:     storyID,
		LexicalForm: lexicalForm,
	})
	if err != nil {
		if errors.Is(err, models.ErrDuplicate) {
			writeJSONError(w, "\""+lexicalForm+"\" is already a target word for this story", http.StatusConflict)
			return
		}
		h.log.Error("Failed to create target vocabulary", "error", err, "storyID", storyID)
		writeJSONError(w, "Failed to create target word", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(word)
}

func (h *Handler) updateTargetVocabulary(w http.ResponseWriter, r *http.Request, storyID int, existing *models.TargetVocabulary) {
	ctx := r.Context()

	var req targetVocabularyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	updated := *existing

	if lexicalForm := strings.TrimSpace(req.LexicalForm); lexicalForm != "" && lexicalForm != existing.LexicalForm {
		occurrences, err := h.lexicalFormOccurrences(r, storyID, lexicalForm)
		if err != nil {
			h.log.Error("Failed to count lexical form occurrences", "error", err, "storyID", storyID)
			writeJSONError(w, "Internal server error", http.StatusInternalServerError)
			return
		}
		if occurrences < models.MinTargetWordOccurrences {
			writeJSONError(w, "\""+lexicalForm+"\" is annotated "+strconv.Itoa(occurrences)+
				" time(s) in this story; at least "+strconv.Itoa(models.MinTargetWordOccurrences)+
				" occurrences are required", http.StatusBadRequest)
			return
		}
		updated.LexicalForm = lexicalForm
	}

	if req.AudioPath != nil {
		audioBucket, ok := validateAssetPath(assetTargetVocabAudio, storyID, existing.ID, *req.AudioPath)
		if !ok {
			writeJSONError(w, "audioPath was not issued for this target word", http.StatusBadRequest)
			return
		}
		updated.AudioPath = *req.AudioPath
		updated.AudioBucket = audioBucket
	}

	if req.ImagePath != nil {
		imageBucket, ok := validateAssetPath(assetTargetVocabImage, storyID, existing.ID, *req.ImagePath)
		if !ok {
			writeJSONError(w, "imagePath was not issued for this target word", http.StatusBadRequest)
			return
		}
		updated.CorrectImagePath = *req.ImagePath
		updated.ImageBucket = imageBucket
	}

	saved, err := models.UpdateTargetVocabulary(ctx, updated)
	if err != nil {
		if errors.Is(err, models.ErrDuplicate) {
			writeJSONError(w, "\""+updated.LexicalForm+"\" is already a target word for this story", http.StatusConflict)
			return
		}
		h.log.Error("Failed to update target vocabulary", "error", err, "wordID", existing.ID)
		writeJSONError(w, "Failed to update target word", http.StatusInternalServerError)
		return
	}

	// The row now points at the new files, so any superseded upload is
	// unreachable — remove it. A storage failure here leaks a file but must not
	// fail a save that already succeeded.
	h.removeSupersededAsset(r, existing.AudioBucket, existing.AudioPath, saved.AudioPath)
	h.removeSupersededAsset(r, existing.ImageBucket, existing.CorrectImagePath, saved.CorrectImagePath)

	if err := models.SignTargetVocabularyURLs(ctx, []models.TargetVocabulary{*saved}, signedURLExpiry); err != nil {
		h.log.Warn("Failed to sign target vocabulary assets", "error", err, "wordID", saved.ID)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(saved)
}

func (h *Handler) deleteTargetVocabulary(w http.ResponseWriter, r *http.Request, storyID int, word *models.TargetVocabulary) {
	if err := models.DeleteTargetVocabulary(r.Context(), storyID, word.ID); err != nil {
		h.log.Error("Failed to delete target vocabulary", "error", err, "wordID", word.ID, "storyID", storyID)
		writeJSONError(w, "Failed to delete target word", http.StatusInternalServerError)
		return
	}

	// recall_sentences.target_vocab_id is ON DELETE SET NULL, so affected
	// sentences survive with a dangling link that the readiness report flags.
	h.removeSupersededAsset(r, word.AudioBucket, word.AudioPath, "")
	h.removeSupersededAsset(r, word.ImageBucket, word.CorrectImagePath, "")

	w.WriteHeader(http.StatusNoContent)
}

// lexicalFormOccurrences counts how many times a lexical form is annotated in a
// story's text.
func (h *Handler) lexicalFormOccurrences(r *http.Request, storyID int, lexicalForm string) (int, error) {
	counts, err := models.GetStoryLexicalFormCounts(r.Context(), storyID)
	if err != nil {
		return 0, err
	}
	for _, count := range counts {
		if count.LexicalForm == lexicalForm {
			return count.Occurrences, nil
		}
	}
	return 0, nil
}

// removeSupersededAsset deletes a stored file that is no longer referenced.
func (h *Handler) removeSupersededAsset(r *http.Request, oldBucket, oldPath, newPath string) {
	if oldPath == "" || oldPath == newPath {
		return
	}
	if err := models.DeleteStorageObject(r.Context(), oldBucket, oldPath); err != nil {
		h.log.Warn("Failed to delete superseded asset", "error", err, "bucket", oldBucket, "path", oldPath)
	}
}
