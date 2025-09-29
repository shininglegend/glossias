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

// ScoreData represents the scoring data for a story
type ScoreData struct {
	StoryTitle             string  `json:"story_title"`
	TotalTimeSeconds       int     `json:"total_time_seconds"`
	VocabAccuracy          float64 `json:"vocab_accuracy"` // Percentage (0-100)
	VocabTimeSeconds       int     `json:"vocab_time_seconds"`
	GrammarAccuracy        float64 `json:"grammar_accuracy"` // Percentage (0-100)
	GrammarTimeSeconds     int     `json:"grammar_time_seconds"`
	TranslationTimeSeconds int     `json:"translation_time_seconds"`
	VideoTimeSeconds       int     `json:"video_time_seconds"`
}

// MissingActivity represents an incomplete activity
type MissingActivity struct {
	Activity    string `json:"activity"`     // "vocab", "grammar", "translation"
	DisplayName string `json:"display_name"` // "Vocabulary", "Grammar", "Translation"
	Route       string `json:"route"`        // "vocab", "grammar", "translate"
	Reason      string `json:"reason"`       // "no_data" or "insufficient_time"
}

// IncompleteDataResponse represents response when data is missing
type IncompleteDataResponse struct {
	Complete          bool              `json:"complete"`
	StoryTitle        string            `json:"story_title"`
	MissingActivities []MissingActivity `json:"missing_activities"`
	Message           string            `json:"message"`
}

func (h *Handler) GetScoresData(w http.ResponseWriter, r *http.Request) {
	storyID := mux.Vars(r)["id"]
	id, err := strconv.Atoi(storyID)
	if err != nil {
		h.sendError(w, "Invalid story ID format", http.StatusBadRequest)
		return
	}

	userID := auth.GetUserID(r)

	// Get story data for title
	story, err := models.GetStoryData(r.Context(), id, userID)
	if err == models.ErrNotFound {
		h.sendError(w, "Story not found", http.StatusNotFound)
		return
	}
	if err != nil {
		h.log.Error("Failed to fetch story data", "error", err, "storyID", id)
		h.sendError(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Get vocab accuracy
	vocabSummary, err := models.GetUserStoryVocabSummary(r.Context(), userID, int32(id))
	if err != nil {
		h.log.Error("Failed to fetch vocab summary", "error", err, "storyID", id, "userID", userID)
		h.sendError(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	var vocabAccuracy float64
	if vocabSummary.TotalAttempted > 0 {
		vocabAccuracy = float64(vocabSummary.CorrectCount) / float64(vocabSummary.TotalAttempted) * 100
	}

	// Get grammar accuracy
	grammarSummary, err := models.GetUserStoryGrammarSummary(r.Context(), userID, int32(id))
	if err != nil {
		h.log.Error("Failed to fetch grammar summary", "error", err, "storyID", id, "userID", userID)
		h.sendError(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	var grammarAccuracy float64
	if grammarSummary.TotalAttempted > 0 {
		grammarAccuracy = float64(grammarSummary.CorrectCount) / float64(grammarSummary.TotalAttempted) * 100
	}

	// Get time tracking data
	timeData, err := models.GetUserStoryTimeTracking(r.Context(), userID, int32(id))
	if err != nil {
		h.log.Error("Failed to fetch time tracking data", "error", err, "storyID", id, "userID", userID)
		h.sendError(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Check for missing or insufficient data
	var missingActivities []MissingActivity
	const minTimeSeconds = 5

	// Check vocab
	if vocabSummary.TotalAttempted == 0 {
		missingActivities = append(missingActivities, MissingActivity{
			Activity:    "vocab",
			DisplayName: "Vocabulary",
			Route:       "vocab",
			Reason:      "no_data",
		})
	} else if timeData.VocabTimeSeconds < minTimeSeconds {
		missingActivities = append(missingActivities, MissingActivity{
			Activity:    "vocab",
			DisplayName: "Vocabulary",
			Route:       "vocab",
			Reason:      "insufficient_time",
		})
	}

	// Check grammar
	if grammarSummary.TotalAttempted == 0 {
		missingActivities = append(missingActivities, MissingActivity{
			Activity:    "grammar",
			DisplayName: "Grammar",
			Route:       "grammar",
			Reason:      "no_data",
		})
	} else if timeData.GrammarTimeSeconds < minTimeSeconds {
		missingActivities = append(missingActivities, MissingActivity{
			Activity:    "grammar",
			DisplayName: "Grammar",
			Route:       "grammar",
			Reason:      "insufficient_time",
		})
	}

	// Check translation
	if timeData.TranslationTimeSeconds < minTimeSeconds {
		missingActivities = append(missingActivities, MissingActivity{
			Activity:    "translation",
			DisplayName: "Translation",
			Route:       "translate",
			Reason:      "insufficient_time",
		})
	}

	// If data is incomplete, return missing activities response
	if len(missingActivities) > 0 {
		incompleteResponse := IncompleteDataResponse{
			Complete:          false,
			StoryTitle:        story.Metadata.Title["en"],
			MissingActivities: missingActivities,
			Message:           "Please complete the missing activities to view your scores",
		}

		response := types.APIResponse{
			Success: true,
			Data:    incompleteResponse,
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	}

	// Calculate total time
	totalTime := timeData.VocabTimeSeconds + timeData.GrammarTimeSeconds +
		timeData.TranslationTimeSeconds + timeData.VideoTimeSeconds

	scoreData := ScoreData{
		StoryTitle:             story.Metadata.Title["en"],
		TotalTimeSeconds:       totalTime,
		VocabAccuracy:          vocabAccuracy,
		VocabTimeSeconds:       timeData.VocabTimeSeconds,
		GrammarAccuracy:        grammarAccuracy,
		GrammarTimeSeconds:     timeData.GrammarTimeSeconds,
		TranslationTimeSeconds: timeData.TranslationTimeSeconds,
		VideoTimeSeconds:       timeData.VideoTimeSeconds,
	}

	response := types.APIResponse{
		Success: true,
		Data:    scoreData,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
