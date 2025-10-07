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

// PageType represents different page types in the learning flow
type PageType struct {
	Path        string `json:"path"`
	DisplayName string `json:"displayName"`
}

var (
	PageTypeVideo     = PageType{Path: "video", DisplayName: "Video"}
	PageTypeVocab     = PageType{Path: "vocab", DisplayName: "Vocabulary"}
	PageTypeTranslate = PageType{Path: "translate", DisplayName: "Translation"}
	PageTypeGrammar   = PageType{Path: "grammar", DisplayName: "Grammar"}
	PageTypeScore     = PageType{Path: "score", DisplayName: "Score"}
)

// Default page order for MVP
var defaultPageOrder = []PageType{
	PageTypeVideo,
	PageTypeVocab,
	PageTypeTranslate,
	PageTypeGrammar,
	PageTypeScore,
}

const minTimeSeconds = 0 // Minimum time in seconds to consider a page "completed" (unused)

// NavigationGuidanceRequest represents the request structure
type NavigationGuidanceRequest struct {
	CurrentPage string `json:"currentPage"`
}

// NavigationGuidanceResponse represents the response structure
type NavigationGuidanceResponse struct {
	NextPage    string `json:"nextPage"`
	DisplayName string `json:"displayName"`
}

// Navigate determines the next page a user should visit
func (h *Handler) Navigate(w http.ResponseWriter, r *http.Request) {
	// Get story ID from URL
	storyIDStr := mux.Vars(r)["id"]
	storyID, err := strconv.Atoi(storyIDStr)
	if err != nil {
		h.sendError(w, "Invalid story ID format", http.StatusBadRequest)
		return
	}

	// Parse JSON request
	var req NavigationGuidanceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.sendError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Get user ID from auth context
	userID := auth.GetUserID(r)
	if userID == "" {
		h.sendError(w, "User not authenticated", http.StatusUnauthorized)
		return
	}

	// Validate story exists
	_, err = models.GetStoryData(r.Context(), storyID, userID)
	if err == models.ErrNotFound {
		h.sendError(w, "Story not found", http.StatusNotFound)
		return
	}
	if err != nil {
		h.log.Error("Failed to fetch story data", "error", err, "storyID", storyID)
		h.sendError(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Get completion status for all pages
	completionStatus, err := h.getPageCompletionStatus(r.Context(), userID, int32(storyID))
	if err != nil {
		h.log.Error("Failed to get completion status", "error", err, "storyID", storyID, "userID", userID)
		h.sendError(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Determine next page
	nextPage := h.determineNextPage(req.CurrentPage, completionStatus)

	response := types.APIResponse{
		Success: true,
		Data: NavigationGuidanceResponse{
			NextPage:    nextPage.Path,
			DisplayName: nextPage.DisplayName,
		},
	}

	h.log.Info("Navigation determined", "userID", userID, "storyID", storyID, "currentPage", req.CurrentPage, "nextPage", nextPage.Path)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// getPageCompletionStatus returns completion status for all page types
func (h *Handler) getPageCompletionStatus(ctx context.Context, userID string, storyID int32) (map[PageType]bool, error) {
	status := make(map[PageType]bool)

	// Video is never considered "complete" for skipping purposes
	status[PageTypeVideo] = false

	// Get vocab completion status
	vocabComplete, err := h.isVocabCompleted(ctx, userID, storyID)
	if err != nil {
		return nil, err
	}
	status[PageTypeVocab] = vocabComplete

	// Get grammar completion status
	grammarComplete, err := h.isGrammarCompleted(ctx, userID, storyID)
	if err != nil {
		return nil, err
	}
	status[PageTypeGrammar] = grammarComplete

	// Get translate completion status
	translateComplete, err := h.isTranslateCompleted(ctx, userID, storyID)
	if err != nil {
		return nil, err
	}
	status[PageTypeTranslate] = translateComplete

	// Score is never considered "complete" for skipping
	status[PageTypeScore] = false

	return status, nil
}

// isVocabCompleted checks if user has completed vocab (correct answers = total vocab items)
func (h *Handler) isVocabCompleted(ctx context.Context, userID string, storyID int32) (bool, error) {
	// Get total vocabulary items in story
	totalVocabItems, err := models.CountStoryVocabItems(ctx, storyID)
	if err != nil {
		return false, err
	}

	if totalVocabItems == 0 {
		return true, nil // No vocab items means complete
	}

	// Check user's correct answers
	vocabSummary, err := models.GetUserStoryVocabSummary(ctx, userID, storyID)
	if err != nil {
		return false, err
	}

	return vocabSummary.CorrectCount == totalVocabItems, nil
}

// isGrammarCompleted checks if user has completed grammar (correct answers == total instances AND sufficient time)
func (h *Handler) isGrammarCompleted(ctx context.Context, userID string, storyID int32) (bool, error) {
	// Get total grammar instances in story
	story, err := models.GetStoryData(ctx, int(storyID), userID)
	if err != nil {
		return false, err
	}

	// Count total grammar instances across all grammar points
	totalInstances := 0
	for _, line := range story.Content.Lines {
		totalInstances += len(line.Grammar)
	}

	if totalInstances == 0 {
		return true, nil // No grammar instances to find
	}

	// Check if user has found all instances
	grammarSummary, err := models.GetUserStoryGrammarSummary(ctx, userID, storyID)
	if err != nil {
		return false, err
	}

	if int(grammarSummary.CorrectCount) < totalInstances {
		return false, nil // Haven't found all instances yet
	}

	// Check time spent
	timeData, err := models.GetUserStoryTimeTracking(ctx, userID, storyID)
	if err != nil {
		return false, err
	}

	return timeData.GrammarTimeSeconds >= minTimeSeconds, nil
}

// isTranslateCompleted checks if user has spent sufficient time on translation
func (h *Handler) isTranslateCompleted(ctx context.Context, userID string, storyID int32) (bool, error) {
	timeData, err := models.GetUserStoryTimeTracking(ctx, userID, storyID)
	if err != nil {
		return false, err
	}

	return timeData.TranslationTimeSeconds >= minTimeSeconds, nil
}

// determineNextPage finds the next page to visit based on current page and completion status
func (h *Handler) determineNextPage(currentPage string, completionStatus map[PageType]bool) PageType {
	// Find current page index in the order
	currentIndex := -1
	for i, page := range defaultPageOrder {
		if page.Path == currentPage {
			currentIndex = i
			break
		}
	}

	// If current page not found in order, start from beginning
	if currentIndex == -1 {
		return PageTypeVideo
	}

	// Starting from next page, find first incomplete page
	for i := currentIndex + 1; i < len(defaultPageOrder); i++ {
		page := defaultPageOrder[i]

		// Video is always visited, others check completion status
		if page.Path == "video" || !completionStatus[page] {
			return page
		}
	}

	// All pages after current are complete, return score
	return PageTypeScore
}
