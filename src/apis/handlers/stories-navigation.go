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
	PageTypeIdentify  = PageType{Path: "identify", DisplayName: "Identify"}
	PageTypeTranslate = PageType{Path: "translate", DisplayName: "Translation"}
	PageTypeProduce   = PageType{Path: "produce", DisplayName: "Production"}
	PageTypeRecall    = PageType{Path: "recall", DisplayName: "Recall"}
	PageTypeGrammar   = PageType{Path: "grammar", DisplayName: "Grammar"}
	PageTypeScore     = PageType{Path: "score", DisplayName: "Score"}
)

// Summer 2026 flow (SUMMER_2026.md). Vocab and Grammar pages remain reachable
// by URL but are not part of the flow: Identify and Produce replaced them.
var defaultPageOrder = []PageType{
	PageTypeVideo,
	PageTypeIdentify,
	PageTypeTranslate,
	PageTypeProduce,
	PageTypeRecall,
	PageTypeScore,
}

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
	completionStatus, err := h.getPageCompletionStatus(r.Context(), userID, storyID)
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

	// h.log.Info("Navigation determined", "userID", userID, "storyID", storyID, "currentPage", req.CurrentPage, "nextPage", nextPage.Path)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// getPageCompletionStatus returns completion status for every page type,
// from a single query. Video and Score are never "complete": video is always
// visited and score is the terminal page.
func (h *Handler) getPageCompletionStatus(ctx context.Context, userID string, storyID int) (map[PageType]bool, error) {
	c, err := models.GetUserStoryPageCompletion(ctx, userID, storyID)
	if err != nil {
		return nil, err
	}
	return map[PageType]bool{
		PageTypeVideo:     false,
		PageTypeVocab:     false, // not in the S26 flow; never skipped-to
		PageTypeIdentify:  c.IdentifyComplete(),
		PageTypeGrammar:   false, // not in the S26 flow; never skipped-to
		PageTypeTranslate: c.TranslateComplete(),
		PageTypeProduce:   c.ProduceComplete(),
		PageTypeRecall:    c.RecallComplete(),
		PageTypeScore:     false,
	}, nil
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
