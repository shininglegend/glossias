package handlers

import (
	"encoding/json"
	"errors"
	"glossias/src/apis/types"
	"glossias/src/auth"
	"glossias/src/pkg/models"
	"math/rand/v2"
	"net/http"
	"slices"
	"strconv"

	"github.com/gorilla/mux"
)

// GetRecallPage returns the Recall phase payload: the story's recall sentences
// shuffled with their correct position withheld, signed picture URLs, the
// sentence audio on each card (override or chained story-line clips), target-word text segments, signed
// story-audio URLs (the phase plays the story audio-only first), and the
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

	// Target words are only used to highlight the vocab form on each card;
	// a failure here must not block the phase.
	words, err := models.GetStoryTargetVocabulary(ctx, id)
	if err != nil {
		h.log.Warn("Failed to fetch target vocabulary for recall page", "error", err, "storyID", id)
		words = nil
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
		Sentences: shuffledRecallCards(sentences, story.Content.Lines, words, audioURLs, rand.Shuffle),
		Attempts:  recallAttempts(summary, len(sentences)),
		Completed: recallCompleted(sentences, correctIDs),
	}

	json.NewEncoder(w).Encode(types.APIResponse{Success: true, Data: data})
}

// shuffledRecallCards strips SequenceOrder from the sentences and returns them
// in a random order. `shuffle` is injected so tests can pin the permutation.
func shuffledRecallCards(
	sentences []models.RecallSentence,
	lines []models.StoryLine,
	words []models.TargetVocabulary,
	audioURLs map[int]string,
	shuffle func(n int, swap func(i, j int)),
) []types.RecallCard {
	cards := make([]types.RecallCard, 0, len(sentences))
	for _, s := range sentences {
		cards = append(cards, types.RecallCard{
			ID:         s.ID,
			HebrewText: s.HebrewText,
			ImageURL:   s.ImageURL,
			AudioURLs:  recallSentenceAudio(s, lines, audioURLs),
			Text:       highlightRecallText(s, lines, words),
		})
	}
	shuffle(len(cards), func(i, j int) { cards[i], cards[j] = cards[j], cards[i] })
	return cards
}

// recallSentenceAudio is the clip(s) played when the student picks this card
// correctly. An uploaded override wins; otherwise every story-line narration
// that makes up the instructor sentence is chained in story order.
func recallSentenceAudio(s models.RecallSentence, lines []models.StoryLine, audioURLs map[int]string) []string {
	if s.AudioURL != "" {
		return []string{s.AudioURL}
	}
	if len(audioURLs) == 0 {
		return nil
	}
	var urls []string
	for _, lineNumber := range models.RecallSentenceCoveredLines(s.HebrewText, lines) {
		if url, ok := audioURLs[lineNumber]; ok {
			urls = append(urls, url)
		}
	}
	return urls
}

// highlightRecallText marks the card's target word the same way Identify
// does (type "target"), using story-line vocabulary positions when the
// sentence is a substring of a line, otherwise the lexical form itself.
func highlightRecallText(s models.RecallSentence, lines []models.StoryLine, words []models.TargetVocabulary) []types.TextSegment {
	if s.HebrewText == "" {
		return []types.TextSegment{{Text: "", Type: "text"}}
	}
	if s.TargetVocabID == nil {
		return []types.TextSegment{{Text: s.HebrewText, Type: "text"}}
	}
	targetID := *s.TargetVocabID
	form := ""
	for _, w := range words {
		if w.ID == targetID {
			form = w.LexicalForm
			break
		}
	}

	hebRunes := []rune(s.HebrewText)
	if form != "" {
		for _, line := range lines {
			start := indexRunes([]rune(line.Text), hebRunes)
			if start < 0 {
				continue
			}
			end := start + len(hebRunes)
			hits := make([][2]int, 0, 1)
			for _, v := range line.Vocabulary {
				if v.LexicalForm != form {
					continue
				}
				if v.Position[0] >= start && v.Position[1] <= end && v.Position[0] < v.Position[1] {
					hits = append(hits, [2]int{v.Position[0] - start, v.Position[1] - start})
				}
			}
			if len(hits) > 0 {
				return segmentByRanges(hebRunes, hits, targetID)
			}
		}
		if idx := indexRunes(hebRunes, []rune(form)); idx >= 0 {
			return segmentByRanges(hebRunes, [][2]int{{idx, idx + len([]rune(form))}}, targetID)
		}
	}
	return []types.TextSegment{{Text: s.HebrewText, Type: "text"}}
}

func segmentByRanges(runes []rune, ranges [][2]int, targetID int) []types.TextSegment {
	slices.SortFunc(ranges, func(a, b [2]int) int { return a[0] - b[0] })
	segs := make([]types.TextSegment, 0, 2*len(ranges)+1)
	last := 0
	for _, r := range ranges {
		start, end := r[0], r[1]
		if start < last || start >= end || end > len(runes) {
			continue
		}
		if start > last {
			segs = append(segs, types.TextSegment{Text: string(runes[last:start]), Type: "text"})
		}
		segs = append(segs, types.TextSegment{
			Text:          string(runes[start:end]),
			Type:          "target",
			TargetVocabID: targetID,
		})
		last = end
	}
	if last < len(runes) || len(segs) == 0 {
		segs = append(segs, types.TextSegment{Text: string(runes[last:]), Type: "text"})
	}
	return segs
}

func indexRunes(haystack, needle []rune) int {
	n := len(needle)
	if n == 0 || n > len(haystack) {
		return -1
	}
	for i := range len(haystack) - n + 1 {
		if slices.Equal(haystack[i:i+n], needle) {
			return i
		}
	}
	return -1
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

// CheckRecallPick grades one sequential selection: the student claims a
// sentence is the one at the asked story position. One answer row is logged.
func (h *Handler) CheckRecallPick(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var req types.CheckRecallPickRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.log.Warn("Invalid request body in CheckRecallPick", "error", err, "ip", r.RemoteAddr)
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

	if _, err := models.GetStoryData(ctx, id, userID); err != nil {
		if err == models.ErrNotFound {
			h.sendError(w, "Story not found", http.StatusNotFound)
			return
		}
		h.log.Error("Failed to fetch story in CheckRecallPick", "error", err, "storyID", id)
		h.sendError(w, "Failed to fetch story", http.StatusInternalServerError)
		return
	}

	correct, err := models.SaveRecallPick(ctx, userID, id, req.SentenceID, req.Position)
	switch {
	case errors.Is(err, models.ErrInvalidRecallOrder):
		h.log.Warn("Invalid recall pick", "error", err, "storyID", id, "userID", userID, "ip", r.RemoteAddr)
		h.sendError(w, "Submitted pick does not match this story's sentences", http.StatusBadRequest)
		return
	case errors.Is(err, models.ErrNotFound):
		h.sendError(w, "This story has no recall sentences", http.StatusNotFound)
		return
	case err != nil:
		h.log.Error("Failed to save recall pick", "error", err, "userID", userID, "storyID", id)
		h.sendError(w, "Failed to save answer", http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(types.APIResponse{
		Success: true,
		Data:    types.CheckRecallPickResponse{Correct: correct},
	})
}
