package stories

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

type Line struct {
	Text     string
	AudioURL string
}

type PageData struct {
	Lines []Line
}

// Read from html file
func ServePage1(w http.ResponseWriter, r *http.Request) {
	// Get the story id from the URL
	storyID := r.URL.Query().Get("id")
	if storyID == "" {
		http.Error(w, fmt.Sprintf("Missing or invalid story ID. Got: '%v'", storyID), http.StatusBadRequest)
		return
	}

	lines := []Line{}
	// Load the audio files from the folder
	audioDir, err := os.ReadDir(fmt.Sprintf("src/data/stories_audio/%v", storyID))
	if err != nil {
		if err == os.ErrNotExist {
			http.Error(w, fmt.Sprintf("Story with ID '%v' not found", storyID), http.StatusNotFound)
			return
		}
		log.Fatal(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	// Load the text file
	textFile, err := os.Open(fmt.Sprintf("src/data/stories_text/%v.txt", storyID))
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to open text file for story with ID '%v'", storyID), http.StatusInternalServerError)
		return
	}
	defer textFile.Close()
	textLines := []string{}
	for {
		var line string
		_, err = fmt.Fscanln(textFile, &line)
		if err != nil {
			break
		}
		textLines = append(textLines, line)
	}
	if err != nil {
		log.(err)
		http.Error(w, fmt.Sprintf("Failed to read text file for story with ID '%v'", storyID), http.StatusInternalServerError)
		return
	}

	// Split by newlines, should equal audio line count (excluding blanks)
	if len(audioDir) > len(textLines) {
		http.Error(w, "Mismatch between audio and text files", http.StatusInternalServerError)
		return
	}
	for i, line := range textLines {
		// Remove the markers for the words and indicators.
		if strings.TrimSpace(line) == "" {
			lines = append(lines, Line{
				Text:     "",
				AudioURL: "",
			})
			continue
		}
		// For other files, the | marks special words. In this case, we remove it.
		plainLine := strings.ReplaceAll(line, "|", "")

		lines = append(lines, Line{
			Text:     plainLine,
			AudioURL: fmt.Sprintf("/static/stories_audio/%s", audioDir[i].Name()),
		})
	}
	// Add the text to the template.
	data := PageData{
		Lines: lines,
	}

	// Get the absolute path to the template file
	templatePath, err := filepath.Abs("src/templates/page1.html")
	if err != nil {
		http.Error(w, "Failed to find template", http.StatusInternalServerError)
		return
	}

	// Parse and execute the template
	tmpl, err := template.ParseFiles(templatePath)
	if err != nil {
		http.Error(w, "Failed to parse template", http.StatusInternalServerError)
		return
	}

	err = tmpl.Execute(w, data)
	if err != nil {
		http.Error(w, "Failed to execute template", http.StatusInternalServerError)
		return
	}
	// Serve the page
}
