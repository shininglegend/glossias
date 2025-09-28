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

	// Parse optional grammar_point_id query parameter
	var targetGrammarPointID *int
	if gpIDStr := r.URL.Query().Get("grammar_point_id"); gpIDStr != "" {
		if gpID, err := strconv.Atoi(gpIDStr); err != nil {
			h.sendError(w, "Invalid grammar_point_id format", http.StatusBadRequest)
			return
		} else {
			targetGrammarPointID = &gpID
		}
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

	// Get grammar points for the story
	grammarPoints, err := models.GetStoryGrammarPoints(r.Context(), id)
	if err != nil {
		h.log.Error("Failed to fetch grammar points", "error", err)
		h.sendError(w, "Failed to fetch grammar points", http.StatusInternalServerError)
		return
	}

	// Find target grammar point
	var selectedPoint *models.GrammarPoint
	if targetGrammarPointID != nil {
		// Find specific grammar point by ID
		for _, gp := range grammarPoints {
			if gp.ID == *targetGrammarPointID {
				selectedPoint = &gp
				break
			}
		}
		if selectedPoint == nil {
			h.sendError(w, "Grammar point not found", http.StatusNotFound)
			return
		}
	} else if len(grammarPoints) > 0 {
		// Use first grammar point if no specific ID provided
		selectedPoint = &grammarPoints[0]
	}

	var grammarPointID int
	var grammarPoint string
	var grammarDescription string
	var instancesCount int

	if selectedPoint != nil {
		grammarPointID = selectedPoint.ID
		grammarPoint = selectedPoint.Name
		grammarDescription = selectedPoint.Description

		// Count instances of this grammar point in the story
		for _, line := range story.Content.Lines {
			for _, grammar := range line.Grammar {
				if grammar.GrammarPointID != nil && *grammar.GrammarPointID == selectedPoint.ID {
					instancesCount++
				}
			}
		}
	}

	lines := make([]types.LineText, len(story.Content.Lines))
	for i, line := range story.Content.Lines {
		lines[i] = types.LineText{
			Text: line.Text,
		}
	}

	data := types.GrammarPageData{
		PageData: types.PageData{
			StoryID:    storyID,
			StoryTitle: story.Metadata.Title["en"],
		},
		Lines:              lines,
		LanguageCode:       story.Metadata.Description.Language,
		GrammarPointID:     grammarPointID,
		GrammarPoint:       grammarPoint,
		GrammarDescription: grammarDescription,
		InstancesCount:     instancesCount,
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
		h.sendError(w, fmt.Sprintf("Expected %d grammar points but got %d answers. If expected == 0, then check both grammar point ID and story", totalExpected, totalAnswered), http.StatusBadRequest)
		return
	}

	// Create user selection map for quick lookup
	userSelections := make(map[int]map[int]bool) // lineNumber -> position -> selected
	for _, answer := range req.Answers {
		// Validate line number (convert from 1-based to 0-based)
		if answer.LineNumber < 1 || answer.LineNumber > len(story.Content.Lines) {
			h.sendError(w, fmt.Sprintf("Invalid line number: %d", answer.LineNumber), http.StatusBadRequest)
			return
		}

		if userSelections[answer.LineNumber] == nil {
			userSelections[answer.LineNumber] = make(map[int]bool)
		}
		for _, pos := range answer.Positions {
			userSelections[answer.LineNumber][pos] = true
		}
	}

	// Build grammar instances (all actual grammar points in the story)
	var grammarInstances []types.GrammarInstance
	correct := 0
	wrong := 0

	for lineNum, grammarItems := range grammarItemsMap {
		for _, item := range grammarItems {
			// Check if user selected this grammar item
			userSelected := false
			if lineSelections, exists := userSelections[lineNum]; exists {
				for pos := range lineSelections {
					if pos >= item.Position[0] && pos < item.Position[1] {
						userSelected = true
						break
					}
				}
			}

			grammarInstances = append(grammarInstances, types.GrammarInstance{
				LineNumber:   lineNum,
				Position:     item.Position,
				Text:         item.Text,
				UserSelected: userSelected,
			})

			if userSelected {
				correct++
			} else {
				wrong++
			}
		}
	}

	// Build user selections (what the user actually clicked)
	var userSelectionResults []types.UserSelection
	for lineNum, lineSelections := range userSelections {
		line := story.Content.Lines[lineNum-1]
		grammarItems := grammarItemsMap[lineNum]

		for pos := range lineSelections {
			// Check if this position matches any grammar item
			isCorrect := false
			matchedText := ""
			matchedPosition := [2]int{pos, pos + 1}

			for _, item := range grammarItems {
				if pos >= item.Position[0] && pos < item.Position[1] {
					isCorrect = true
					matchedText = item.Text
					matchedPosition = item.Position
					break
				}
			}

			if !isCorrect {
				// Get the character the user clicked on
				matchedText = string([]rune(line.Text)[pos : pos+1])
			}

			userSelectionResults = append(userSelectionResults, types.UserSelection{
				LineNumber: lineNum,
				Position:   matchedPosition,
				Text:       matchedText,
				Correct:    isCorrect,
			})
		}
	}

	// Ensure correct + wrong = total answers
	if correct+wrong != totalExpected {
		h.log.Error("Score calculation error", "correct", correct, "wrong", wrong, "expected", totalExpected)
	}

	// Find next grammar point
	var nextGrammarPointID *int
	grammarPoints, err := models.GetStoryGrammarPoints(r.Context(), id)
	if err == nil {
		// Find current grammar point index and get next one
		for i, gp := range grammarPoints {
			if gp.ID == req.GrammarPointID && i+1 < len(grammarPoints) {
				nextGrammarPointID = &grammarPoints[i+1].ID
				break
			}
		}
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
			Correct:            correct,
			Wrong:              wrong,
			TotalAnswers:       totalExpected,
			GrammarInstances:   grammarInstances,
			UserSelections:     userSelectionResults,
			NextGrammarPointID: nextGrammarPointID,
		},
	}

	json.NewEncoder(w).Encode(response)
}
