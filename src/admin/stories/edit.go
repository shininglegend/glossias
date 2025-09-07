// glossias/src/admin/stories/edit.go
package stories

import (
	"encoding/json"
	"fmt"
	"glossias/src/pkg/models"
	"net/http"
	"strconv"
	"strings"

	"github.com/gorilla/mux"
)

// EditStoryResponse wraps the story data for the editor
type EditStoryResponse struct {
	Story   *models.Story `json:"story"`
	Success bool          `json:"success"`
	Error   string        `json:"error,omitempty"`
}

func (h *Handler) editStoryHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	storyID, err := strconv.Atoi(vars["id"])
	if err != nil {
		http.Error(w, "Invalid story ID", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case http.MethodGet:
		h.handleGetStory(w, r, storyID)
	case http.MethodPut:
		h.handleUpdateStory(w, r, storyID)
	case http.MethodPost:
		h.addStoryHandler(w, r)
	case http.MethodDelete:
		h.deleteStoryHandler(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *Handler) handleGetStory(w http.ResponseWriter, r *http.Request, storyID int) {
	// Fetch story from database
	story, err := models.GetStoryData(r.Context(), storyID)
	if err != nil {
		if err == models.ErrNotFound {
			http.Error(w, "Story not found", http.StatusNotFound)
			return
		}
		h.log.Error("Failed to fetch story", "error", err, "storyID", storyID)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(EditStoryResponse{
		Story:   story,
		Success: true,
	})
}

func (h *Handler) handleUpdateStory(w http.ResponseWriter, r *http.Request, storyID int) {
	// Parse request body
	var story models.Story
	if err := json.NewDecoder(r.Body).Decode(&story); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
		return
	}

	// Ensure story ID matches
	if story.Metadata.StoryID != storyID {
		http.Error(w, "Story ID mismatch", http.StatusBadRequest)
		return
	}

	// Validate story data
	if err := validateStory(&story); err != nil {
		http.Error(w, fmt.Sprintf("Invalid story data: %v", err), http.StatusBadRequest)
		return
	}

	// Update story in database
	if err := models.SaveStoryData(r.Context(), storyID, &story); err != nil {
		h.log.Error("Failed to update story", "error", err, "storyID", storyID)
		http.Error(w, "Failed to update story", http.StatusInternalServerError)
		return
	}

	// Return success response

	json.NewEncoder(w).Encode(EditStoryResponse{
		Success: true,
		Story:   &story,
	})
}

func validateStory(story *models.Story) error {
	if story == nil {
		return fmt.Errorf("story cannot be nil")
	}

	// Validate required fields
	if story.Metadata.WeekNumber < 0 {
		return fmt.Errorf("invalid week number")
	}
	if !strings.Contains("abcde", story.Metadata.DayLetter) {
		return fmt.Errorf("invalid day letter")
	}
	if story.Metadata.Author.Name == "" {
		return fmt.Errorf("author name required")
	}
	if len(story.Content.Lines) == 0 {
		return fmt.Errorf("story must contain at least one line")
	}

	// Validate lines
	for i, line := range story.Content.Lines {
		if line.LineNumber != i+1 {
			return fmt.Errorf("invalid line numbering at line %d", i+1)
		}
		if strings.TrimSpace(line.Text) == "" {
			return fmt.Errorf("empty line at position %d", i+1)
		}
	}

	return nil
}
