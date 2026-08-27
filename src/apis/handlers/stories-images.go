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

// GetSignedImageURLs returns signed URLs for image files in a story
func (h *Handler) GetSignedImageURLs(w http.ResponseWriter, r *http.Request) {
	storyID := mux.Vars(r)["id"]
	id, err := strconv.Atoi(storyID)
	if err != nil {
		h.sendError(w, "Invalid story ID format", http.StatusBadRequest)
		return
	}

	// Get label filter from query parameters
	label := r.URL.Query().Get("label")

	// Generate signed URLs (expires in 1 hour)
	signedURLs, err := models.GetSignedImageURLsForStory(r.Context(), id, auth.GetUserID(r), label, expiresInSeconds)
	if err == models.ErrNotFound {
		h.sendError(w, "Story or image files not found.", http.StatusNotFound)
		return
	}
	if err != nil {
		h.log.Error("Failed to generate signed image URLs", "error", err)
		h.sendError(w, "Failed to generate signed URLs", http.StatusInternalServerError)
		return
	}

	response := types.APIResponse{
		Success: true,
		Data:    signedURLs,
	}

	json.NewEncoder(w).Encode(response)
}
