package apis

import (
	"encoding/json"
	"fmt"
	"glossias/internal/pkg/models"
	"log/slog"
	"net/http"
	"os"
	"slices"
	"strconv"

	"github.com/gorilla/mux"
)

const storiesDir = "static/stories/"
const vocabBlank = "%"

type Handler struct {
	log *slog.Logger
}

func NewHandler(logger *slog.Logger) *Handler {
	return &Handler{
		log: logger,
	}
}

// RegisterRoutes registers all API routes
func (h *Handler) RegisterRoutes(mux *mux.Router) {
	apiRouter := mux.PathPrefix("/api").Subrouter()
	apiRouter.HandleFunc("/stories", h.GetStories).Methods("GET")
	apiRouter.HandleFunc("/stories/{id}/page1", h.GetPage1).Methods("GET")
	apiRouter.HandleFunc("/stories/{id}/page2", h.GetPage2).Methods("GET")
	apiRouter.HandleFunc("/stories/{id}/page3", h.GetPage3).Methods("GET")
	apiRouter.HandleFunc("/stories/{id}/check-vocab", h.CheckVocab).Methods("POST")
}

// GetStories returns JSON array of all stories
func (h *Handler) GetStories(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Get stories from database (reusing existing models function)
	dbStories, err := models.GetAllStories(r.URL.Query().Get("lang"))
	if err != nil {
		h.log.Error("Failed to fetch stories from database", "error", err)
		h.sendError(w, "Failed to fetch stories", http.StatusInternalServerError)
		return
	}

	// Convert to API format
	stories := ConvertStoriesToAPI(dbStories)
	response := APIResponse{
		Success: true,
		Data: StoriesResponse{
			Stories: stories,
		},
	}

	json.NewEncoder(w).Encode(response)
}

// GetPage1 returns JSON data for story page 1
func (h *Handler) GetPage1(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	storyID := mux.Vars(r)["id"]
	id, err := strconv.Atoi(storyID)
	if err != nil {
		h.sendError(w, "Invalid story ID format", http.StatusBadRequest)
		return
	}

	// Get story data (reusing existing models function)
	story, err := models.GetStoryData(id)
	if err == models.ErrNotFound {
		h.sendError(w, "Story not found", http.StatusNotFound)
		return
	}
	if err != nil {
		h.log.Error("Failed to fetch story data", "error", err)
		h.sendError(w, "Failed to fetch story data", http.StatusInternalServerError)
		return
	}

	// Process lines (reusing logic from original handler)
	lines := h.processLinesForPage1(*story, id)

	data := PageData{
		StoryID:    storyID,
		StoryTitle: story.Metadata.Title["en"],
		Lines:      lines,
	}

	response := APIResponse{
		Success: true,
		Data:    data,
	}

	json.NewEncoder(w).Encode(response)
}

// GetPage2 returns JSON data for story page 2 (vocabulary)
func (h *Handler) GetPage2(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	storyID := mux.Vars(r)["id"]
	id, err := strconv.Atoi(storyID)
	if err != nil {
		h.sendError(w, "Invalid story ID format", http.StatusBadRequest)
		return
	}

	story, err := models.GetStoryData(id)
	if err == models.ErrNotFound {
		h.sendError(w, "Story not found", http.StatusNotFound)
		return
	}
	if err != nil {
		h.log.Error("Failed to fetch story data", "error", err)
		h.sendError(w, "Failed to fetch story data", http.StatusInternalServerError)
		return
	}

	// Process lines and vocab bank (reusing logic from original handler)
	lines, vocabBank := h.processLinesForPage2(*story, id)

	data := Page2Data{
		PageData: PageData{
			StoryID:    storyID,
			StoryTitle: story.Metadata.Title["en"],
			Lines:      lines,
		},
		VocabBank: vocabBank,
	}

	response := APIResponse{
		Success: true,
		Data:    data,
	}

	json.NewEncoder(w).Encode(response)
}

// GetPage3 returns JSON data for story page 3 (grammar)
func (h *Handler) GetPage3(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	storyID := mux.Vars(r)["id"]
	id, err := strconv.Atoi(storyID)
	if err != nil {
		h.sendError(w, "Invalid story ID format", http.StatusBadRequest)
		return
	}

	story, err := models.GetStoryData(id)
	if err == models.ErrNotFound {
		h.sendError(w, "Story not found", http.StatusNotFound)
		return
	}
	if err != nil {
		h.log.Error("Failed to fetch story data", "error", err)
		h.sendError(w, "Failed to fetch story data", http.StatusInternalServerError)
		return
	}

	// Process lines for grammar (reusing logic from original handler)
	lines := h.processLinesForPage3(*story, id)

	data := Page3Data{
		PageData: PageData{
			StoryID:    storyID,
			StoryTitle: story.Metadata.Title["en"],
			Lines:      lines,
		},
		GrammarPoint: story.Metadata.GrammarPoint,
	}

	response := APIResponse{
		Success: true,
		Data:    data,
	}

	json.NewEncoder(w).Encode(response)
}

// CheckVocab handles vocabulary checking (JSON version of existing handler)
func (h *Handler) CheckVocab(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var req CheckVocabRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.log.Warn("Invalid request body in CheckVocab", "error", err, "ip", r.RemoteAddr)
		h.sendError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	storyID := mux.Vars(r)["id"]
	id, err := strconv.Atoi(storyID)
	if err != nil {
		h.log.Warn("Invalid story ID in CheckVocab", "storyID", storyID, "ip", r.RemoteAddr)
		h.sendError(w, "Invalid story ID", http.StatusBadRequest)
		return
	}

	// Get story from database (reusing existing models function)
	story, err := models.GetStoryData(id)
	if err != nil {
		h.log.Error("Failed to fetch story in CheckVocab", "error", err, "storyID", id)
		h.sendError(w, "Failed to fetch story", http.StatusInternalServerError)
		return
	}

	// Build expected answers map for all lines
	expectedAnswers := make(map[int]int)
	for i, line := range story.Content.Lines {
		expectedAnswers[i] = len(line.Vocabulary)
	}

	var results []VocabResult

	// Process each line's answers
	for _, answer := range req.Answers {
		// Validate line number
		if answer.LineNumber < 0 || answer.LineNumber >= len(story.Content.Lines) {
			h.log.Warn("Invalid line number in CheckVocab", "lineNumber", answer.LineNumber, "maxLines", len(story.Content.Lines), "ip", r.RemoteAddr)
			h.sendValidationError(w, fmt.Sprintf("Invalid line number: %d", answer.LineNumber), expectedAnswers)
			return
		}

		line := story.Content.Lines[answer.LineNumber]

		// Build correct answers map and debug log all correct answers for this line
		correctAnswers := make(map[string]bool)
		expectedVocabCount := len(line.Vocabulary)

		for _, vocab := range line.Vocabulary {
			correctAnswers[vocab.LexicalForm] = true
		}

		h.log.Debug("Correct answers for line", "lineNumber", answer.LineNumber, "correctAnswers", correctAnswers)

		// Validate answer count - user must provide answers for ALL vocab words in the line
		if len(answer.Answers) != expectedVocabCount {
			h.log.Warn("Wrong number of answers provided", "lineNumber", answer.LineNumber, "expected", expectedVocabCount, "provided", len(answer.Answers), "ip", r.RemoteAddr)
			h.sendValidationError(w, fmt.Sprintf("Line %d expects %d answers, got %d", answer.LineNumber, expectedVocabCount, len(answer.Answers)), expectedAnswers)
			return
		}

		// Process each answer for this line
		for i, userAnswer := range answer.Answers {
			isCorrect := correctAnswers[userAnswer]

			// Find the correct answer for this position
			var correctAnswer string
			if i < len(line.Vocabulary) {
				correctAnswer = line.Vocabulary[i].LexicalForm
			}

			results = append(results, VocabResult{
				Correct:       isCorrect,
				UserAnswer:    userAnswer,
				CorrectAnswer: correctAnswer,
				Line:          answer.LineNumber,
			})
		}
	}

	h.log.Debug("Vocab check completed", "ip", r.RemoteAddr, "totalResults", len(results))

	response := APIResponse{
		Success: true,
		Data: CheckVocabResponse{
			Answers: results,
		},
	}

	json.NewEncoder(w).Encode(response)
}

// Helper functions (extracted from original handlers)

func (h *Handler) processLinesForPage1(story models.Story, id int) []Line {
	lines := make([]Line, 0, len(story.Content.Lines))

	// Load audio files
	folderPath := fmt.Sprintf(storiesDir+"stories_audio/%v_%v%v",
		story.Metadata.Description.Language,
		story.Metadata.WeekNumber,
		story.Metadata.DayLetter)
	audioDir, err := os.ReadDir(folderPath)

	for i, dbLine := range story.Content.Lines {
		var audioFile *string
		if err == nil && i < len(audioDir) {
			temp := fmt.Sprintf("/%v/%v", folderPath, audioDir[i].Name())
			audioFile = &temp
		}

		lines = append(lines, Line{
			Text:     []string{dbLine.Text},
			AudioURL: audioFile,
		})
	}

	return lines
}

func (h *Handler) processLinesForPage2(story models.Story, id int) ([]Line, []string) {
	lines := make([]Line, len(story.Content.Lines))
	vocabBank := make([]string, 0)

	// Load audio files
	folderPath := fmt.Sprintf(storiesDir+"stories_audio/%v_%v%v",
		story.Metadata.Description.Language,
		story.Metadata.WeekNumber,
		story.Metadata.DayLetter)
	audioDir, err := os.ReadDir(folderPath)

	for i, line := range story.Content.Lines {
		series := []string{}
		runes := []rune(line.Text)
		lastEnd := 0

		// Sort vocab words
		slices.SortFunc(line.Vocabulary, h.sortVocab)

		for j := 0; j < len(line.Vocabulary); j++ {
			vocab := line.Vocabulary[j]
			vocabBank = append(vocabBank, vocab.LexicalForm)
			start := vocab.Position[0]
			if start >= lastEnd {
				series = append(series, string(runes[lastEnd:start]))
			}
			series = append(series, vocabBlank)
			lastEnd = vocab.Position[1]
		}
		if lastEnd < len(runes) {
			series = append(series, string(runes[lastEnd:]))
		}
		hasVocab := len(line.Vocabulary) > 0

		var audioFile *string
		if err == nil && i < len(audioDir) {
			temp := fmt.Sprintf("/%v/%v", folderPath, audioDir[i].Name())
			audioFile = &temp
		}

		lines[i] = Line{
			Text:              series,
			AudioURL:          audioFile,
			HasVocabOrGrammar: hasVocab,
		}
	}

	// Sort and dedupe vocab bank
	slices.Sort(vocabBank)
	vocabBank = slices.Compact(vocabBank)

	return lines, vocabBank
}

func (h *Handler) processLinesForPage3(story models.Story, id int) []Line {
	lines := make([]Line, len(story.Content.Lines))

	// Load audio files
	folderPath := fmt.Sprintf(storiesDir+"stories_audio/%v_%v%v",
		story.Metadata.Description.Language,
		story.Metadata.WeekNumber,
		story.Metadata.DayLetter)
	audioDir, err := os.ReadDir(folderPath)

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

		for _, grammar := range line.Grammar {
			start := grammar.Position[0]
			if start >= lastEnd {
				series = append(series, string(runes[lastEnd:start]))
			}
			series = append(series, "%", grammar.Text, "&")
			lastEnd = grammar.Position[1]
		}

		if lastEnd < len(runes) {
			series = append(series, string(runes[lastEnd:]))
		}

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

	return lines
}

func (h *Handler) sortVocab(a, b models.VocabularyItem) int {
	if a.Position[0] < b.Position[0] {
		return -1
	}
	if a.Position[0] > b.Position[0] {
		return 1
	}
	return 0
}

func (h *Handler) sendError(w http.ResponseWriter, message string, status int) {
	w.WriteHeader(status)
	response := APIResponse{
		Success: false,
		Error:   message,
	}
	json.NewEncoder(w).Encode(response)
}

func (h *Handler) sendValidationError(w http.ResponseWriter, message string, expectedAnswers map[int]int) {
	w.WriteHeader(http.StatusBadRequest)
	response := APIResponse{
		Success: false,
		Error:   message,
		Data: LineValidationError{
			Message:         message,
			ExpectedAnswers: expectedAnswers,
		},
	}
	json.NewEncoder(w).Encode(response)
}
