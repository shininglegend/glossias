package handlers

import (
	"encoding/json"
	"fmt"
	"glossias/src/apis/types"
	"glossias/src/auth"
	"glossias/src/pkg/models"
	"net/http"
	"slices"
	"strconv"

	"github.com/gorilla/mux"
)

// GetVocabPage returns JSON data for story page 2 (vocabulary)
func (h *Handler) GetVocabPage(w http.ResponseWriter, r *http.Request) {
	storyID := mux.Vars(r)["id"]
	id, err := strconv.Atoi(storyID)
	if err != nil {
		h.sendError(w, "Invalid story ID format", http.StatusBadRequest)
		return
	}

	story, err := models.GetStoryData(r.Context(), id, auth.GetUserID(r))
	if err == models.ErrNotFound {
		h.sendError(w, "Story not found", http.StatusNotFound)
		return
	}
	if err != nil {
		h.log.Error("Failed to fetch story data", "error", err)
		h.sendError(w, "Failed to fetch story data", http.StatusInternalServerError)
		return
	}

	lines, vocabBank := h.generateVocabLines(*story, id)

	data := types.VocabPageData{
		PageData: types.PageData{
			StoryID:    storyID,
			StoryTitle: story.Metadata.Title["en"],
			Language:   story.Metadata.Description.Language,
		},
		Lines:     lines,
		VocabBank: vocabBank,
	}

	response := types.APIResponse{
		Success: true,
		Data:    data,
	}

	json.NewEncoder(w).Encode(response)
}

// generateVocabLines prepares lines with vocabulary blanks and vocab bank
func (h *Handler) generateVocabLines(story models.Story, id int) ([]types.Line, []string) {
	lines := make([]types.Line, len(story.Content.Lines))
	vocabBank := make([]string, 0)

	for i, line := range story.Content.Lines {
		series := []string{}
		runes := []rune(line.Text)
		lastEnd := 0

		// Sort vocab words by position
		slices.SortFunc(line.Vocabulary, h.sortVocab)

		for j := 0; j < len(line.Vocabulary); j++ {
			vocab := line.Vocabulary[j]
			vocabBank = append(vocabBank, vocab.LexicalForm)
			start := vocab.Position[0]
			if start >= lastEnd {
				series = append(series, string(runes[lastEnd:start]))
			}
			series = append(series, vocabBlank)
			lastEnd = vocab.Position[1]
		}
		if lastEnd < len(runes) {
			series = append(series, string(runes[lastEnd:]))
		}

		// Convert audio files to API format (vocab_missing label only)
		audioFiles := make([]types.AudioFile, 0)
		for _, audio := range line.AudioFiles {
			if audio.Label == "vocab_missing" {
				audioFiles = append(audioFiles, types.AudioFile{
					ID:         audio.ID,
					FilePath:   audio.FilePath,
					FileBucket: audio.FileBucket,
					Label:      audio.Label,
				})
			}
		}

		lines[i] = types.Line{
			Text:       series,
			AudioFiles: audioFiles,
		}
	}

	// Sort and dedupe vocab bank
	slices.Sort(vocabBank)
	vocabBank = slices.Compact(vocabBank)

	return lines, vocabBank
}

// CheckVocab handles vocabulary checking for a single line
func (h *Handler) CheckVocab(w http.ResponseWriter, r *http.Request) {

	var req types.CheckVocabRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.log.Warn("Invalid request body in CheckVocab", "error", err, "ip", r.RemoteAddr)
		h.sendError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Only process one line at a time
	if len(req.Answers) != 1 {
		h.sendError(w, "Must provide answers for exactly one line", http.StatusBadRequest)
		return
	}

	storyID := mux.Vars(r)["id"]
	id, err := strconv.Atoi(storyID)
	if err != nil {
		h.log.Warn("Invalid story ID in CheckVocab", "storyID", storyID, "ip", r.RemoteAddr)
		h.sendError(w, "Invalid story ID", http.StatusBadRequest)
		return
	}

	story, err := models.GetStoryData(r.Context(), id, auth.GetUserID(r))
	if err != nil {
		h.log.Error("Failed to fetch story in CheckVocab", "error", err, "storyID", id)
		h.sendError(w, "Failed to fetch story", http.StatusInternalServerError)
		return
	}

	answer := req.Answers[0]

	// Validate line number
	if answer.LineNumber < 0 || answer.LineNumber >= len(story.Content.Lines) {
		h.log.Warn("Invalid line number in CheckVocab", "lineNumber", answer.LineNumber, "maxLines", len(story.Content.Lines), "ip", r.RemoteAddr)
		h.sendError(w, fmt.Sprintf("Invalid line number: %d", answer.LineNumber), http.StatusBadRequest)
		return
	}

	line := story.Content.Lines[answer.LineNumber]
	expectedVocabCount := len(line.Vocabulary)

	// Validate answer count
	if len(answer.Answers) != expectedVocabCount {
		h.log.Warn("Wrong number of answers provided", "lineNumber", answer.LineNumber, "expected", expectedVocabCount, "provided", len(answer.Answers), "ip", r.RemoteAddr)
		h.sendError(w, fmt.Sprintf("Expected %d answers, got %d", expectedVocabCount, len(answer.Answers)), http.StatusBadRequest)
		return
	}

	// Check if all answers are correct
	allCorrect := true
	for i, userAnswer := range answer.Answers {
		if i >= len(line.Vocabulary) || userAnswer != line.Vocabulary[i].LexicalForm {
			allCorrect = false
			break
		}
	}

	// Save score if available
	userID := auth.GetUserID(r)
	if userID != "" {
		incorrectAnswer := ""
		if !allCorrect {
			// Join the user's answers to save as incorrect answer
			incorrectAnswer = fmt.Sprintf("%v", answer.Answers)
		}
		if err := models.SaveVocabScore(r.Context(), userID, id, answer.LineNumber, allCorrect, incorrectAnswer); err != nil {
			h.log.Error("Failed to save vocab score", "error", err, "userID", userID, "storyID", id, "line", answer.LineNumber)
		}
	}

	response := types.APIResponse{
		Success: true,
		Data: types.CheckVocabResponse{
			Correct: allCorrect,
		},
	}

	json.NewEncoder(w).Encode(response)
}
