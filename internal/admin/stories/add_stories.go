// glossias/internal/admin/handler.go
package stories

import (
	"crypto/rand"
	"encoding/hex"
	"glossias/internal/pkg/models"
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

func (h *Handler) renderAddStoryForm(w http.ResponseWriter, _ *http.Request) {
	// Render the add story form
	if err := h.te.Render(w, "admin/addStory.html", nil); err != nil {
		h.log.Error("Failed to render add story form", "error", err)
		http.Error(w, "Failed to render add story form", http.StatusInternalServerError)
	}
}
