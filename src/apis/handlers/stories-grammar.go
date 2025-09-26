package handlers

import (
	"encoding/json"
	"fmt"
	"glossias/src/apis/types"
	"glossias/src/auth"
	"glossias/src/pkg/models"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
)

// GetGrammarPage returns JSON data for story page 3 (grammar)
func (h *Handler) GetGrammarPage(w http.ResponseWriter, r *http.Request) {
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

	// TODO: Send only the first grammar point instead of blank story
	// For now, return blank lines - in the future this will iterate through grammar points
	lines := make([]types.Line, len(story.Content.Lines))
	for i, line := range story.Content.Lines {
		lines[i] = types.Line{
			Text:              []string{line.Text},
			AudioFiles:        []types.AudioFile{},
			HasVocabOrGrammar: len(line.Grammar) > 0,
		}
	}

	data := types.GrammarPageData{
		PageData: types.PageData{
			StoryID:    storyID,
			StoryTitle: story.Metadata.Title["en"],
			Lines:      lines,
		},
		GrammarPoint: "TODO: First grammar point name and description",
	}

	response := types.APIResponse{
		Success: true,
		Data:    data,
	}

	json.NewEncoder(w).Encode(response)
}

// CheckGrammar handles grammar checking for multiple lines of the same grammar point
func (h *Handler) CheckGrammar(w http.ResponseWriter, r *http.Request) {
	var req types.CheckGrammarRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.log.Warn("Invalid request body in CheckGrammar", "error", err, "ip", r.RemoteAddr)
		h.sendError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	storyID := mux.Vars(r)["id"]
	id, err := strconv.Atoi(storyID)
	if err != nil {
		h.log.Warn("Invalid story ID in CheckGrammar", "storyID", storyID, "ip", r.RemoteAddr)
		h.sendError(w, "Invalid story ID", http.StatusBadRequest)
		return
	}

	story, err := models.GetStoryData(r.Context(), id, auth.GetUserID(r))
	if err != nil {
		h.log.Error("Failed to fetch story in CheckGrammar", "error", err, "storyID", id)
		h.sendError(w, "Failed to fetch story", http.StatusInternalServerError)
		return
	}

	// Count total grammar items matching the grammar point ID
	totalExpected := 0
	grammarItemsMap := make(map[int][]models.GrammarItem) // line -> grammar items

	for _, line := range story.Content.Lines {
		var lineItems []models.GrammarItem
		for _, grammar := range line.Grammar {
			if grammar.GrammarPointID != nil && *grammar.GrammarPointID == req.GrammarPointID {
				totalExpected++
				lineItems = append(lineItems, grammar)
			}
		}
		if len(lineItems) > 0 {
			grammarItemsMap[line.LineNumber] = lineItems
		}
	}

	// Count total user answers
	totalAnswered := 0
	for _, answer := range req.Answers {
		totalAnswered += len(answer.Positions)
	}

	// Check if answer count matches expected
	if totalAnswered != totalExpected {
		h.sendError(w, fmt.Sprintf("Expected %d grammar points but got %d answers", totalExpected, totalAnswered), http.StatusBadRequest)
		return
	}

	// Score the answers
	correct := 0
	wrong := 0
	var results []types.GrammarResult

	for _, answer := range req.Answers {
		// Validate line number
		if answer.LineNumber < 0 || answer.LineNumber >= len(story.Content.Lines) {
			h.sendError(w, fmt.Sprintf("Invalid line number: %d", answer.LineNumber), http.StatusBadRequest)
			return
		}

		lineItems := grammarItemsMap[answer.LineNumber]
		line := story.Content.Lines[answer.LineNumber]

		// Check each position against grammar items on this line
		for _, pos := range answer.Positions {
			found := false
			for _, item := range lineItems {
				if pos >= item.Position[0] && pos < item.Position[1] {
					correct++
					results = append(results, types.GrammarResult{
						LineNumber: answer.LineNumber,
						Position:   item.Position,
						Text:       item.Text,
						Correct:    true,
					})
					found = true
					break
				}
			}
			if !found {
				wrong++
				// For incorrect answers, we still need to show where they clicked
				results = append(results, types.GrammarResult{
					LineNumber: answer.LineNumber,
					Position:   [2]int{pos, pos + 1}, // Single character position
					Text:       string([]rune(line.Text)[pos : pos+1]),
					Correct:    false,
				})
			}
		}

		// Add any missed grammar items from this line as incorrect
		for _, item := range lineItems {
			covered := false
			for _, pos := range answer.Positions {
				if pos >= item.Position[0] && pos < item.Position[1] {
					covered = true
					break
				}
			}
			if !covered {
				// This grammar item wasn't selected
				results = append(results, types.GrammarResult{
					LineNumber: answer.LineNumber,
					Position:   item.Position,
					Text:       item.Text,
					Correct:    false,
				})
			}
		}
	}

	// Ensure correct + wrong = total answers
	if correct+wrong != totalExpected {
		h.log.Error("Score calculation error", "correct", correct, "wrong", wrong, "expected", totalExpected)
	}

	// Save scores if user is authenticated
	userID := auth.GetUserID(r)
	if userID != "" {
		lineScores := make(map[int]bool)
		for _, answer := range req.Answers {
			// Determine if this line was answered correctly (all positions correct)
			lineCorrect := true
			lineItems := grammarItemsMap[answer.LineNumber]

			if len(answer.Positions) != len(lineItems) {
				lineCorrect = false
			} else {
				for _, item := range lineItems {
					found := false
					for _, pos := range answer.Positions {
						if pos >= item.Position[0] && pos < item.Position[1] {
							found = true
							break
						}
					}
					if !found {
						lineCorrect = false
						break
					}
				}
			}
			lineScores[answer.LineNumber] = lineCorrect
		}

		if err := models.SaveGrammarScoresForPoint(r.Context(), userID, id, req.GrammarPointID, lineScores); err != nil {
			h.log.Error("Failed to save grammar scores", "error", err, "userID", userID, "storyID", id)
		}
	}

	response := types.APIResponse{
		Success: true,
		Data: types.CheckGrammarResponse{
			Correct:      correct,
			Wrong:        wrong,
			TotalAnswers: totalExpected,
			Results:      results,
		},
	}

	json.NewEncoder(w).Encode(response)
}
