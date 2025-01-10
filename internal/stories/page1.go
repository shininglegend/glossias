package stories

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/gorilla/mux"
)

// Read from html file
func (h *Handler) ServePage1(w http.ResponseWriter, r *http.Request) {
	// Add the text to the template.
	data := PageData{}
	// Get the story id from the URL
	storyID := mux.Vars(r)["id"]
	if storyID == "" {
		h.log.Info("Missing or invalid story ID", "story_id", storyID)
		http.Error(w, fmt.Sprintf("Missing or invalid story ID. Got: '%v'", storyID), http.StatusBadRequest)
		return
	}

	lines := []Line{}
	// Load the audio files from the folder
	audioDir, err := os.ReadDir(fmt.Sprintf(storiesDir+"stories_audio/%v", storyID))
	if err != nil {
		if err == os.ErrNotExist {
			h.log.Info("Story not found", "story_id", storyID)
			http.Error(w, fmt.Sprintf("Story with ID '%v' not found", storyID), http.StatusNotFound)
			return
		}
		h.log.Error("Failed to read audio files", "error", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	// Load the text file
	textBytes, err := os.ReadFile(fmt.Sprintf(storiesDir+"stories_text/%v.txt", storyID))
	if err != nil {
		h.log.Error("Failed to read text file", "error", err)
		http.Error(w, fmt.Sprintf("Failed to read text file for story with ID '%v'", storyID), http.StatusInternalServerError)
		return
	}
	// Split the content by newlines, preserving empty lines
	textLines := strings.Split(string(textBytes), "\n")

	// Clean up any carriage returns and trailing/leading whitespace
	for i, line := range textLines {
		textLines[i] = strings.TrimSpace(strings.ReplaceAll(line, "\r", ""))
	}
	// Get the title
	storyTitle := textLines[0]
	textLines = textLines[1:]

	// Split by newlines, should equal audio line count (excluding blanks)
	if len(audioDir) > len(textLines) {
		h.log.Error("Mismatch between audio and text files", "audio_count", len(audioDir), "text_count", len(textLines))
		http.Error(w, "Mismatch between audio and text files", http.StatusInternalServerError)
		return
	}
	for i, line := range textLines {
		// Remove the markers for the words and indicators.
		if strings.TrimSpace(line) == "" {
			lines = append(lines, Line{
				Text:     "",
				AudioURL: nil,
			})
			continue
		}
		// For other files, the | marks special words. In this case, we remove it.
		plainLine := strings.ReplaceAll(line, "|", "")
		var audioFile *string
		// Ignore the audio file if it doesn't exist.
		if i < len(audioDir) {
			// Get the file path
			temp := fmt.Sprintf("/static/stories/stories_audio/%s/%s", storyID, audioDir[i].Name())
			audioFile = &temp
		}
		lines = append(lines, Line{
			Text:     plainLine,
			AudioURL: audioFile,
		})
	}
	data.StoryID = storyID
	data.StoryTitle = storyTitle
	data.Lines = lines

	// Get the absolute path to the template file
	templatePath, err := filepath.Abs("src/templates/page1.html")
	if err != nil {
		h.log.Error("Failed to find template", "error", err)
		http.Error(w, "Failed to find template", http.StatusInternalServerError)
		return
	}

	// Parse and execute the template
	tmpl, err := template.ParseFiles(templatePath)
	if err != nil {
		h.log.Error("Failed to parse template", "error", err)
		http.Error(w, "Failed to parse template", http.StatusInternalServerError)
		return
	}
	// Execute and serve the page
	err = tmpl.Execute(w, data)
	if err != nil {
		h.log.Error("Failed to execute template", "error", err)
		http.Error(w, "Failed to execute template", http.StatusInternalServerError)
		return
	}
}
