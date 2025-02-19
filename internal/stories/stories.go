// stories.go
package stories

import (
	"glossias/internal/pkg/models"
	"glossias/internal/pkg/templates"
	"log/slog"
	"net/http"
	"path/filepath"
	"text/template"

	"github.com/gorilla/mux"
)

const storiesDir = "static/stories/"

type Handler struct {
	log *slog.Logger
	te  *templates.TemplateEngine
}

type Line struct {
	Text              []string // Array of the line's text. May only have one item
	AudioURL          *string
	HasVocabOrGrammar bool
}

type PageData struct {
	StoryID    string
	StoryTitle string
	Lines      []Line
}

func NewHandler(logger *slog.Logger, te *templates.TemplateEngine) *Handler {
	return &Handler{
		log: logger,
		te:  te,
	}
}

// RegisterRoutes registers all story-related routes
func (h *Handler) RegisterRoutes(mux *mux.Router) {
	mux.HandleFunc("/stories/{id}/page1", h.ServePage1).Methods("GET").Name("page1")
	mux.HandleFunc("/stories/{id}/page2", h.ServePage2).Methods("GET").Name("page2")
	mux.HandleFunc("/stories/{id}/page3", h.ServePage3).Methods("GET").Name("page3")
	mux.HandleFunc("/stories/{id}/check-vocab", h.CheckVocabAnswers).Methods("POST")

	mux.HandleFunc("/", h.ServeIndex).Methods("GET")
}

// stories.go
// Add these new types to the existing file
type Story struct {
	ID         int
	Title      string
	WeekNumber int
	DayLetter  string
}

type IndexData struct {
	Stories []Story
}

func (h *Handler) ServeIndex(w http.ResponseWriter, r *http.Request) {
	// If they gave a language param, use it
	if r.URL.Query().Get("lang") != "" {
		h.log.Info("Language parameter provided", "language", r.URL.Query().Get("lang"))
	} else {
		h.log.Info("No language parameter provided")
	}
	// Get stories from database
	dbStories, err := models.GetAllStories(r.URL.Query().Get("lang")) // TODO: Add language from request, if given
	if err != nil {
		h.log.Error("Failed to fetch stories from database", "error", err)
		http.Error(w, "Failed to fetch stories", http.StatusInternalServerError)
		return
	}

	// Convert database stories to template format
	stories := make([]Story, 0, len(dbStories))
	for _, dbStory := range dbStories {
		stories = append(stories, Story{
			ID:         dbStory.Metadata.StoryID,
			Title:      dbStory.Metadata.Title["en"], // Using English title
			WeekNumber: dbStory.Metadata.WeekNumber,
			DayLetter:  dbStory.Metadata.DayLetter,
		})
	}

	// Get template path
	templatePath, err := filepath.Abs("src/templates/index.html")
	if err != nil {
		h.log.Error("Failed to find template", "error", err)
		http.Error(w, "Failed to find template", http.StatusInternalServerError)
		return
	}

	// Parse and execute template
	tmpl, err := template.ParseFiles(templatePath)
	if err != nil {
		h.log.Error("Failed to parse template", "error", err)
		http.Error(w, "Failed to parse template", http.StatusInternalServerError)
		return
	}

	data := IndexData{
		Stories: stories,
	}

	err = tmpl.Execute(w, data)
	if err != nil {
		h.log.Error("Failed to execute template", "error", err)
		http.Error(w, "Failed to execute template", http.StatusInternalServerError)
		return
	}
}
