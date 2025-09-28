package handlers

import (
	"context"
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
			Language:   story.Metadata.Description.Language,
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

// buildCorrectGrammarMap creates mapping of grammar items by line number
func buildCorrectGrammarMap(story *models.Story, grammarPointID int) (map[int][]models.GrammarItem, int, error) {
	grammarItemsMap := make(map[int][]models.GrammarItem)
	totalExpected := 0

	for _, line := range story.Content.Lines {
		var lineItems []models.GrammarItem
		for _, grammar := range line.Grammar {
			if grammar.GrammarPointID != nil && *grammar.GrammarPointID == grammarPointID {
				totalExpected++
				lineItems = append(lineItems, grammar)
			}
		}
		if len(lineItems) > 0 {
			grammarItemsMap[line.LineNumber] = lineItems
		}
	}

	return grammarItemsMap, totalExpected, nil
}

// buildUserSelections creates mapping of user selections for quick lookup
func buildUserSelections(answers []types.GrammarAnswer, storyLineCount int) (map[int]map[int]bool, int, error) {
	userSelections := make(map[int]map[int]bool)
	totalAnswered := 0

	for _, answer := range answers {
		if answer.LineNumber < 1 || answer.LineNumber > storyLineCount {
			return nil, 0, fmt.Errorf("invalid line number: %d", answer.LineNumber)
		}

		if userSelections[answer.LineNumber] == nil {
			userSelections[answer.LineNumber] = make(map[int]bool)
		}
		for _, pos := range answer.Positions {
			userSelections[answer.LineNumber][pos] = true
		}
		totalAnswered += len(answer.Positions)
	}

	return userSelections, totalAnswered, nil
}

// buildGrammarInstances creates grammar instances and calculates correct/wrong counts
func buildGrammarInstances(grammarItemsMap map[int][]models.GrammarItem, userSelections map[int]map[int]bool) ([]types.GrammarInstance, int, int, error) {
	var grammarInstances []types.GrammarInstance
	correct := 0
	wrong := 0

	for lineNum, grammarItems := range grammarItemsMap {
		for _, item := range grammarItems {
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

	return grammarInstances, correct, wrong, nil
}

// buildUserSelectionResults creates results for what user clicked
func buildUserSelectionResults(userSelections map[int]map[int]bool, grammarItemsMap map[int][]models.GrammarItem, story *models.Story) ([]types.UserSelection, error) {
	var userSelectionResults []types.UserSelection

	for lineNum, lineSelections := range userSelections {
		line := story.Content.Lines[lineNum-1]
		grammarItems := grammarItemsMap[lineNum]

		for pos := range lineSelections {
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

	return userSelectionResults, nil
}

// findNextGrammarPoint finds the next grammar point ID in sequence
func findNextGrammarPoint(ctx context.Context, storyID, currentGrammarPointID int) (*int, error) {
	grammarPoints, err := models.GetStoryGrammarPoints(ctx, storyID)
	if err != nil {
		return nil, err
	}

	for i, gp := range grammarPoints {
		if gp.ID == currentGrammarPointID && i+1 < len(grammarPoints) {
			return &grammarPoints[i+1].ID, nil
		}
	}

	return nil, nil
}

// saveUserScores saves grammar scores for authenticated user
func saveUserScores(ctx context.Context, userID string, storyID, grammarPointID int, answers []types.GrammarAnswer, correctGrammarMap map[int][]models.GrammarItem) error {
	if userID == "" {
		return nil
	}

	var correctAnswers []struct {
		LineNumber int
		Position   [2]int
		Text       string
	}
	var incorrectAnswers []struct {
		LineNumber int
		Position   int
	}

	fmt.Printf("[DEBUG] Processing %d user answer lines\n", len(answers))

	// Loop over user's answers
	for _, answer := range answers {
		fmt.Printf("[DEBUG] Processing line %d with positions: %v\n", answer.LineNumber, answer.Positions)

		correctItemsForLine, hasCorrectItems := correctGrammarMap[answer.LineNumber]
		if !hasCorrectItems {
			// Line has no correct grammar points - all answers are incorrect
			fmt.Printf("[DEBUG] Line %d has no correct items - all %d positions incorrect\n", answer.LineNumber, len(answer.Positions))
			for _, pos := range answer.Positions {
				incorrectAnswers = append(incorrectAnswers, struct {
					LineNumber int
					Position   int
				}{
					LineNumber: answer.LineNumber,
					Position:   pos,
				})
			}
		} else {
			// Line has correct grammar points - check each user position
			remainingCorrectItems := make([]models.GrammarItem, len(correctItemsForLine))
			copy(remainingCorrectItems, correctItemsForLine)

			fmt.Printf("[DEBUG] Line %d has %d correct items to match against\n", answer.LineNumber, len(remainingCorrectItems))

			for _, userPos := range answer.Positions {
				matched := false

				// Check if this position matches any remaining correct item
				for i, correctItem := range remainingCorrectItems {
					if userPos >= correctItem.Position[0] && userPos < correctItem.Position[1] {
						fmt.Printf("[DEBUG] Position %d matches correct item '%s' [%d,%d)\n",
							userPos, correctItem.Text, correctItem.Position[0], correctItem.Position[1])

						// Add to correct answers
						correctAnswers = append(correctAnswers, struct {
							LineNumber int
							Position   [2]int
							Text       string
						}{
							LineNumber: answer.LineNumber,
							Position:   correctItem.Position,
							Text:       correctItem.Text,
						})

						// Remove this item from remaining to avoid dupes
						remainingCorrectItems = append(remainingCorrectItems[:i], remainingCorrectItems[i+1:]...)
						matched = true
						break
					}
				}

				if !matched {
					fmt.Printf("[DEBUG] Position %d on line %d is incorrect\n", userPos, answer.LineNumber)
					incorrectAnswers = append(incorrectAnswers, struct {
						LineNumber int
						Position   int
					}{
						LineNumber: answer.LineNumber,
						Position:   userPos,
					})
				}
			}
		}
	}

	fmt.Printf("[DEBUG] Final results: %d correct, %d incorrect\n", len(correctAnswers), len(incorrectAnswers))

	// Save correct answers
	if len(correctAnswers) > 0 {
		if err := models.SaveCorrectAnswers(ctx, userID, storyID, grammarPointID, correctAnswers); err != nil {
			return err
		}
	}

	// Save incorrect answers
	if len(incorrectAnswers) > 0 {
		if err := models.SaveIncorrectAnswers(ctx, userID, storyID, grammarPointID, incorrectAnswers); err != nil {
			return err
		}
	}

	return nil
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

	correctGrammarMap, totalExpected, err := buildCorrectGrammarMap(story, req.GrammarPointID)
	if err != nil {
		h.log.Error("Failed to build correct grammar map", "error", err)
		h.sendError(w, "Internal error", http.StatusInternalServerError)
		return
	}

	userSelections, totalAnswered, err := buildUserSelections(req.Answers, len(story.Content.Lines))
	if err != nil {
		h.log.Warn("Invalid user selections", "error", err)
		h.sendError(w, err.Error(), http.StatusBadRequest)
		return
	}

	if totalAnswered != totalExpected {
		h.sendError(w, fmt.Sprintf("Expected %d grammar points but got %d answers. If expected == 0, then check both grammar point ID and story", totalExpected, totalAnswered), http.StatusBadRequest)
		return
	}

	grammarInstances, correct, wrong, err := buildGrammarInstances(correctGrammarMap, userSelections)
	if err != nil {
		h.log.Error("Failed to build grammar instances", "error", err)
		h.sendError(w, "Internal error", http.StatusInternalServerError)
		return
	}

	userSelectionResults, err := buildUserSelectionResults(userSelections, correctGrammarMap, story)
	if err != nil {
		h.log.Error("Failed to build user selection results", "error", err)
		h.sendError(w, "Internal error", http.StatusInternalServerError)
		return
	}

	if correct+wrong != totalExpected {
		h.log.Error("Score calculation error", "correct", correct, "wrong", wrong, "expected", totalExpected)
	}

	nextGrammarPointID, err := findNextGrammarPoint(r.Context(), id, req.GrammarPointID)
	if err != nil {
		h.log.Warn("Failed to find next grammar point", "error", err)
	}

	if err := saveUserScores(r.Context(), auth.GetUserID(r), id, req.GrammarPointID, req.Answers, correctGrammarMap); err != nil {
		h.log.Error("Failed to save grammar scores", "error", err, "userID", auth.GetUserID(r), "storyID", id)
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
