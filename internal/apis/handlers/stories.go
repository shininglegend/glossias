package handlers

import (
	"encoding/json"
	"glossias/internal/apis/types"
	"glossias/internal/pkg/models"
	"net/http"

	"github.com/gorilla/mux"
)

// RegisterRoutes registers all API routes with the given router
func (h *Handler) RegisterRoutes(router *mux.Router) {
	// Base is /api/stories
	// Stories list endpoint
	router.HandleFunc("", h.GetStories).Methods("GET", "OPTIONS")

	// Basic hello test route
	router.HandleFunc("/hello", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]string{"message": "Hello from admin stories!"})
	}).Methods("GET", "OPTIONS")

	// Individual page endpoints
	router.HandleFunc("/{id}/page1", h.GetPage1).Methods("GET", "OPTIONS")
	router.HandleFunc("/{id}/page2", h.GetPage2).Methods("GET", "OPTIONS")
	router.HandleFunc("/{id}/page3", h.GetPage3).Methods("GET", "OPTIONS")
	router.HandleFunc("/{id}/page4", h.GetPage4).Methods("GET", "OPTIONS")

	// Vocabulary checking endpoint
	router.HandleFunc("/{id}/check-vocab", h.CheckVocab).Methods("POST", "OPTIONS")
}

// GetStories returns JSON array of all stories
func (h *Handler) GetStories(w http.ResponseWriter, r *http.Request) {
	// Get language parameter (optional)
	lang := r.URL.Query().Get("lang")

	// Fetch stories from database
	dbStories, err := models.GetAllStories(lang)
	if err != nil {
		h.log.Error("Failed to fetch stories from database", "error", err)
		h.sendError(w, "Failed to fetch stories", http.StatusInternalServerError)
		return
	}

	// Convert to API format
	stories := types.ConvertStoriesToAPI(dbStories)
	response := types.APIResponse{
		Success: true,
		Data: types.StoriesResponse{
			Stories: stories,
		},
	}

	json.NewEncoder(w).Encode(response)
}
