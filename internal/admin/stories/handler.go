// glossias/internal/admin/stories/handler.go
package stories

import (
	"encoding/json"
	"glossias/internal/pkg/templates"
	"log/slog"
	"net/http"

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
	// Base: /api/admin/stories
	stories := r.PathPrefix("/stories").Subrouter()

	// Basic hello test route
	stories.HandleFunc("/hello", func(w http.ResponseWriter, r *http.Request) {

		json.NewEncoder(w).Encode(map[string]string{"message": "Hello from admin/stories!"})
	}).Methods("GET", "OPTIONS")

	// Individual story endpoints
	stories.HandleFunc("", h.addStoryHandler).Methods("POST", "OPTIONS")
	stories.HandleFunc("/{id:[0-9]+}", h.editStoryHandler).Methods("GET", "PUT", "DELETE", "OPTIONS")
	stories.HandleFunc("/{id:[0-9]+}/metadata", h.metadataHandler).Methods("GET", "PUT", "OPTIONS")
	stories.HandleFunc("/{id:[0-9]+}/annotations", h.annotationsHandler).
		Methods("GET", "PUT", "DELETE", "OPTIONS")
}
