package handlers

import (
	"context"
	"encoding/json"
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
	foundInstances := []types.GrammarInstance{}
	incorrectInstances := []types.UserSelection{}
	var nextGrammarPointID *int

	if selectedPoint != nil {
		grammarPointID = selectedPoint.ID
		grammarPoint = selectedPoint.Name
		grammarDescription = selectedPoint.Description

		// Build grammar map for this grammar point
		grammarItemsMap, totalExpected, err := buildCorrectGrammarMap(story, selectedPoint.ID)
		if err != nil {
			h.log.Error("Failed to build grammar map", "error", err)
			h.sendError(w, "Failed to process grammar data", http.StatusInternalServerError)
			return
		}
		instancesCount = totalExpected

		userID := auth.GetUserID(r)
		if userID == "" {
			h.sendError(w, "Authentication required", http.StatusUnauthorized)
			return
		}
		// Get user's correct answers for this grammar point
		correctAnswers, err := models.GetUserGrammarScoresByGrammarPoint(r.Context(), userID, id, selectedPoint.ID)
		if err != nil {
			h.log.Warn("Failed to get user grammar scores", "error", err)
		} else {
			// Build found instances from user's correct answers
			for _, answer := range correctAnswers {
				if lineGrammar, exists := grammarItemsMap[int(answer.LineNumber)]; exists {
					for _, item := range lineGrammar {
						foundInstances = append(foundInstances, types.GrammarInstance{
							LineNumber: int(answer.LineNumber),
							Position:   item.Position,
							Text:       item.Text,
						})
					}
				}
			}
		}

		// Get user's incorrect answers for this grammar point
		incorrectAnswers, err := models.GetUserGrammarIncorrectAnswers(r.Context(), userID, id, selectedPoint.ID)
		if err != nil {
			h.log.Warn("Failed to get user incorrect answers", "error", err)
		} else {
			// Build incorrect instances from user's wrong answers
			for _, answer := range incorrectAnswers {
				for _, pos := range answer.SelectedPositions {
					matchedText := ""
					if int(answer.SelectedLine) > 0 && int(answer.SelectedLine) <= len(story.Content.Lines) {
						line := story.Content.Lines[int(answer.SelectedLine)-1]
						runes := []rune(line.Text)
						if pos >= 0 && int(pos) < len(runes) {
							matchedText = string(runes[pos])
						}
					}
					incorrectInstances = append(incorrectInstances, types.UserSelection{
						LineNumber: int(answer.SelectedLine),
						Position:   [2]int{int(pos), int(pos) + 1},
						Text:       matchedText,
						Correct:    false,
					})
				}
			}
		}

		// Check if all instances found and get next grammar point
		if len(foundInstances) >= instancesCount {
			nextGrammarPointID, _ = findNextGrammarPoint(r.Context(), id, selectedPoint.ID)
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
			Language:   story.Metadata.Language,
		},
		Lines:              lines,
		LanguageCode:       story.Metadata.Language,
		GrammarPointID:     grammarPointID,
		GrammarPoint:       grammarPoint,
		GrammarDescription: grammarDescription,
		InstancesCount:     instancesCount,
		FoundInstances:     foundInstances,
		IncorrectInstances: incorrectInstances,
		NextGrammarPoint:   nextGrammarPointID,
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

// findNextGrammarPointFromSlice finds the next grammar point ID in sequence from provided slice
func findNextGrammarPointFromSlice(grammarPoints []models.GrammarPoint, currentGrammarPointID int) *int {
	for i, gp := range grammarPoints {
		if gp.ID == currentGrammarPointID && i+1 < len(grammarPoints) {
			return &grammarPoints[i+1].ID
		}
	}
	return nil
}

// findNextGrammarPoint finds the next grammar point ID in sequence (legacy function)
func findNextGrammarPoint(ctx context.Context, storyID, currentGrammarPointID int) (*int, error) {
	grammarPoints, err := models.GetStoryGrammarPoints(ctx, storyID)
	if err != nil {
		return nil, err
	}
	return findNextGrammarPointFromSlice(grammarPoints, currentGrammarPointID), nil
}

// CheckGrammar handles single grammar selection checking (one click at a time)
func (h *Handler) CheckGrammar(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := auth.GetUserID(r)
	if userID == "" {
		h.sendError(w, "Authentication required", http.StatusUnauthorized)
		return
	}

	var req types.CheckSingleGrammarRequest
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

	// Get story data - cached, fastest call
	story, err := models.GetStoryData(ctx, id, userID)
	if err != nil {
		h.log.Error("Failed to fetch story data", "error", err, "storyID", id)
		h.sendError(w, "Failed to fetch story data", http.StatusInternalServerError)
		return
	}

	// Build grammar map for this grammar point - O(n) on story lines
	grammarItemsMap, totalInstances := buildCorrectGrammarMapOptimized(story, req.GrammarPointID)

	// Check if user clicked on a correct instance - O(1) hash lookup + small array scan
	isCorrect := false
	var matchedGrammar *models.GrammarItem
	if lineGrammar, exists := grammarItemsMap[req.LineNumber]; exists {
		for i := range lineGrammar {
			item := &lineGrammar[i]
			if req.Position >= item.Position[0] && req.Position < item.Position[1] {
				isCorrect = true
				matchedGrammar = item
				break
			}
		}
	}

	matchedPosition := [2]int{req.Position, req.Position + 1}
	if isCorrect && matchedGrammar != nil {
		matchedPosition = matchedGrammar.Position
	}

	// Save the score - cannot avoid this DB call
	if err := models.SaveSingleGrammarSelection(ctx, userID, id, req.GrammarPointID, req.LineNumber, req.Position, isCorrect); err != nil {
		h.log.Error("Failed to save grammar selection", "error", err, "userID", userID, "storyID", id)
	}

	// Get current found count and check completion - single DB call
	foundCount, err := models.CountFoundGrammarInstances(ctx, userID, id, req.GrammarPointID)
	if err != nil {
		h.log.Error("Failed to count found instances", "error", err, "userID", userID, "storyID", id)
		// Continue without count rather than failing
		foundCount = 0
	}

	allComplete := foundCount >= totalInstances

	// Find next grammar point if all complete - optimized to avoid extra DB call
	var nextGrammarPointID *int
	if allComplete {
		grammarPoints, err := models.GetStoryGrammarPoints(ctx, id)
		if err != nil {
			h.log.Error("Failed to get grammar points for next lookup", "error", err, "storyID", id)
		} else {
			nextGrammarPointID = findNextGrammarPointFromSlice(grammarPoints, req.GrammarPointID)
		}
	}

	response := types.APIResponse{
		Success: true,
		Data: types.CheckSingleGrammarResponse{
			Correct:          isCorrect,
			MatchedPosition:  matchedPosition,
			TotalInstances:   totalInstances,
			NextGrammarPoint: nextGrammarPointID,
		},
	}

	json.NewEncoder(w).Encode(response)
}

// buildCorrectGrammarMapOptimized creates mapping of grammar items by line number (optimized version)
func buildCorrectGrammarMapOptimized(story *models.Story, grammarPointID int) (map[int][]models.GrammarItem, int) {
	grammarItemsMap := make(map[int][]models.GrammarItem)
	totalExpected := 0

	// Single pass through story lines
	for _, line := range story.Content.Lines {
		for _, grammar := range line.Grammar {
			if grammar.GrammarPointID != nil && *grammar.GrammarPointID == grammarPointID {
				totalExpected++
				grammarItemsMap[line.LineNumber] = append(grammarItemsMap[line.LineNumber], grammar)
			}
		}
	}

	return grammarItemsMap, totalExpected
}
