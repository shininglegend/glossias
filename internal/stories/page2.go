// logos-stories/internal/stories/page2.go
package stories

import (
	"encoding/json"
	"fmt"
	"logos-stories/internal/pkg/models"
	"net/http"
	"os"
	"slices"
	"strconv"

	"github.com/gorilla/mux"
)

type Page2Data struct {
	StoryID    string
	StoryTitle string
	Lines      []Line
	VocabBank  []string
}

// Add these new types// page2.go
type VocabAnswer struct {
	LineNumber int      `json:"lineNumber"`
	Answers    []string `json:"answers"`
}

type CheckVocabRequest struct {
	Answers []VocabAnswer `json:"answers"`
}

type CheckVocabResponse struct {
	Answers []struct {
		Correct bool   `json:"correct"`
		Word    string `json:"word"`
	} `json:"answers"`
}

func (h *Handler) ServePage2(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	id, err := strconv.Atoi(mux.Vars(r)["id"])
	if err != nil {
		http.Error(w, "Invalid story ID", http.StatusBadRequest)
		return
	}

	story, err := models.GetStoryData(id)
	if err != nil {
		h.log.Error("Failed to fetch story", "error", err)
		http.Error(w, "Failed to fetch story", http.StatusInternalServerError)
		return
	}

	lines := make([]Line, len(story.Content.Lines))
	vocabBank := make([]string, 0)

	// Load the audio files from the folder (keeping existing audio handling)
	folderPath := fmt.Sprintf(storiesDir+"stories_audio/%v_%v%v", story.Metadata.Description.Language, story.Metadata.WeekNumber, story.Metadata.DayLetter)
	audioDir, err := os.ReadDir(folderPath)
	if err != nil && !os.IsNotExist(err) {
		h.log.Error("Failed to read audio files", "error", err)
		http.Error(w, "Failed to read audio files", http.StatusInternalServerError)
		return
	}

	// Process each line for vocabulary words
	for i, line := range story.Content.Lines {
		series := []string{}
		runes := []rune(line.Text)
		lastEnd := 0
		// Sort the vocab words
		slices.SortFunc(line.Vocabulary, sortVocab)

		for j := 0; j < len(line.Vocabulary); j++ {
			vocab := line.Vocabulary[j]
			vocabBank = append(vocabBank, vocab.LexicalForm)
			start := vocab.Position[0]
			// Only add non-empty segments
			if start >= lastEnd {
				// Add the text before the vocab word
				series = append(series, string(runes[lastEnd:start]))
			}
			series = append(series, "%")
			lastEnd = vocab.Position[1]
		}
		if lastEnd < len(runes) {
			series = append(series, string(runes[lastEnd:]))
		}
		hasVocab := len(line.Vocabulary) > 0

		var audioFile *string
		// Match audio files with lines if they exist
		if err == nil && i < len(audioDir) {
			temp := fmt.Sprintf("/%v/%v", folderPath, audioDir[i].Name())
			audioFile = &temp
		}

		lines[i] = Line{
			Text:     series,
			AudioURL: audioFile,
			HasVocab: hasVocab,
		}
	}

	// Sort the vocab bank
	slices.Sort(vocabBank)

	data := Page2Data{
		StoryID:    strconv.Itoa(id),
		StoryTitle: story.Metadata.Title["en"],
		Lines:      lines,
		VocabBank:  vocabBank,
	}

	if err := h.te.Render(w, "page2_go.html", data); err != nil {
		h.log.Error("Failed to render page", "error", err)
		http.Error(w, "Failed to render page", http.StatusInternalServerError)
	}
}

func (h *Handler) CheckVocabAnswers(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req CheckVocabRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Get story ID from URL
	id, err := strconv.Atoi(mux.Vars(r)["id"])
	if err != nil {
		http.Error(w, "Invalid story ID", http.StatusBadRequest)
		return
	}

	// Get story from database to check against
	story, err := models.GetStoryData(id)
	if err != nil {
		http.Error(w, "Failed to fetch story", http.StatusInternalServerError)
		return
	}

	resp := CheckVocabResponse{
		Answers: make([]struct {
			Correct bool   `json:"correct"`
			Word    string `json:"word"`
		}, 0),
	}

	// Check each line's answers against database
	for _, answer := range req.Answers {
		if answer.LineNumber < 0 || answer.LineNumber >= len(story.Content.Lines) {
			h.log.Warn("Invalid line number", "line number", answer.LineNumber)
			continue // Skip invalid line numbers
		}
		line := story.Content.Lines[answer.LineNumber]
		correctAnswers := make(map[string]bool)
		for _, vocab := range line.Vocabulary {
			correctAnswers[vocab.LexicalForm] = true
		}

		// Check each answer for this line
		for _, ans := range answer.Answers {
			isCorrect := correctAnswers[ans]
			resp.Answers = append(resp.Answers, struct {
				Correct bool   `json:"correct"`
				Word    string `json:"word"`
			}{
				Correct: isCorrect,
				Word:    ans,
			})
		}
		h.log.Debug("Checked line", "line number", answer.LineNumber, "correct answers", correctAnswers)
	}
	// Log the number of correct answers with the IP to the database and console
	h.log.Debug("Submission", "IP:", r.RemoteAddr, "Number of correct answers", len(resp.Answers))
	// TODO: Save to database

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func sortVocab(a, b models.VocabularyItem) int {
	if a.Position[0] < b.Position[0] {
		return -1 // Changed: return -1 for "less than"
	}
	if a.Position[0] > b.Position[0] {
		return 1 // Added: return 1 for "greater than"
	}
	return 0 // Added: return 0 for "equal"
}
