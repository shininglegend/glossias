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
			Lines:      lines,
		},
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

		hasVocab := len(line.Vocabulary) > 0

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
			Text:              series,
			AudioFiles:        audioFiles,
			HasVocabOrGrammar: hasVocab,
		}
	}

	// Sort and dedupe vocab bank
	slices.Sort(vocabBank)
	vocabBank = slices.Compact(vocabBank)

	return lines, vocabBank
}

// CheckVocab handles vocabulary checking
func (h *Handler) CheckVocab(w http.ResponseWriter, r *http.Request) {

	var req types.CheckVocabRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.log.Warn("Invalid request body in CheckVocab", "error", err, "ip", r.RemoteAddr)
		h.sendError(w, "Invalid request body", http.StatusBadRequest)
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

	// Build expected answers map for validation
	expectedAnswers := make(map[int]int)
	for i, line := range story.Content.Lines {
		expectedAnswers[i] = len(line.Vocabulary)
	}

	var results []types.VocabResult

	// Process each line's answers
	for _, answer := range req.Answers {
		// Validate line number
		if answer.LineNumber < 0 || answer.LineNumber >= len(story.Content.Lines) {
			h.log.Warn("Invalid line number in CheckVocab", "lineNumber", answer.LineNumber, "maxLines", len(story.Content.Lines), "ip", r.RemoteAddr)
			h.sendValidationError(w, fmt.Sprintf("Invalid line number: %d", answer.LineNumber), expectedAnswers)
			return
		}

		line := story.Content.Lines[answer.LineNumber]

		// Build correct answers map
		correctAnswers := make(map[string]bool)
		expectedVocabCount := len(line.Vocabulary)

		for _, vocab := range line.Vocabulary {
			correctAnswers[vocab.LexicalForm] = true
		}

		h.log.Debug("Correct answers for line", "lineNumber", answer.LineNumber, "correctAnswers", correctAnswers)

		// Validate answer count
		if len(answer.Answers) != expectedVocabCount {
			h.log.Warn("Wrong number of answers provided", "lineNumber", answer.LineNumber, "expected", expectedVocabCount, "provided", len(answer.Answers), "ip", r.RemoteAddr)
			h.sendValidationError(w, fmt.Sprintf("Line %d expects %d answers, got %d", answer.LineNumber, expectedVocabCount, len(answer.Answers)), expectedAnswers)
			return
		}

		// Process each answer for this line
		for i, userAnswer := range answer.Answers {
			isCorrect := correctAnswers[userAnswer]

			// Find the correct answer for this position
			var correctAnswer string
			if i < len(line.Vocabulary) {
				correctAnswer = line.Vocabulary[i].LexicalForm
			}

			results = append(results, types.VocabResult{
				Correct:       isCorrect,
				UserAnswer:    userAnswer,
				CorrectAnswer: correctAnswer,
				Line:          answer.LineNumber,
			})
		}
	}

	h.log.Debug("Vocab check completed", "ip", r.RemoteAddr, "totalResults", len(results))

	response := types.APIResponse{
		Success: true,
		Data: types.CheckVocabResponse{
			Answers: results,
		},
	}

	json.NewEncoder(w).Encode(response)
}
