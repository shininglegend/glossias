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

	// Navigation endpoint
	router.HandleFunc("/{id}/next", h.Navigate).Methods("POST", "OPTIONS")

	// List stories by course endpoint
	router.HandleFunc("/by-course/{course_id}", h.GetCourseStories).Methods("GET", "OPTIONS")

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

// Get stories by course returns a list of stories for a given course
func (h *Handler) GetCourseStories(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	courseIDStr := vars["course_id"]
	courseID, err := strconv.Atoi(courseIDStr)
	if err != nil {
		http.Error(w, "Invalid course ID", http.StatusBadRequest)
		return
	}

	userID, ok := auth.GetUserIDWithOk(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Check if user has read access to this course
	if !auth.HasReadPermission(r.Context(), userID, int32(courseID)) {
		http.Error(w, "Access denied", http.StatusForbidden)
		return
	}

	stories, err := models.GetStoriesForCourse(r.Context(), courseID)
	if err != nil {
		h.log.Error("Failed to get stories for course", "error", err, "course_id", courseID)
		json.NewEncoder(w).Encode(types.APIResponse{
			Success: false,
			Error:   "Failed to get stories for course",
		})
		return
	}

	json.NewEncoder(w).Encode(types.APIResponse{
		Success: true,
		Data:    stories,
	})
}
