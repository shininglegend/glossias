// glossias/src/admin/handler.go
package stories

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"glossias/src/auth"
	"glossias/src/pkg/models"
	"net/http"
	"strings"
	"time"
)

// generateAuthorID creates a random 8-character hex string
func generateAuthorID() string {
	bytes := make([]byte, 4)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

type AddStoryRequest struct {
	TitleEn         string `json:"titleEn"`
	LanguageCode    string `json:"languageCode"`
	AuthorName      string `json:"authorName"`
	WeekNumber      int    `json:"weekNumber"`
	DayLetter       string `json:"dayLetter"`
	DescriptionText string `json:"descriptionText"`
	StoryText       string `json:"storyText"`
	CourseID        int    `json:"courseId"`
}

func (h *Handler) addStoryHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	// Parse JSON
	var req AddStoryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}
	// Validate required fields
	if req.TitleEn == "" || req.LanguageCode == "" || req.AuthorName == "" || req.StoryText == "" {
		http.Error(w, "Missing required fields", http.StatusBadRequest)
		return
	}

	// Validate course acces
	if !models.IsUserSuperAdmin(ctx, auth.GetUserID(r)) && !models.IsUserCourseAdmin(ctx, auth.GetUserID(r), int32(req.CourseID)) {
		http.Error(w, "Forbidden: not a course admin", http.StatusForbidden)
		return
	}
	// Check course exists
	_, err := models.GetCourse(ctx, int32(req.CourseID))
	if err != nil {
		if err == models.ErrNotFound {
			http.Error(w, "Course not found", http.StatusBadRequest)
			return
		}
		h.log.Error("failed to verify course existence", "error", err, "course_id", req.CourseID)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Process the story
	story, err := h.processAddStory(req)
	if err != nil {
		h.log.Error("Failed to process story", "error", err)
		http.Error(w, "Failed to process story", http.StatusInternalServerError)
		return
	}

	// Save the story
	if err := models.SaveNewStory(ctx, story); err != nil {
		h.log.Error("Failed to save story", "error", err)
		http.Error(w, "Failed to save story", http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(map[string]any{
		"success": true,
		"storyId": story.Metadata.StoryID,
	})
}

func (h *Handler) processAddStory(req AddStoryRequest) (*models.Story, error) {
	// Split story text into lines
	textLines := strings.Split(req.StoryText, "\n")

	// Create story lines
	lines := make([]models.StoryLine, len(textLines))
	for i, text := range textLines {
		if strings.TrimSpace(text) == "" && i == len(textLines)-1 {
			// Skip trailing empty line at end
			continue
		}
		lines[i] = models.StoryLine{
			LineNumber: i + 1,
			Text:       text,
		}
	}

	// Create story structure
	updateTime := time.Now().UTC()
	story := &models.Story{
		Metadata: models.StoryMetadata{
			// ID will be added later
			WeekNumber: req.WeekNumber,
			DayLetter:  req.DayLetter,
			Title: map[string]string{
				"en": req.TitleEn,
			},
			Author: models.Author{
				ID:   generateAuthorID(),
				Name: req.AuthorName,
			},
			Description: models.Description{
				Language: req.LanguageCode,
				Text:     req.DescriptionText,
			},
			CourseID:     &req.CourseID,
			LastRevision: &updateTime,
		},
		Content: models.StoryContent{
			Lines: lines,
		},
	}

	return story, nil
}
