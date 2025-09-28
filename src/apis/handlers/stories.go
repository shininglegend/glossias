package handlers

import (
	"encoding/json"
	"glossias/src/apis/types"
	"glossias/src/auth"
	"glossias/src/pkg/models"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
)

const expiresInSeconds = 60 * 60 // 1 hour

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
	router.HandleFunc("/{id}/metadata", h.GetStoryMetadata).Methods("GET", "OPTIONS")
	router.HandleFunc("/{id}/story-with-audio", h.GetAudioPage).Methods("GET", "OPTIONS")
	router.HandleFunc("/{id}/vocab", h.GetVocabPage).Methods("GET", "OPTIONS")
	router.HandleFunc("/{id}/grammar", h.GetGrammarPage).Methods("GET", "OPTIONS")
	router.HandleFunc("/{id}/translate", h.GetTranslateData).Methods("POST", "OPTIONS")
	router.HandleFunc("/{id}/scores", h.GetScoresData).Methods("GET", "OPTIONS")

	// Audio endpoints
	router.HandleFunc("/{id}/audio/signed", h.GetSignedAudioURLs).Methods("GET", "OPTIONS")

	// Vocabulary checking endpoint
	router.HandleFunc("/{id}/check-vocab", h.CheckVocab).Methods("POST", "OPTIONS")
	// Grammar checking endpoint
	router.HandleFunc("/{id}/check-grammar", h.CheckGrammar).Methods("POST", "OPTIONS")
}

// GetStories returns JSON array of all stories
func (h *Handler) GetStories(w http.ResponseWriter, r *http.Request) {
	// Get language parameter (optional)
	lang := r.URL.Query().Get("lang")

	// Fetch stories from database
	dbStories, err := models.GetAllStories(r.Context(), lang, auth.GetUserID(r))
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

// GetSignedAudioURLs returns signed URLs for audio files in a story
func (h *Handler) GetSignedAudioURLs(w http.ResponseWriter, r *http.Request) {
	storyID := mux.Vars(r)["id"]
	id, err := strconv.Atoi(storyID)
	if err != nil {
		h.sendError(w, "Invalid story ID format", http.StatusBadRequest)
		return
	}

	// Get label filter from query parameters
	label := r.URL.Query().Get("label")

	// Generate signed URLs (expires in 4 hours)
	signedURLs, err := models.GetSignedAudioURLsForStory(r.Context(), id, auth.GetUserID(r), label, expiresInSeconds)
	if err == models.ErrNotFound {
		h.sendError(w, "Story or audio files not found.", http.StatusNotFound)
		return
	}
	if err != nil {
		h.log.Error("Failed to generate signed audio URLs", "error", err)
		h.sendError(w, "Failed to generate signed URLs", http.StatusInternalServerError)
		return
	}

	response := types.APIResponse{
		Success: true,
		Data:    signedURLs,
	}

	json.NewEncoder(w).Encode(response)
}
