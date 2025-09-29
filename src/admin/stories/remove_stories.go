// glossias/src/admin/stories/remove_stories.go
package stories

import (
	"encoding/json"
	"fmt"
	"glossias/src/pkg/models"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/gorilla/mux"
)

func (h *Handler) deleteStoryHandler(w http.ResponseWriter, r *http.Request, userID string) {
	vars := mux.Vars(r)
	storyID, err := strconv.Atoi(vars["id"])
	if err != nil {
		writeJSONError(w, "Invalid story ID", http.StatusBadRequest)
		return
	}

	// Validate user permissions
	if !models.IsUserCourseOrSuperAdmin(r.Context(), userID, int32(storyID)) {
		http.Error(w, "Forbidden: not a course admin", http.StatusForbidden)
		return
	}

	// First fetch the story data for logging
	story, err := models.GetStoryData(r.Context(), storyID, userID)
	if err != nil {
		if err == models.ErrNotFound {
			writeJSONError(w, "Story not found", http.StatusNotFound)
			return
		}
		h.log.Error("Failed to fetch story for deletion", "error", err)
		writeJSONError(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Log deletion to file
	if err := logDeletion(r, story); err != nil {
		h.log.Error("Failed to log deletion", "error", err)
		// Continue with deletion even if logging fails
	}

	// Delete from database
	if err := models.Delete(r.Context(), storyID); err != nil {
		h.log.Error("Failed to delete story", "error", err)
		writeJSONError(w, "Failed to delete story", http.StatusInternalServerError)
		return
	}

	// Return success
	json.NewEncoder(w).Encode(map[string]bool{"success": true})
}

func logDeletion(r *http.Request, story *models.Story) error {
	// Ensure logs directory exists
	logDir := "logs"
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return fmt.Errorf("failed to create logs directory: %w", err)
	}

	// Open log file with date prefix
	logPath := filepath.Join(logDir, fmt.Sprintf("deletions_%s.log",
		time.Now().Format("2006-01")))
	f, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open log file: %w", err)
	}
	defer f.Close()

	// Format log entry
	logEntry := fmt.Sprintf("[%s] IP: %s - Deleted Story ID: %d\nTitle: %s\nContent:\n%s\n\n",
		time.Now().Format(time.RFC3339),
		r.RemoteAddr,
		story.Metadata.StoryID,
		story.Metadata.Title["en"],
		formatStoryContent(story.Content.Lines))

	if _, err := f.WriteString(logEntry); err != nil {
		return fmt.Errorf("failed to write to log: %w", err)
	}
	return nil
}

func formatStoryContent(lines []models.StoryLine) string {
	var content string
	for _, line := range lines {
		content += line.Text + "\n"
	}
	return content
}
