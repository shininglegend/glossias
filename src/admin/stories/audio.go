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

type AudioUploadRequest struct {
	StoryID    int    `json:"storyId"`
	LineNumber int    `json:"lineNumber"`
	Label      string `json:"label"`
	FileName   string `json:"fileName"`
}

type AudioUploadResponse struct {
	UploadURL  string `json:"uploadUrl"`
	FilePath   string `json:"filePath"`
	FileBucket string `json:"fileBucket"`
}

const (
	bucket = "audio-files"
)

func (h *Handler) audioUploadHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "POST":
		h.requestAudioUploadURL(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *Handler) requestAudioUploadURL(w http.ResponseWriter, r *http.Request) {
	var req AudioUploadRequest
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
	if req.StoryID <= 0 || req.LineNumber <= 0 || req.Label == "" || req.FileName == "" {
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

	// Check if line exists
	lineExists, err := models.LineExists(r.Context(), req.StoryID, req.LineNumber)
	if err != nil {
		h.log.Error("Failed to check line existence", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	if !lineExists {
		http.Error(w, "Line not found", http.StatusNotFound)
		return
	}

	// Check if audio file already exists for this line and label
	existingAudioFiles, err := models.GetLineAudioFiles(r.Context(), req.StoryID, req.LineNumber)
	if err != nil {
		h.log.Error("Failed to check existing audio files", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	for _, audioFile := range existingAudioFiles {
		if audioFile.Label == req.Label {
			http.Error(w, "Audio file with this label already exists for this line", http.StatusConflict)
			return
		}
	}

	// Generate file path: stories/{storyID}/line_{lineNumber}_{label}_{filename}
	timestamp := time.Now().Unix()
	// Sanitize filename to prevent path traversal
	sanitizedFilename := strings.ReplaceAll(req.FileName, "/", "")
	sanitizedFilename = strings.ReplaceAll(sanitizedFilename, "\\", "")
	sanitizedFilename = strings.ReplaceAll(sanitizedFilename, "..", "")
	filePath := "stories/" + strconv.Itoa(req.StoryID) + "/line_" +
		strconv.Itoa(req.LineNumber) + "_" + req.Label + "_" +
		strconv.FormatInt(timestamp, 10) + "_" + sanitizedFilename

	// Generate signed upload URL
	signedURL, err := models.GenerateSignedUploadURL(r.Context(), bucket, filePath)
	if err != nil {
		h.log.Error("Failed to generate signed upload URL", "error", err)
		http.Error(w, "Failed to generate upload URL", http.StatusInternalServerError)
		return
	}

	response := AudioUploadResponse{
		UploadURL:  signedURL,
		FilePath:   filePath,
		FileBucket: bucket,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (h *Handler) audioDeleteHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "DELETE" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	type DeleteAudioRequest struct {
		StoryID    int `json:"storyId"`
		LineNumber int `json:"lineNumber"`
	}

	var req DeleteAudioRequest
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
	if req.StoryID <= 0 || req.LineNumber <= 0 {
		http.Error(w, "Invalid request: storyId and lineNumber must be positive", http.StatusBadRequest)
		return
	}

	// Delete all audio files for the line
	err := models.DeleteLineAudioFiles(r.Context(), req.StoryID, req.LineNumber)
	if err != nil {
		h.log.Error("Failed to delete line audio files", "error", err)
		http.Error(w, "Failed to delete audio files", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) confirmAudioUploadHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	type ConfirmUploadRequest struct {
		StoryID    int    `json:"storyId"`
		LineNumber int    `json:"lineNumber"`
		FilePath   string `json:"filePath"`
		FileBucket string `json:"fileBucket"`
		Label      string `json:"label"`
	}

	var req ConfirmUploadRequest
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
	expectedPrefix := "stories/" + strconv.Itoa(req.StoryID) + "/line_" + strconv.Itoa(req.LineNumber) + "_" + req.Label + "_"
	if !strings.HasPrefix(req.FilePath, expectedPrefix) {
		http.Error(w, "Invalid file path", http.StatusBadRequest)
		return
	}

	// Create audio file record in database
	audioFile, err := models.CreateAudioFile(r.Context(), req.StoryID, req.LineNumber,
		req.FilePath, req.FileBucket, req.Label)
	if err != nil {
		if err == models.ErrAudioFileExists {
			http.Error(w, "An audio file with this label already exists for this line", http.StatusBadRequest)
			return
		}
		h.log.Error("Failed to create audio file record", "error", err)
		http.Error(w, "Failed to create audio file record", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(audioFile)
}
