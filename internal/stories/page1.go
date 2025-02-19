package stories

import (
	"fmt"
	"glossias/internal/pkg/models"
	"net/http"
	"os"
	"strconv"

	"github.com/gorilla/mux"
)

// page1.go
func (h *Handler) ServePage1(w http.ResponseWriter, r *http.Request) {
	// Get the story id from the URL
	vars := mux.Vars(r)
	storyID := vars["id"]
	if storyID == "" {
		h.log.Info("Missing story ID", "story_id", storyID)
		http.Error(w, "Missing story ID", http.StatusBadRequest)
		return
	}

	// Convert string ID to int
	id, err := strconv.Atoi(storyID)
	if err != nil {
		h.log.Info("Invalid story ID format", "story_id", storyID)
		http.Error(w, "Invalid story ID format", http.StatusBadRequest)
		return
	}

	// Get story data from database
	story, err := models.GetStoryData(id)
	if err == models.ErrNotFound {
		h.log.Info("Story not found", "story_id", id)
		http.Error(w, "Story not found", http.StatusNotFound)
		return
	}
	if err != nil {
		h.log.Error("Failed to fetch story data", "error", err)
		http.Error(w, "Failed to fetch story data", http.StatusInternalServerError)
		return
	}

	// Load the audio files from the folder (keeping existing audio handling)
	folderPath := fmt.Sprintf(storiesDir+"stories_audio/%v_%v%v", story.Metadata.Description.Language, story.Metadata.WeekNumber, story.Metadata.DayLetter)
	audioDir, err := os.ReadDir(folderPath)
	if err != nil && !os.IsNotExist(err) {
		h.log.Error("Failed to read audio files", "error", err)
		http.Error(w, "Failed to read audio files", http.StatusInternalServerError)
		return
	}

	// Convert database story lines to template format
	lines := make([]Line, 0, len(story.Content.Lines))
	for i, dbLine := range story.Content.Lines {
		var audioFile *string
		// Match audio files with lines if they exist
		if err == nil && i < len(audioDir) {
			temp := fmt.Sprintf("/%v/%v", folderPath, audioDir[i].Name())
			audioFile = &temp
		}

		lines = append(lines, Line{
			Text:     []string{dbLine.Text},
			AudioURL: audioFile,
		})
	}

	data := PageData{
		StoryID:    storyID,
		StoryTitle: story.Metadata.Title["en"], // Using English title
		Lines:      lines,
	}

	err = h.te.Render(w, "page1.html", data)
	if err != nil {
		h.log.Error("Failed to render page", "error", err)
		http.Error(w, "Failed to render page", http.StatusInternalServerError)
		return
	}
}
