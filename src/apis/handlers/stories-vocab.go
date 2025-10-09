package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"glossias/src/apis/types"
	"glossias/src/auth"
	"glossias/src/pkg/models"
	"net/http"
	"slices"
	"strconv"
	"strings"

	"github.com/gorilla/mux"
)

// GetVocabPage returns JSON data for story page 2 (vocabulary)
func (h *Handler) GetVocabPage(w http.ResponseWriter, r *http.Request) {
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

	lines, vocabBank, err := h.generateVocabLines(ctx, *story, userID)
	if err != nil {
		h.log.Error("Failed to generate vocabulary lines", "error", err)
		h.sendError(w, "Failed to generate vocabulary lines", http.StatusInternalServerError)
		return
	}

	data := types.VocabPageData{
		PageData: types.PageData{
			StoryID:    storyID,
			StoryTitle: story.Metadata.Title["en"],
			Language:   story.Metadata.Language,
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
func (h *Handler) generateVocabLines(ctx context.Context, story models.Story, userID string) ([]types.VocabLine, []string, error) {
	// Get incomplete vocab items for this user
	incompleteVocab, err := models.GetLinesWithoutVocabForUser(ctx, userID, story.Metadata.StoryID)
	if err != nil {
		return nil, nil, err
	}

	// Create lookup map for incomplete vocab positions
	incompleteMap := make(map[int]map[int]bool)
	for _, item := range incompleteVocab {
		if incompleteMap[item.LineNumber] == nil {
			incompleteMap[item.LineNumber] = make(map[int]bool)
		}
		incompleteMap[item.LineNumber][item.Position] = true
	}
	lines := make([]types.VocabLine, len(story.Content.Lines))
	vocabBank := make([]string, 0)

	for i, line := range story.Content.Lines {
		segments := []types.TextSegment{}
		runes := []rune(line.Text)
		lastEnd := 0

		// Sort vocab words by position
		slices.SortFunc(line.Vocabulary, h.sortVocab)

		// Check if this line has any incomplete vocab
		lineIncompletePositions := incompleteMap[i+1] // Lines are 1-indexed in DB
		hasIncompleteVocab := len(lineIncompletePositions) > 0

		if hasIncompleteVocab {
			// Process vocab items, showing blanks only for incomplete ones
			for j := 0; j < len(line.Vocabulary); j++ {
				vocab := line.Vocabulary[j]
				start := vocab.Position[0]
				end := vocab.Position[1]

				if start >= lastEnd {
					segments = append(segments, types.TextSegment{
						Text: string(runes[lastEnd:start]),
						Type: "text",
					})
				}

				// Check if this vocab position is incomplete
				if lineIncompletePositions[start] {
					segments = append(segments, types.TextSegment{
						Text:     vocabBlank,
						Type:     "blank",
						VocabKey: fmt.Sprintf("%d-%d", i, j),
					})
				} else {
					// Show the actual word for completed vocab
					segments = append(segments, types.TextSegment{
						Text: vocab.LexicalForm,
						Type: "completed",
					})

				}
				lastEnd = end
			}
			if lastEnd < len(runes) {
				segments = append(segments, types.TextSegment{
					Text: string(runes[lastEnd:]),
					Type: "text",
				})
			}
		} else {
			// No incomplete vocab on this line, show original text
			segments = append(segments, types.TextSegment{
				Text: line.Text,
				Type: "text",
			})
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

		lines[i] = types.VocabLine{
			Text:       segments,
			AudioFiles: audioFiles,
		}
		// If the line had a vocab item, add it to the bank
		if len(line.Vocabulary) > 0 {
			for _, vocab := range line.Vocabulary {
				vocabBank = append(vocabBank, vocab.LexicalForm)
			}
		}
	}

	// Sort and dedupe vocab bank
	slices.Sort(vocabBank)
	vocabBank = slices.Compact(vocabBank)

	return lines, vocabBank, nil
}

// CheckVocab handles vocabulary checking for individual words
func (h *Handler) CheckVocab(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
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

	// Parse vocabKey format: "lineIndex-vocabIndex"
	parts := strings.Split(req.VocabKey, "-")
	if len(parts) != 2 {
		h.log.Warn("Invalid vocab_key format in CheckVocab", "vocab_key", req.VocabKey, "ip", r.RemoteAddr)
		h.sendError(w, "Invalid vocab_key format", http.StatusBadRequest)
		return
	}

	lineIndex, err := strconv.Atoi(parts[0])
	if err != nil {
		h.log.Warn("Invalid line index in vocab_key", "vocab_key", req.VocabKey, "ip", r.RemoteAddr)
		h.sendError(w, "Invalid line index in vocab_key", http.StatusBadRequest)
		return
	}

	vocabIndex, err := strconv.Atoi(parts[1])
	if err != nil {
		h.log.Warn("Invalid vocab index in vocab_key", "vocab_key", req.VocabKey, "ip", r.RemoteAddr)
		h.sendError(w, "Invalid vocab index in vocab_key", http.StatusBadRequest)
		return
	}

	story, err := models.GetStoryData(r.Context(), id, auth.GetUserID(r))
	if err != nil {
		h.log.Error("Failed to fetch story in CheckVocab", "error", err, "storyID", id)
		h.sendError(w, "Failed to fetch story", http.StatusInternalServerError)
		return
	}

	// Validate line number
	if lineIndex < 0 || lineIndex >= len(story.Content.Lines) {
		h.log.Warn("Invalid line number in CheckVocab", "lineNumber", lineIndex, "maxLines", len(story.Content.Lines), "ip", r.RemoteAddr)
		h.sendError(w, fmt.Sprintf("Invalid line number: %d", lineIndex), http.StatusBadRequest)
		return
	}

	line := story.Content.Lines[lineIndex]
	if len(line.Vocabulary) == 0 {
		h.log.Warn("No vocabulary on line in CheckVocab", "lineNumber", lineIndex, "ip", r.RemoteAddr)
		h.sendError(w, fmt.Sprintf("No vocabulary on line: %d", lineIndex), http.StatusBadRequest)
		return
	}

	// Validate vocab index
	if vocabIndex < 0 || vocabIndex >= len(line.Vocabulary) {
		h.log.Warn("Invalid vocab index in CheckVocab", "vocabIndex", vocabIndex, "maxVocab", len(line.Vocabulary), "ip", r.RemoteAddr)
		h.sendError(w, fmt.Sprintf("Invalid vocab index: %d", vocabIndex), http.StatusBadRequest)
		return
	}

	// Check if the answer is correct
	expectedAnswer := line.Vocabulary[vocabIndex].LexicalForm
	isCorrect := req.Answer == expectedAnswer

	// Save individual vocab score
	userID := auth.GetUserID(r)
	if userID == "" {
		h.sendError(w, "Authentication required", http.StatusUnauthorized)
		return
	}
	incorrectAnswer := ""
	if !isCorrect {
		incorrectAnswer = req.Answer
	}
	if err := models.SaveVocabScore(ctx, userID, id, lineIndex, vocabIndex, isCorrect, incorrectAnswer); err != nil {
		h.log.Error("Failed to save vocab score", "error", err, "userID", userID, "storyID", id, "line", lineIndex)
		h.sendError(w, "Failed to save vocab score", http.StatusInternalServerError)
		return
	}

	// Check if all vocab items on this line are now complete (if answer was correct)
	var originalLine *string
	allLineComplete := false
	if isCorrect {
		// Check if all vocab on line is completed by user
		allLineComplete, err = models.CheckAllVocabCompleteForLine(r.Context(), userID, id, lineIndex)
		if err == nil && allLineComplete {
			originalLine = &line.Text
		}
	}

	responseData := types.CheckVocabResponse{
		Correct:      isCorrect,
		LineComplete: allLineComplete,
		OriginalLine: originalLine,
	}

	response := types.APIResponse{
		Success: true,
		Data:    responseData,
	}

	json.NewEncoder(w).Encode(response)
}
