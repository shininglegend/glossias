// glossias/internal/admin/admin.go
package admin

import (
	"log/slog"
	"net/http"
	"os"

	"glossias/internal/admin/stories"
	"glossias/internal/pkg/models"
	"glossias/internal/pkg/templates"

	"github.com/gorilla/mux"
)

type Handler struct {
	log     *slog.Logger
	stories *stories.Handler
	te      *templates.TemplateEngine
	client  clerk.Client // Add Clerk client
}

func NewHandler(log *slog.Logger, te *templates.TemplateEngine) *Handler {
	client, _ := clerk.NewClient(os.Getenv("CLERK_SECRET_KEY"))
	if client == nil {
		if os.Getenv("DATABASE_URL") == "" {
			log.Warn("Testing mode: Clerk is disabled.")
		} else {
			panic("CLERK_SECRET_KEY environment variable not set")
		}
	}
	return &Handler{
		log:     log,
		te:      te,
		stories: stories.NewHandler(log, te),
		client:  client,
	}
}

func (h *Handler) requireAdmin(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		sess := clerk.SessionFromContext(r.Context())
		if sess == nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// Get user claims
		claims, err := h.client.Sessions().GetClaims(sess.ID)
		if err != nil {
			h.log.Error("Failed to get claims", "error", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		// Check if user has admin role
		if !hasAdminRole(claims) {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}

		next(w, r)
	}
}

func hasAdminRole(claims map[string]interface{}) bool {
	// Implement your role checking logic here
	// Example: roles, ok := claims["roles"].([]interface{})
	return true // Temporary
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
