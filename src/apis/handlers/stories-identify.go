package handlers

import (
	"encoding/json"
	"glossias/src/apis/types"
	"glossias/src/auth"
	"glossias/src/pkg/models"
	"net/http"
	"slices"
	"strconv"

	"github.com/gorilla/mux"
)

// GetIdentifyPage returns the Identify phase payload: every line segmented with
// target-word markers, the story's target words with signed word-audio and
// picture URLs, and signed narration URLs per line.
//
// A story whose target vocabulary is not yet authored returns an empty
// target_words list rather than an error, so the phase degrades to plain
// narration instead of failing.
func (h *Handler) GetIdentifyPage(w http.ResponseWriter, r *http.Request) {
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

	words, err := models.GetStoryTargetVocabulary(ctx, id)
	if err != nil {
		h.log.Error("Failed to fetch target vocabulary", "error", err, "storyID", id)
		h.sendError(w, "Failed to fetch target vocabulary", http.StatusInternalServerError)
		return
	}
	if err := models.SignTargetVocabularyURLs(ctx, words, expiresInSeconds); err != nil {
		h.log.Error("Failed to sign target vocabulary URLs", "error", err, "storyID", id)
		h.sendError(w, "Failed to sign asset URLs", http.StatusInternalServerError)
		return
	}

	occurrences, err := models.GetTargetVocabularyOccurrences(ctx, id)
	if err != nil {
		h.log.Error("Failed to fetch target vocabulary occurrences", "error", err, "storyID", id)
		h.sendError(w, "Failed to fetch target vocabulary", http.StatusInternalServerError)
		return
	}

	// Narration is optional at the storage layer (the Vocab and Translate
	// pages fetch it separately and tolerate failure); keep that tolerance.
	audioURLs, err := models.GetSignedAudioURLsForStory(ctx, id, userID, "complete", expiresInSeconds)
	if err != nil {
		h.log.Warn("Failed to sign narration URLs for identify page", "error", err, "storyID", id)
		audioURLs = map[int]string{}
	}

	answers, err := models.GetUserIdentifyCorrectAnswers(ctx, userID, id)
	if err != nil {
		h.log.Error("Failed to fetch identify answers", "error", err, "storyID", id, "userID", userID)
		h.sendError(w, "Failed to fetch identify progress", http.StatusInternalServerError)
		return
	}
	picks, completed := identifyProgress(occurrences, answers)

	targetWords := make([]types.IdentifyTargetWord, 0, len(words))
	for _, w := range words {
		targetWords = append(targetWords, types.IdentifyTargetWord{
			ID:          w.ID,
			LexicalForm: w.LexicalForm,
			AudioURL:    w.AudioURL,
			ImageURL:    w.ImageURL,
		})
	}

	data := types.IdentifyPageData{
		PageData: types.PageData{
			StoryID:    storyID,
			StoryTitle: story.Metadata.Title["en"],
			Language:   story.Metadata.Language,
		},
		Lines:        buildIdentifyLines(story.Content.Lines, occurrences),
		TargetWords:  targetWords,
		AudioURLs:    audioURLs,
		CorrectPicks: picks,
		Completed:    completed,
	}

	json.NewEncoder(w).Encode(types.APIResponse{Success: true, Data: data})
}

// identifyProgress reduces the user's correct answers to the distinct
// (line, word) quizzes answered, 0-based and ordered by line, and reports
// whether the phase is complete: every target-word occurrence in the story has
// a correct pick. A story with no occurrences is never complete this way — it
// has no quizzes, so the student just listens through.
//
// Completion is derived rather than stored so the phase cannot be redone once
// finished and so mixed-generation data (extra answers, retries, words added
// later by an author) resolves the same way every time.
func identifyProgress(occurrences []models.TargetVocabularyOccurrence, answers []models.IdentifyAnswer) ([]types.IdentifyPick, bool) {
	type key struct{ line, word int }
	answered := make(map[key]bool, len(answers))
	picks := make([]types.IdentifyPick, 0, len(answers))
	for _, a := range answers {
		k := key{a.LineNumber, a.TargetVocabID}
		if answered[k] {
			continue
		}
		answered[k] = true
		picks = append(picks, types.IdentifyPick{LineIndex: a.LineNumber - 1, TargetVocabID: a.TargetVocabID})
	}
	slices.SortFunc(picks, func(a, b types.IdentifyPick) int {
		if a.LineIndex != b.LineIndex {
			return a.LineIndex - b.LineIndex
		}
		return a.TargetVocabID - b.TargetVocabID
	})

	if len(occurrences) == 0 {
		return picks, false
	}
	for _, occ := range occurrences {
		if !answered[key{occ.LineNumber, occ.TargetVocabID}] {
			return picks, false
		}
	}
	return picks, true
}

// buildIdentifyLines segments each line's text, marking target-word
// occurrences as "target" segments. Occurrence positions are rune offsets and
// line numbers are 1-based, as stored in vocabulary_items. Overlapping or
// out-of-range occurrences are dropped rather than corrupting the text.
func buildIdentifyLines(storyLines []models.StoryLine, occurrences []models.TargetVocabularyOccurrence) []types.IdentifyLine {
	byLine := make(map[int][]models.TargetVocabularyOccurrence)
	for _, occ := range occurrences {
		byLine[occ.LineNumber] = append(byLine[occ.LineNumber], occ)
	}

	lines := make([]types.IdentifyLine, 0, len(storyLines))
	for i, line := range storyLines {
		runes := []rune(line.Text)
		occs := byLine[i+1]
		slices.SortFunc(occs, func(a, b models.TargetVocabularyOccurrence) int {
			return a.Position[0] - b.Position[0]
		})

		segments := make([]types.TextSegment, 0, 2*len(occs)+1)
		targetIDs := make([]int, 0, len(occs))
		lastEnd := 0
		for _, occ := range occs {
			start, end := occ.Position[0], occ.Position[1]
			if start < lastEnd || start >= end || end > len(runes) {
				continue
			}
			if start > lastEnd {
				segments = append(segments, types.TextSegment{Text: string(runes[lastEnd:start]), Type: "text"})
			}
			segments = append(segments, types.TextSegment{
				Text:          string(runes[start:end]),
				Type:          "target",
				TargetVocabID: occ.TargetVocabID,
			})
			if !slices.Contains(targetIDs, occ.TargetVocabID) {
				targetIDs = append(targetIDs, occ.TargetVocabID)
			}
			lastEnd = end
		}
		if lastEnd < len(runes) || len(segments) == 0 {
			segments = append(segments, types.TextSegment{Text: string(runes[lastEnd:]), Type: "text"})
		}

		lines = append(lines, types.IdentifyLine{Text: segments, TargetVocabIDs: targetIDs})
	}
	return lines
}

// CheckIdentify records a picture pick for a target word on a line and reports
// whether it was correct. Modeled on CheckVocab: the pick is validated against
// the story's own target words and occurrences before anything is written.
func (h *Handler) CheckIdentify(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var req types.CheckIdentifyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.log.Warn("Invalid request body in CheckIdentify", "error", err, "ip", r.RemoteAddr)
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
		h.log.Error("Failed to fetch story in CheckIdentify", "error", err, "storyID", id)
		h.sendError(w, "Failed to fetch story", http.StatusInternalServerError)
		return
	}

	words, err := models.GetStoryTargetVocabulary(ctx, id)
	if err != nil {
		h.log.Error("Failed to fetch target vocabulary in CheckIdentify", "error", err, "storyID", id)
		h.sendError(w, "Failed to fetch target vocabulary", http.StatusInternalServerError)
		return
	}
	isStoryWord := func(wordID int) bool {
		return slices.ContainsFunc(words, func(w models.TargetVocabulary) bool { return w.ID == wordID })
	}
	if !isStoryWord(req.TargetVocabID) || !isStoryWord(req.SelectedTargetVocabID) {
		h.log.Warn("Target word does not belong to story in CheckIdentify",
			"storyID", id, "target", req.TargetVocabID, "selected", req.SelectedTargetVocabID, "ip", r.RemoteAddr)
		h.sendError(w, "Target word does not belong to this story", http.StatusBadRequest)
		return
	}

	occurrences, err := models.GetTargetVocabularyOccurrences(ctx, id)
	if err != nil {
		h.log.Error("Failed to fetch occurrences in CheckIdentify", "error", err, "storyID", id)
		h.sendError(w, "Failed to fetch target vocabulary", http.StatusInternalServerError)
		return
	}
	lineNumber := req.LineIndex + 1 // stored 1-based, like vocab answers
	onLine := slices.ContainsFunc(occurrences, func(o models.TargetVocabularyOccurrence) bool {
		return o.TargetVocabID == req.TargetVocabID && o.LineNumber == lineNumber
	})
	if !onLine {
		h.log.Warn("Target word not on line in CheckIdentify",
			"storyID", id, "target", req.TargetVocabID, "line", lineNumber, "ip", r.RemoteAddr)
		h.sendError(w, "Target word does not appear on that line", http.StatusBadRequest)
		return
	}

	correct, err := models.SaveIdentifyAnswer(ctx, userID, id, lineNumber, req.TargetVocabID, req.SelectedTargetVocabID)
	if err != nil {
		h.log.Error("Failed to save identify answer", "error", err, "userID", userID, "storyID", id, "line", lineNumber)
		h.sendError(w, "Failed to save answer", http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(types.APIResponse{
		Success: true,
		Data:    types.CheckIdentifyResponse{Correct: correct},
	})
}
