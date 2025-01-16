// logos-stories/internal/admin/stories/handler.go
package stories

import (
	"log/slog"
	"logos-stories/internal/pkg/models"
	"logos-stories/internal/pkg/templates"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
)

type Handler struct {
	log *slog.Logger
	te  *templates.TemplateEngine
}

func NewHandler(log *slog.Logger, te *templates.TemplateEngine) *Handler {
	return &Handler{
		log: log,
		te:  te,
	}
}

func (h *Handler) RegisterRoutes(r *mux.Router) {
	// Stories subrouter
	stories := r.PathPrefix("/stories").Subrouter()

	stories.HandleFunc("/add", h.addStoryHandler).Methods("GET", "POST")
	stories.HandleFunc("/edit/{id}", h.editStoryHandler).Methods("GET", "PUT")
	stories.HandleFunc("/delete/{id}", h.deleteStoryHandler).Methods("GET", "DELETE")
}

func (h *Handler) deleteStoryHandler(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusNotImplemented)
}

func (h *Handler) addStoryHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		// Render the add story form
		h.renderAddStoryForm(w, r)
		return
	}

	// Parse form
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Failed to parse form", http.StatusBadRequest)
		return
	}

	weekNum, err := strconv.Atoi(r.FormValue("weekNumber"))
	if err != nil {
		http.Error(w, "Invalid week number", http.StatusBadRequest)
		return
	}
	// Create request object
	req := AddStoryRequest{
		TitleEn:      r.FormValue("titleEn"),
		LanguageCode: r.FormValue("languageCode"),
		AuthorName:   r.FormValue("authorName"),
		WeekNumber:   weekNum, // You'll need to implement parseInt
		DayLetter:    r.FormValue("dayLetter"),
		StoryText:    r.FormValue("storyText"),
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

	// Redirect to success page or story list
	http.Redirect(w, r, "/admin/stories", http.StatusSeeOther)
}
