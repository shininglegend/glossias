package handlers

import (
	"encoding/json"
	"errors"
	"glossias/src/apis/types"
	"glossias/src/auth"
	"glossias/src/pkg/models"
	"math/rand/v2"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
)

// GetRecallPage returns the Recall phase payload: the story's recall sentences
// shuffled with their correct position withheld, signed picture URLs, signed
// narration URLs per line (the phase plays the story audio-only), and the
// student's progress so far.
//
// A story with no recall sentences returns an empty list rather than an error,
// so the phase degrades to narration plus a notice instead of failing.
func (h *Handler) GetRecallPage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	storyID := mux.Vars(r)["id"]
	id, err := strconv.Atoi(storyID)
	if err != nil {
		h.sendError(w, "Invalid story ID format", http.StatusBadRequest)
		return
	}
	userID := auth.GetUserID(r)
	if userID == "" {
		h.sendError(w, "User not authenticated", http.StatusUnauthorized)
		return
	}

	// GetStoryData enforces course access.
	story, err := models.GetStoryData(ctx, id, userID)
	if err == models.ErrNotFound {
		h.sendError(w, "Story not found", http.StatusNotFound)
		return
	}
	if err != nil {
		h.log.Error("Failed to fetch story data", "error", err)
		h.sendError(w, "Failed to fetch story data", http.StatusInternalServerError)
		return
	}

	sentences, err := models.GetStoryRecallSentences(ctx, id)
	if err != nil {
		h.log.Error("Failed to fetch recall sentences", "error", err, "storyID", id)
		h.sendError(w, "Failed to fetch recall sentences", http.StatusInternalServerError)
		return
	}
	if err := models.SignRecallSentenceURLs(ctx, sentences, expiresInSeconds); err != nil {
		h.log.Error("Failed to sign recall sentence URLs", "error", err, "storyID", id)
		h.sendError(w, "Failed to sign asset URLs", http.StatusInternalServerError)
		return
	}

	// Narration is optional at the storage layer (the Vocab and Translate
	// pages fetch it separately and tolerate failure); keep that tolerance.
	audioURLs, err := models.GetSignedAudioURLsForStory(ctx, id, userID, "complete", expiresInSeconds)
	if err != nil {
		h.log.Warn("Failed to sign narration URLs for recall page", "error", err, "storyID", id)
		audioURLs = map[int]string{}
	}

	correctIDs, err := models.GetUserRecallCorrectSentenceIDs(ctx, userID, id)
	if err != nil {
		h.log.Error("Failed to fetch recall answers", "error", err, "storyID", id, "userID", userID)
		h.sendError(w, "Failed to fetch recall progress", http.StatusInternalServerError)
		return
	}
	summary, err := models.GetUserStoryRecallSummary(ctx, userID, id)
	if err != nil {
		h.log.Error("Failed to fetch recall summary", "error", err, "storyID", id, "userID", userID)
		h.sendError(w, "Failed to fetch recall progress", http.StatusInternalServerError)
		return
	}

	data := types.RecallPageData{
		PageData: types.PageData{
			StoryID:    storyID,
			StoryTitle: story.Metadata.Title["en"],
			Language:   story.Metadata.Language,
		},
		LineCount: len(story.Content.Lines),
		AudioURLs: audioURLs,
		Sentences: shuffledRecallCards(sentences, rand.Shuffle),
		Attempts:  recallAttempts(summary, len(sentences)),
		Completed: recallCompleted(sentences, correctIDs),
	}

	json.NewEncoder(w).Encode(types.APIResponse{Success: true, Data: data})
}

// shuffledRecallCards strips SequenceOrder from the sentences and returns them
// in a random order. `shuffle` is injected so tests can pin the permutation.
func shuffledRecallCards(sentences []models.RecallSentence, shuffle func(n int, swap func(i, j int))) []types.RecallCard {
	cards := make([]types.RecallCard, 0, len(sentences))
	for _, s := range sentences {
		cards = append(cards, types.RecallCard{
			ID:         s.ID,
			HebrewText: s.HebrewText,
			ImageURL:   s.ImageURL,
		})
	}
	shuffle(len(cards), func(i, j int) { cards[i], cards[j] = cards[j], cards[i] })
	return cards
}

// recallAttempts derives how many orderings the student has submitted: every
// attempt logs exactly one row per sentence, so total rows / sentences is the
// attempt count. A story with no sentences has had no attempts.
func recallAttempts(summary models.AnswerSummary, sentenceCount int) int {
	if sentenceCount == 0 {
		return 0
	}
	return int(summary.CorrectCount+summary.IncorrectCount) / sentenceCount
}

// recallCompleted reports whether the student has placed every one of the
// story's current sentences correctly at least once. This is derived rather
// than stored so it resolves the same way for mixed-generation data (retries,
// sentences re-authored later) and mirrors how Vocab completion is judged. A
// story with no sentences is never complete — there is nothing to recall.
func recallCompleted(sentences []models.RecallSentence, correctIDs []int) bool {
	if len(sentences) == 0 {
		return false
	}
	correct := make(map[int]bool, len(correctIDs))
	for _, id := range correctIDs {
		correct[id] = true
	}
	for _, s := range sentences {
		if !correct[s.ID] {
			return false
		}
	}
	return true
}

// CheckRecall grades one submitted ordering of the story's recall sentences,
// logs every position as a correct or incorrect answer, and returns
// per-position correctness. Students may resubmit until every position is
// right; each attempt is logged so scoring can count retries.
func (h *Handler) CheckRecall(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var req types.CheckRecallRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.log.Warn("Invalid request body in CheckRecall", "error", err, "ip", r.RemoteAddr)
		h.sendError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	storyID := mux.Vars(r)["id"]
	id, err := strconv.Atoi(storyID)
	if err != nil {
		h.sendError(w, "Invalid story ID", http.StatusBadRequest)
		return
	}
	userID := auth.GetUserID(r)
	if userID == "" {
		h.sendError(w, "Authentication required", http.StatusUnauthorized)
		return
	}

	// Access check (course membership) — the returned story is otherwise unused.
	if _, err := models.GetStoryData(ctx, id, userID); err != nil {
		if err == models.ErrNotFound {
			h.sendError(w, "Story not found", http.StatusNotFound)
			return
		}
		h.log.Error("Failed to fetch story in CheckRecall", "error", err, "storyID", id)
		h.sendError(w, "Failed to fetch story", http.StatusInternalServerError)
		return
	}

	results, err := models.SaveRecallAttempt(ctx, userID, id, req.OrderedSentenceIDs)
	switch {
	case errors.Is(err, models.ErrInvalidRecallOrder):
		h.log.Warn("Invalid recall ordering", "error", err, "storyID", id, "userID", userID, "ip", r.RemoteAddr)
		h.sendError(w, "Submitted ordering does not match this story's sentences", http.StatusBadRequest)
		return
	case errors.Is(err, models.ErrNotFound):
		h.sendError(w, "This story has no recall sentences", http.StatusNotFound)
		return
	case err != nil:
		h.log.Error("Failed to save recall attempt", "error", err, "userID", userID, "storyID", id)
		h.sendError(w, "Failed to save answer", http.StatusInternalServerError)
		return
	}

	allCorrect := true
	for _, ok := range results {
		if !ok {
			allCorrect = false
			break
		}
	}

	json.NewEncoder(w).Encode(types.APIResponse{
		Success: true,
		Data:    types.CheckRecallResponse{Results: results, AllCorrect: allCorrect},
	})
}
