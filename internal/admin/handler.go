// glossias/internal/admin/admin.go
package admin

import (
	"log/slog"
	"net/http"
	"os"

	"glossias/internal/admin/stories"
	"glossias/internal/pkg/auth"
	"glossias/internal/pkg/models"
	"glossias/internal/pkg/templates"

	"github.com/gorilla/mux"
)

type Handler struct {
	log     *slog.Logger
	stories *stories.Handler
	te      *templates.TemplateEngine
}

func NewHandler(log *slog.Logger, te *templates.TemplateEngine) *Handler {
	key := os.Getenv("CLERK_SECRET_KEY")
	if key == "" {
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
	}
}

func (h *Handler) RegisterRoutes(r *mux.Router) {
	// Create admin subrouter
	admin := r.PathPrefix("/admin").Subrouter()

	// Initialize Clerk middleware
	clerkMiddleware, err := auth.NewClerkMiddleware()
	if err != nil {
		h.log.Error("Failed to initialize Clerk middleware", "error", err)
		panic(err)
	}

	// Apply Clerk authentication to all admin routes
	admin.Use(clerkMiddleware.RequireAuth)
	// Apply admin-specific middleware
	admin.Use(h.adminAuthMiddleware)

	admin.HandleFunc("", h.homeHandler).Methods("GET")

	// Register all admin routes
	h.stories.RegisterRoutes(admin)
}

func (h *Handler) adminAuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get token from context
		token, ok := r.Context().Value("clerk_token").(string)
		if !ok || token == "" {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// Get claims from context
		claims, ok := r.Context().Value("clerk_claims").(map[string]interface{})
		if !ok {
			http.Error(w, "Invalid claims", http.StatusUnauthorized)
			return
		}

		// Check if user has admin role
		if !hasAdminRole(claims) {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func hasAdminRole(claims map[string]interface{}) bool {
	// Get roles from claims
	roles, ok := claims["roles"].([]interface{})
	if !ok {
		return false
	}

	// Check if user has admin role
	for _, role := range roles {
		if roleStr, ok := role.(string); ok && roleStr == "admin" {
			return true
		}
	}

	return false
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
