// glossias/src/admin/stories/handler.go
package stories

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"

	"glossias/src/pkg/models"

	"github.com/gorilla/mux"
)

type Handler struct {
	log *slog.Logger
}

func NewHandler(log *slog.Logger) *Handler {
	return &Handler{
		log: log,
	}
}

// validateStoryID middleware ensures the story ID exists
func (h *Handler) validateStoryID(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		storyIDStr, exists := vars["id"]
		if !exists {
			next(w, r)
			return
		}

		storyID, err := strconv.Atoi(storyIDStr)
		if err != nil {
			http.Error(w, "Invalid story ID", http.StatusBadRequest)
			return
		}

		exists, err = models.StoryExists(r.Context(), int32(storyID))
		if err != nil {
			h.log.Error("Failed to check if story exists", "error", err, "storyID", storyID)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		if !exists {
			http.Error(w, "Story not found", http.StatusNotFound)
			return
		}

		next(w, r)
	}
}

func (h *Handler) RegisterRoutes(r *mux.Router) {
	// Base: /api/admin/stories
	stories := r.PathPrefix("/stories").Subrouter()

	// Basic hello test route
	stories.HandleFunc("/hello", func(w http.ResponseWriter, r *http.Request) {

		json.NewEncoder(w).Encode(map[string]string{"message": "Hello from admin/stories!"})
	}).Methods("GET", "OPTIONS")

	// Individual story endpoints
	stories.HandleFunc("", h.addStoryHandler).Methods("POST", "OPTIONS")
	stories.HandleFunc("/{id:[0-9]+}", h.validateStoryID(h.editStoryHandler)).Methods("GET", "PUT", "DELETE", "OPTIONS")
	stories.HandleFunc("/{id:[0-9]+}/metadata", h.validateStoryID(h.metadataHandler)).Methods("GET", "PUT", "OPTIONS")
	stories.HandleFunc("/{id:[0-9]+}/annotations", h.validateStoryID(h.annotationsHandler)).
		Methods("GET", "POST", "PUT", "DELETE", "OPTIONS")

	// Translation endpoints
	stories.HandleFunc("/{id:[0-9]+}/translations", h.validateStoryID(h.translationsHandler)).Methods("GET", "PUT", "DELETE", "OPTIONS")
	stories.HandleFunc("/{id:[0-9]+}/translations/line", h.validateStoryID(h.lineTranslationHandler)).Methods("GET", "PUT", "DELETE", "OPTIONS")
	stories.HandleFunc("/{id:[0-9]+}/translations/lang/{lang}", h.validateStoryID(h.translationsByLanguageHandler)).Methods("GET", "OPTIONS")

	// Audio upload endpoints
	stories.HandleFunc("/audio/upload", h.audioUploadHandler).Methods("POST", "OPTIONS")
	stories.HandleFunc("/audio/confirm", h.confirmAudioUploadHandler).Methods("POST", "OPTIONS")
}
