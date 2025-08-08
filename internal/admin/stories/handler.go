// glossias/internal/admin/stories/handler.go
package stories

import (
    "encoding/json"
	"log/slog"
	"glossias/internal/pkg/models"
	"glossias/internal/pkg/templates"
	"net/http"
	"strconv"
    "strings"

	"github.com/gorilla/mux"
)

type Handler struct {
	log            *slog.Logger
	te             *templates.TemplateEngine
	allowedOrigins []string
}

func NewHandler(log *slog.Logger, te *templates.TemplateEngine) *Handler {
	return &Handler{
		log:            log,
		te:             te,
        // [UNUSED NOTE] Admin HTML pages migrated to React. CORS kept for API calls.
        allowedOrigins: []string{"http://localhost:3000", "http://localhost:5173"}, // Dev
	}
}

func (h *Handler) RegisterRoutes(r *mux.Router) {
	// Stories subrouter
	stories := r.PathPrefix("/stories").Subrouter()

    stories.HandleFunc("/add", h.addStoryHandler).Methods("GET", "POST", "OPTIONS")
    stories.HandleFunc("/{id:[0-9]+}", h.editStoryHandler).Methods("GET", "PUT", "OPTIONS")
    stories.HandleFunc("/{id:[0-9]+}/metadata", h.metadataHandler).Methods("GET", "PUT", "OPTIONS")
    // [UNUSED] HTML page was served here; now handled by React route /admin/stories/:id/annotate
    stories.HandleFunc("/{id:[0-9]+}/annotate", h.handleGetEditPage).Methods("GET")
	stories.HandleFunc("/api/{id:[0-9]+}", h.annotationsHandler).
		Methods("GET", "PUT", "DELETE", "OPTIONS")
    stories.HandleFunc("/delete/{id}", h.deleteStoryHandler).Methods("GET", "DELETE", "OPTIONS")
}

// [+] Add CORS middleware helper
func (h *Handler) setCORSHeaders(w http.ResponseWriter, r *http.Request) {
	origin := r.Header.Get("Origin")
	// Allow configured origins or all in development
	for _, allowed := range h.allowedOrigins {
		if origin == allowed {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			break
		}
	}
	w.Header().Set("Access-Control-Allow-Methods", "GET, PUT, DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	w.Header().Set("Access-Control-Max-Age", "3600")
}

func (h *Handler) addStoryHandler(w http.ResponseWriter, r *http.Request) {
    h.setCORSHeaders(w, r)
    if r.Method == http.MethodOptions {
        w.WriteHeader(http.StatusOK)
        return
    }
	if r.Method == "GET" {
        // [UNUSED] Rendered HTML; React SPA now handles UI
		h.renderAddStoryForm(w, r)
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

    // Respond JSON for SPA
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(map[string]any{
        "success": true,
        "storyId": story.Metadata.StoryID,
    })
}
