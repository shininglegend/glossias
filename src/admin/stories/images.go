package stories

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"glossias/src/auth"
	"glossias/src/pkg/models"
)

type ImageUploadRequest struct {
	StoryID  int    `json:"storyId"`
	Label    string `json:"label"`
	FileName string `json:"fileName"`
}

type ImageUploadResponse struct {
	UploadURL  string `json:"uploadUrl"`
	FilePath   string `json:"filePath"`
	FileBucket string `json:"fileBucket"`
}

const (
	imagesBucket = "images"
)

func (h *Handler) imageUploadHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "POST":
		h.requestImageUploadURL(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *Handler) requestImageUploadURL(w http.ResponseWriter, r *http.Request) {
	var req ImageUploadRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Admin authentication check
	userID, ok := auth.GetUserIDWithOk(r)
	if !ok || !models.CanUserEditStory(r.Context(), userID, int32(req.StoryID)) {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Validate request data
	if req.StoryID <= 0 || req.Label == "" || req.FileName == "" {
		http.Error(w, "Invalid Request: Data is outside bounds. Ensure all data is present and try again.", http.StatusBadRequest)
		return
	}

	// Check if story exists
	exists, err := models.StoryExists(r.Context(), int32(req.StoryID))
	if err != nil {
		h.log.Error("Failed to check story existence", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	if !exists {
		http.Error(w, "Story not found", http.StatusNotFound)
		return
	}

	// Generate file path: stories/{storyID}/image_{label}_{filename}
	timestamp := time.Now().Unix()
	// Sanitize filename to prevent path traversal
	sanitizedFilename := strings.ReplaceAll(req.FileName, "/", "")
	sanitizedFilename = strings.ReplaceAll(sanitizedFilename, "\\", "")
	sanitizedFilename = strings.ReplaceAll(sanitizedFilename, "..", "")
	filePath := "stories/" + strconv.Itoa(req.StoryID) + "/image_" +
		req.Label + "_" + strconv.FormatInt(timestamp, 10) + "_" + sanitizedFilename

	// Generate signed upload URL
	signedURL, err := models.GenerateSignedUploadURL(r.Context(), imagesBucket, filePath)
	if err != nil {
		h.log.Error("Failed to generate signed upload URL", "error", err)
		http.Error(w, "Failed to generate upload URL", http.StatusInternalServerError)
		return
	}

	response := ImageUploadResponse{
		UploadURL:  signedURL,
		FilePath:   filePath,
		FileBucket: imagesBucket,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (h *Handler) imageDeleteHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "DELETE" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	type DeleteImageRequest struct {
		StoryID int `json:"storyId"`
		ImageID int `json:"imageId"`
	}

	var req DeleteImageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Admin authentication check
	userID, ok := auth.GetUserIDWithOk(r)
	if !ok || !models.CanUserEditStory(r.Context(), userID, int32(req.StoryID)) {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Validate request data
	if req.StoryID <= 0 || req.ImageID <= 0 {
		http.Error(w, "Invalid request: storyId and imageId must be positive", http.StatusBadRequest)
		return
	}

	// Delete image file
	err := models.DeleteStoryImage(r.Context(), req.ImageID)
	if err != nil {
		h.log.Error("Failed to delete story image", "error", err)
		http.Error(w, "Failed to delete image file", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) confirmImageUploadHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	type ConfirmImageUploadRequest struct {
		StoryID    int    `json:"storyId"`
		FilePath   string `json:"filePath"`
		FileBucket string `json:"fileBucket"`
		Label      string `json:"label"`
	}

	var req ConfirmImageUploadRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Admin authentication check for confirm step
	userID, ok := auth.GetUserIDWithOk(r)
	if !ok || !models.CanUserEditStory(r.Context(), userID, int32(req.StoryID)) {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Verify file path matches expected pattern to prevent path manipulation
	expectedPrefix := "stories/" + strconv.Itoa(req.StoryID) + "/image_" + req.Label + "_"
	if !strings.HasPrefix(req.FilePath, expectedPrefix) {
		http.Error(w, "Invalid file path", http.StatusBadRequest)
		return
	}

	// Create story image record in database
	storyImage, err := models.CreateStoryImage(r.Context(), req.StoryID,
		req.FilePath, req.FileBucket, req.Label)
	if err != nil {
		h.log.Error("Failed to create story image record", "error", err)
		http.Error(w, "Failed to create story image record", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(storyImage)
}
