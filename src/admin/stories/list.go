// glossias/src/admin/stories/list.go
package stories

import (
	"encoding/json"
	"glossias/src/pkg/models"
	"net/http"
)

type StoryListItem struct {
	ID         int    `json:"id"`
	Title      string `json:"title"`
	WeekNumber int    `json:"week_number"`
	DayLetter  string `json:"day_letter"`
	CourseID   *int   `json:"course_id,omitempty"`
	CourseName string `json:"course_name,omitempty"`
}

type ListResponse struct {
	Success bool            `json:"success"`
	Data    ListDataWrapper `json:"data"`
}

type ListDataWrapper struct {
	Stories []StoryListItem `json:"stories"`
}

// listStoriesHandler returns all stories that the user is allowed to manage
// GET /api/admin/stories
func (h *Handler) listStoriesHandler(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("userID").(string)

	stories, err := models.GetAllStoriesForAdmin(r.Context(), userID)
	if err != nil {
		h.log.Error("Failed to get stories for admin", "error", err)
		http.Error(w, "Failed to fetch stories", http.StatusInternalServerError)
		return
	}

	// Convert to response format
	listItems := make([]StoryListItem, 0, len(stories))
	for _, story := range stories {
		// Get English title, fallback to first available
		title := story.Metadata.Title["en"]
		if title == "" {
			for _, t := range story.Metadata.Title {
				title = t
				break
			}
		}

		item := StoryListItem{
			ID:         story.Metadata.StoryID,
			Title:      title,
			WeekNumber: story.Metadata.WeekNumber,
			DayLetter:  story.Metadata.DayLetter,
			CourseID:   story.Metadata.CourseID,
			CourseName: story.Metadata.CourseName,
		}
		listItems = append(listItems, item)
	}

	response := ListResponse{
		Success: true,
		Data: ListDataWrapper{
			Stories: listItems,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
