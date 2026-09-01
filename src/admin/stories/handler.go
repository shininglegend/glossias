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
	stories.HandleFunc("/audio/delete", h.audioDeleteHandler).Methods("DELETE", "OPTIONS")

	// Image upload endpoints
	stories.HandleFunc("/image/upload", h.imageUploadHandler).Methods("POST", "OPTIONS")
	stories.HandleFunc("/image/confirm", h.confirmImageUploadHandler).Methods("POST", "OPTIONS")
	stories.HandleFunc("/image/delete", h.imageDeleteHandler).Methods("DELETE", "OPTIONS")

	// Summer 2026 phase authoring (T7). Assets for these editors are attached to
	// the owning target_vocabulary / recall_sentences row rather than registered
	// in story_images; see phase_assets.go.
	stories.HandleFunc("/{id:[0-9]+}/content-readiness", h.validateStoryID(h.contentReadinessHandler)).
		Methods("GET", "OPTIONS")
	stories.HandleFunc("/{id:[0-9]+}/phase-assets/upload", h.validateStoryID(h.phaseAssetUploadHandler)).
		Methods("POST", "OPTIONS")

	stories.HandleFunc("/{id:[0-9]+}/target-vocabulary", h.validateStoryID(h.targetVocabularyHandler)).
		Methods("GET", "POST", "OPTIONS")
	stories.HandleFunc("/{id:[0-9]+}/target-vocabulary/{wordId:[0-9]+}", h.validateStoryID(h.targetVocabularyItemHandler)).
		Methods("PUT", "DELETE", "OPTIONS")

	stories.HandleFunc("/{id:[0-9]+}/produce", h.validateStoryID(h.produceHandler)).
		Methods("GET", "OPTIONS")
	stories.HandleFunc("/{id:[0-9]+}/produce/explanation", h.validateStoryID(h.produceExplanationHandler)).
		Methods("PUT", "DELETE", "OPTIONS")
	stories.HandleFunc("/{id:[0-9]+}/produce/segments/{order:[0-9]+}", h.validateStoryID(h.produceSegmentHandler)).
		Methods("PUT", "DELETE", "OPTIONS")

	stories.HandleFunc("/{id:[0-9]+}/recall", h.validateStoryID(h.recallHandler)).
		Methods("GET", "OPTIONS")
	stories.HandleFunc("/{id:[0-9]+}/recall/sentences/{order:[0-9]+}", h.validateStoryID(h.recallSentenceHandler)).
		Methods("PUT", "DELETE", "OPTIONS")

	// Per-student performance on this story, and per-student progress reset.
	stories.HandleFunc("/{id:[0-9]+}/students", h.validateStoryID(h.storyStudentsHandler)).
		Methods("GET", "OPTIONS")
	stories.HandleFunc("/{id:[0-9]+}/students/{userId}/progress", h.validateStoryID(h.resetStudentProgressHandler)).
		Methods("DELETE", "OPTIONS")
}

// contentReadinessHandler reports which of the new phases are fully authored for
// a story. The editors show it as a checklist, and it is the same report
// navigation can use to skip phases whose content is absent.
func (h *Handler) contentReadinessHandler(w http.ResponseWriter, r *http.Request) {
	storyID, ok := h.authorizeStoryEdit(w, r)
	if !ok {
		return
	}

	readiness, err := models.GetStoryContentReadiness(r.Context(), storyID)
	if err != nil {
		h.log.Error("Failed to build content readiness report", "error", err, "storyID", storyID)
		writeJSONError(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(readiness)
}
