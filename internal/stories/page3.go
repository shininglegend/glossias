package stories

import (
	"fmt"
	"logos-stories/internal/pkg/models"
	"net/http"
	"os"
	"slices"
	"strconv"

	"github.com/gorilla/mux"
)

type Page3Data struct {
	StoryID      string
	StoryTitle   string
	GrammarPoint string
	Lines        []Line
}

func (h *Handler) ServePage3(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	// Get story ID from URL
	id, err := strconv.Atoi(mux.Vars(r)["id"])
	if err != nil {
		http.Error(w, "Invalid story ID", http.StatusBadRequest)
		return
	}

	// Get story data
	story, err := models.GetStoryData(id)
	if err != nil {
		h.log.Error("Failed to fetch story", "error", err)
		http.Error(w, "Failed to fetch story", http.StatusInternalServerError)
		return
	}

	lines := make([]Line, len(story.Content.Lines))

	// Load audio files from folder
	folderPath := fmt.Sprintf(storiesDir+"stories_audio/%v_%v%v",
		story.Metadata.Description.Language,
		story.Metadata.WeekNumber,
		story.Metadata.DayLetter)
	audioDir, err := os.ReadDir(folderPath)
	if err != nil && !os.IsNotExist(err) {
		h.log.Error("Failed to read audio files", "error", err)
		http.Error(w, "Failed to read audio files", http.StatusInternalServerError)
		return
	}

	// Process each line for grammar points
	for i, line := range story.Content.Lines {
		series := []string{}
		runes := []rune(line.Text)
		lastEnd := 0

		// Sort grammar points by position
		slices.SortFunc(line.Grammar, func(a, b models.GrammarItem) int {
			if a.Position[0] < b.Position[0] {
				return -1
			}
			if a.Position[0] > b.Position[0] {
				return 1
			}
			return 0
		})

		// Process grammar points
		for _, grammar := range line.Grammar {
			start := grammar.Position[0]
			if start >= lastEnd {
				series = append(series, string(runes[lastEnd:start]))
			}
			// Mark grammar points with special character for template
			series = append(series, "%", grammar.Text, "&")
			lastEnd = grammar.Position[1]
		}

		if lastEnd < len(runes) {
			series = append(series, string(runes[lastEnd:]))
		}

		// Add audio if available
		var audioFile *string
		if err == nil && i < len(audioDir) {
			temp := fmt.Sprintf("/%v/%v", folderPath, audioDir[i].Name())
			audioFile = &temp
		}

		lines[i] = Line{
			Text:              series,
			AudioURL:          audioFile,
			HasVocabOrGrammar: len(line.Grammar) > 0,
		}
	}

	data := Page3Data{
		StoryID:      strconv.Itoa(id),
		StoryTitle:   story.Metadata.Title["en"],
		GrammarPoint: story.Metadata.GrammarPoint,
		Lines:        lines,
	}

	if err := h.te.Render(w, "page3.html", data); err != nil {
		h.log.Error("Failed to render page", "error", err)
		http.Error(w, "Failed to render page", http.StatusInternalServerError)
	}

	// TODO: Future enhancement - Allow users to select and identify grammar points
}
