// stories.go
package stories

import (
	"embed"
	"log/slog"
	"logos-stories/internal/pkg/templates"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/gorilla/mux"
)

const storiesDir = "static/stories/"

type Handler struct {
	log       *slog.Logger
	templates *templates.TemplateEngine
}

type Line struct {
	Text     string
	AudioURL *string
}

type PageData struct {
	StoryID    string
	StoryTitle string
	Lines      []Line
}

func NewHandler(logger *slog.Logger, templateFS embed.FS) *Handler {
	return &Handler{
		templates: templates.New(templateFS),
		log:       logger,
	}
}

// RegisterRoutes registers all story-related routes
func (h *Handler) RegisterRoutes(mux *mux.Router) {
	mux.HandleFunc("/stories/{id}/page1", h.ServePage1).Methods("GET").Name("page1")
	mux.HandleFunc("/stories", h.ServeIndex).Methods("GET")
}

// stories.go
// Add these new types to the existing file
type Story struct {
	ID    string
	Title string
}

type IndexData struct {
	Stories []Story
}

// Add this new method
func (h *Handler) ServeIndex(w http.ResponseWriter, r *http.Request) {
	// Get all story files
	files, err := os.ReadDir(storiesDir + "stories_text")
	if err != nil {
		h.log.Error("Failed to read stories directory", "error", err)
		http.Error(w, "Failed to read stories", http.StatusInternalServerError)
		return
	}

	stories := make([]Story, 0, len(files))
	// Process each file
	for _, file := range files {
		if filepath.Ext(file.Name()) != ".txt" {
			continue
		}

		// Extract ID from filename (remove .txt extension)
		id := strings.TrimSuffix(file.Name(), ".txt")

		// Read the file to get the title (first line)
		content, err := os.ReadFile(filepath.Join(storiesDir, "stories_text", file.Name()))
		if err != nil {
			h.log.Error("Failed to read story file", "error", err, "file", file.Name())
			continue
		}

		// Get first line as title
		lines := strings.Split(string(content), "\n")
		if len(lines) == 0 {
			h.log.Error("Empty story file", "file", file.Name())
			continue
		}

		title := strings.TrimSpace(lines[0])

		stories = append(stories, Story{
			ID:    id,
			Title: title,
		})
	}

	// Sort stories by ID for consistent presentation
	sort.Slice(stories, func(i, j int) bool {
		return stories[i].ID < stories[j].ID
	})

	data := IndexData{
		Stories: stories,
	}

	if err := h.templates.Render(w, "index.html", data); err != nil {
		h.log.Error("Failed to render template", "error", err)
		http.Error(w, "Failed to render template", http.StatusInternalServerError)
		return
	}
}

// Implement the GetTitle interface
func (p PageData) GetTitle() string {
	return p.StoryTitle
}
