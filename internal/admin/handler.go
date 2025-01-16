// logos-stories/internal/admin/admin.go
package admin

import (
	"log/slog"
	"net/http"

	"logos-stories/internal/admin/stories"
	"logos-stories/internal/pkg/models"
	"logos-stories/internal/pkg/templates"

	"github.com/gorilla/mux"
)

type Handler struct {
	log     *slog.Logger
	stories *stories.Handler
	te      *templates.TemplateEngine
}

func NewHandler(log *slog.Logger, te *templates.TemplateEngine) *Handler {
	return &Handler{
		log:     log,
		te:      te,
		stories: stories.NewHandler(log, te),
	}
}

func (h *Handler) RegisterRoutes(r *mux.Router) {
	// Create admin subrouter
	admin := r.PathPrefix("/admin").Subrouter()

	// Admin-specific middleware
	admin.Use(h.adminAuthMiddleware)

	admin.HandleFunc("", h.homeHandler).Methods("GET")

	// Register all admin routes
	h.stories.RegisterRoutes(admin)
}

func (h *Handler) adminAuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Admin authentication logic here
		next.ServeHTTP(w, r)
	})
}

// Added new handler
func (h *Handler) homeHandler(w http.ResponseWriter, r *http.Request) {
	stories, err := models.GetAllStories("")
	if err != nil {
		h.log.Error("Failed to fetch stories", "error", err)
		http.Error(w, "Failed to fetch stories", http.StatusInternalServerError)
		return
	}

	data := map[string]interface{}{
		"Stories": stories,
	}

	if err := h.te.Render(w, "admin/adminHome.html", data); err != nil {
		h.log.Error("Failed to render admin home", "error", err)
		http.Error(w, "Failed to render admin home", http.StatusInternalServerError)
	}
}
