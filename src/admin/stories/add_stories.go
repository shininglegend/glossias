// glossias/src/admin/handler.go
package stories

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"glossias/src/pkg/models"
	"net/http"
	"strconv"
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
}

func (h *Handler) addStoryHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	// Parse JSON or form for backward compatibility
	var req AddStoryRequest
	if strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}
	} else {
		if err := r.ParseForm(); err != nil {
			http.Error(w, "Failed to parse form", http.StatusBadRequest)
			return
		}
		weekNum, err := strconv.Atoi(r.FormValue("weekNumber"))
		if err != nil {
			http.Error(w, "Invalid week number", http.StatusBadRequest)
			return
		}
		req = AddStoryRequest{
			TitleEn:      r.FormValue("titleEn"),
			LanguageCode: r.FormValue("languageCode"),
			AuthorName:   r.FormValue("authorName"),
			WeekNumber:   weekNum,
			DayLetter:    r.FormValue("dayLetter"),
			StoryText:    r.FormValue("storyText"),
		}
	}

	// Process the story
	story, err := h.processAddStory(req)
	if err != nil {
		h.log.Error("Failed to process story", "error", err)
		http.Error(w, "Failed to process story", http.StatusInternalServerError)
		return
	}

	// Save the story
	if err := models.SaveNewStory(story); err != nil {
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
	textLines := strings.Split(strings.TrimSpace(req.StoryText), "\n")

	// Create story lines
	lines := make([]models.StoryLine, len(textLines))
	for i, text := range textLines {
		lines[i] = models.StoryLine{
			LineNumber: i + 1,
			Text:       strings.TrimSpace(text),
			// Vocabulary: []models.VocabularyItem{},
			// Grammar:    []models.GrammarItem{},
			// Footnotes:  []models.Footnote{},
		}
	}

	// Create story structure
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
			LastRevision: time.Now().UTC(),
		},
		Content: models.StoryContent{
			Lines: lines,
		},
	}

	return story, nil
}
