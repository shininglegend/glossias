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
	// Stories list endpoint
	router.HandleFunc("/stories", h.GetStories).Methods("GET", "OPTIONS")

	// Individual page endpoints
	router.HandleFunc("/stories/{id}/page1", h.GetPage1).Methods("GET", "OPTIONS")
	router.HandleFunc("/stories/{id}/page2", h.GetPage2).Methods("GET", "OPTIONS")
	router.HandleFunc("/stories/{id}/page3", h.GetPage3).Methods("GET", "OPTIONS")
	router.HandleFunc("/stories/{id}/page4", h.GetPage4).Methods("GET", "OPTIONS")

	// Vocabulary checking endpoint
	router.HandleFunc("/stories/{id}/check-vocab", h.CheckVocab).Methods("POST", "OPTIONS")
}

// GetStories returns JSON array of all stories
func (h *Handler) GetStories(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

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
