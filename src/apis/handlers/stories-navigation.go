package handlers

import (
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
	PageTypeVideo     = PageType{Path: "video", DisplayName: "Watch"}
	PageTypeVocab     = PageType{Path: "vocab", DisplayName: "Vocabulary"}
	PageTypeIdentify  = PageType{Path: "identify", DisplayName: "Identify"}
	PageTypeTranslate = PageType{Path: "translate", DisplayName: "Translate"}
	PageTypeProduce   = PageType{Path: "produce", DisplayName: "Produce"}
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

// NavigationGuidanceResponse represents the response structure.
// CompletedAttempts lets the Video page offer the Score page to a student who
// has already finished the story, since finishing wipes the live rows that
// NextPage is computed from.
type NavigationGuidanceResponse struct {
	NextPage          string `json:"nextPage"`
	DisplayName       string `json:"displayName"`
	CompletedAttempts int    `json:"completedAttempts"`
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

	completion, err := models.GetUserStoryPageCompletion(r.Context(), userID, storyID)
	if err != nil {
		h.log.Error("Failed to get completion status", "error", err, "storyID", storyID, "userID", userID)
		h.sendError(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	nextPage := h.determineNextPage(req.CurrentPage, completion)

	response := types.APIResponse{
		Success: true,
		Data: NavigationGuidanceResponse{
			NextPage:          nextPage.Path,
			DisplayName:       nextPage.DisplayName,
			CompletedAttempts: completion.CompletedAttempts,
		},
	}

	// h.log.Info("Navigation determined", "userID", userID, "storyID", storyID, "currentPage", req.CurrentPage, "nextPage", nextPage.Path)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// pageCompletionStatus maps one completion row onto every page type. Video
// and Score are never "complete": video is always visited and score is the
// terminal page.
func pageCompletionStatus(c *models.PageCompletion) map[PageType]bool {
	return map[PageType]bool{
		PageTypeVideo:     false,
		PageTypeVocab:     false, // not in the S26 flow; never skipped-to
		PageTypeIdentify:  c.IdentifyComplete(),
		PageTypeGrammar:   false, // not in the S26 flow; never skipped-to
		PageTypeTranslate: c.TranslateComplete(),
		PageTypeProduce:   c.ProduceComplete(),
		PageTypeRecall:    c.RecallComplete(),
		PageTypeScore:     false,
	}
}

func storyListProgress(c *models.PageCompletion) (status string, next PageType) {
	next = determineNextPageFrom("list", c)
	switch {
	case c.FlowComplete():
		return "complete", next
	case c.LaterPhaseStarted():
		return "in_progress", next
	default:
		return "not_started", next
	}
}

func (h *Handler) determineNextPage(currentPage string, c *models.PageCompletion) PageType {
	return determineNextPageFrom(currentPage, c)
}

func determineNextPageFrom(currentPage string, c *models.PageCompletion) PageType {
	completionStatus := pageCompletionStatus(c)

	// List resume: unstarted live rows always open Watch, even when archived
	// attempts exist. Once a later phase has progress, skip Watch the same
	// way Continue-from-Video does.
	if currentPage == "list" {
		if !c.LaterPhaseStarted() {
			return PageTypeVideo
		}
		currentPage = PageTypeVideo.Path
	}

	currentIndex := -1
	for i, page := range defaultPageOrder {
		if page.Path == currentPage {
			currentIndex = i
			break
		}
	}

	if currentIndex == -1 {
		return PageTypeVideo
	}

	// Resume from Video jumps to the first incomplete phase. Continue from an
	// exercise page (Identify, Translate, …) advances to the next page in
	// order so a completed Translate is not skipped after Identify.
	skipCompleted := currentPage == PageTypeVideo.Path

	for i := currentIndex + 1; i < len(defaultPageOrder); i++ {
		page := defaultPageOrder[i]

		if page.Path == "video" || !skipCompleted || !completionStatus[page] {
			return page
		}
	}

	return PageTypeScore
}
