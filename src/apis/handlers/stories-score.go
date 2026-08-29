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

// ScoreData is the student's result for a completed story: one accuracy per
// five-phase activity (Identify, Produce, Recall), a per-phase time breakdown,
// and the legacy Vocabulary/Grammar counts for stories authored before the
// five-phase flow (the frontend shows those cards only when attempts exist).
type ScoreData struct {
	StoryTitle       string  `json:"story_title"`
	TotalTimeSeconds int     `json:"total_time_seconds"`
	OverallAccuracy  float64 `json:"overall_accuracy"` // Percentage (0-100)

	IdentifyAccuracy       float64 `json:"identify_accuracy"` // Percentage (0-100)
	IdentifyCorrectCount   int     `json:"identify_correct_count"`
	IdentifyIncorrectCount int     `json:"identify_incorrect_count"`
	IdentifyTotal          int     `json:"identify_total"`

	ProduceScore             float64 `json:"produce_score"` // AI average over graded segments (0-100)
	ProduceSegmentsSubmitted int     `json:"produce_segments_submitted"`
	ProduceSegmentsGraded    int     `json:"produce_segments_graded"`
	ProduceTotal             int     `json:"produce_total"`

	RecallAccuracy       float64 `json:"recall_accuracy"` // Percentage (0-100)
	RecallCorrectCount   int     `json:"recall_correct_count"`
	RecallIncorrectCount int     `json:"recall_incorrect_count"`
	RecallAttempts       int     `json:"recall_attempts"`
	RecallTotal          int     `json:"recall_total"`

	VocabAccuracy         float64 `json:"vocab_accuracy"` // Percentage (0-100)
	VocabCorrectCount     int     `json:"vocab_correct_count"`
	VocabIncorrectCount   int     `json:"vocab_incorrect_count"`
	GrammarAccuracy       float64 `json:"grammar_accuracy"` // Percentage (0-100)
	GrammarCorrectCount   int     `json:"grammar_correct_count"`
	GrammarIncorrectCount int     `json:"grammar_incorrect_count"`

	VideoTimeSeconds       int `json:"video_time_seconds"`
	IdentifyTimeSeconds    int `json:"identify_time_seconds"`
	TranslationTimeSeconds int `json:"translation_time_seconds"`
	ProduceTimeSeconds     int `json:"produce_time_seconds"`
	RecallTimeSeconds      int `json:"recall_time_seconds"`
	VocabTimeSeconds       int `json:"vocab_time_seconds"`
	GrammarTimeSeconds     int `json:"grammar_time_seconds"`
}

// MissingActivity represents an incomplete activity
type MissingActivity struct {
	Activity    string `json:"activity"`     // "identify", "translation", "produce", "recall"
	DisplayName string `json:"display_name"` // "Identify", "Translation", "Produce", "Recall"
	Route       string `json:"route"`        // "identify", "translate", "produce", "recall"
	Reason      string `json:"reason"`       // "no_data" (never started) or "incomplete" (started, not finished)
}

// IncompleteDataResponse represents response when data is missing
type IncompleteDataResponse struct {
	Complete          bool              `json:"complete"`
	StoryTitle        string            `json:"story_title"`
	MissingActivities []MissingActivity `json:"missing_activities"`
	Message           string            `json:"message"`
}

// GetScoresData serves GET /api/stories/{id}/scores. A phase blocks the score
// page only if the story has content for it (a story missing Identify words,
// Produce segments or Recall sentences degrades to fewer cards, not a wall).
// Legacy vocab/grammar pages are not in the flow and never block.
func (h *Handler) GetScoresData(w http.ResponseWriter, r *http.Request) {
	storyID := mux.Vars(r)["id"]
	id, err := strconv.Atoi(storyID)
	if err != nil {
		h.sendError(w, "Invalid story ID format", http.StatusBadRequest)
		return
	}

	userID := auth.GetUserID(r)

	// Story data for the title and the access check.
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
	title := story.Metadata.Title["en"]

	completion, err := models.GetUserStoryPageCompletion(r.Context(), userID, id)
	if err != nil {
		h.log.Error("Failed to fetch page completion", "error", err, "storyID", id, "userID", userID)
		h.sendError(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	summary, err := models.GetUserStoryScoreSummary(r.Context(), userID, id)
	if err != nil {
		h.log.Error("Failed to fetch score summary", "error", err, "storyID", id, "userID", userID)
		h.sendError(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	missing := missingActivities(completion, summary)
	if len(missing) > 0 {
		h.writeJSON(w, types.APIResponse{
			Success: true,
			Data: IncompleteDataResponse{
				Complete:          false,
				StoryTitle:        title,
				MissingActivities: missing,
				Message:           "Please complete the missing activities to view your scores",
			},
		})
		return
	}

	timeData, err := models.GetUserStoryTimeTracking(r.Context(), userID, int32(id))
	if err != nil {
		h.log.Error("Failed to fetch time tracking data", "error", err, "storyID", id, "userID", userID)
		h.sendError(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	legacy := getVocabAndGrammarCount(*story)
	totals := models.PhaseTotals{
		VocabTotal:    int(legacy.VocabCount),
		GrammarTotal:  int(legacy.GrammarCount),
		IdentifyTotal: completion.IdentifyTotal,
		ProduceTotal:  completion.ProduceTotal,
		RecallTotal:   completion.RecallTotal,
	}
	scores := models.ComputePhaseScores(*summary, totals)

	totalTime := timeData.VideoTimeSeconds + timeData.IdentifyTimeSeconds +
		timeData.TranslationTimeSeconds + timeData.ProduceTimeSeconds + timeData.RecallTimeSeconds +
		timeData.VocabTimeSeconds + timeData.GrammarTimeSeconds

	h.writeJSON(w, types.APIResponse{
		Success: true,
		Data: ScoreData{
			StoryTitle:       title,
			TotalTimeSeconds: totalTime,
			OverallAccuracy:  scores.Overall,

			IdentifyAccuracy:       scores.IdentifyAccuracy,
			IdentifyCorrectCount:   summary.IdentifyCorrect,
			IdentifyIncorrectCount: summary.IdentifyIncorrect,
			IdentifyTotal:          totals.IdentifyTotal,

			ProduceScore:             scores.ProduceScore,
			ProduceSegmentsSubmitted: summary.ProduceSubmitted,
			ProduceSegmentsGraded:    summary.ProduceGraded,
			ProduceTotal:             totals.ProduceTotal,

			RecallAccuracy:       scores.RecallAccuracy,
			RecallCorrectCount:   summary.RecallCorrect,
			RecallIncorrectCount: summary.RecallIncorrect,
			RecallAttempts:       scores.RecallAttempts,
			RecallTotal:          totals.RecallTotal,

			VocabAccuracy:         scores.VocabAccuracy,
			VocabCorrectCount:     summary.VocabCorrect,
			VocabIncorrectCount:   summary.VocabIncorrect,
			GrammarAccuracy:       scores.GrammarAccuracy,
			GrammarCorrectCount:   summary.GrammarCorrect,
			GrammarIncorrectCount: summary.GrammarIncorrect,

			VideoTimeSeconds:       timeData.VideoTimeSeconds,
			IdentifyTimeSeconds:    timeData.IdentifyTimeSeconds,
			TranslationTimeSeconds: timeData.TranslationTimeSeconds,
			ProduceTimeSeconds:     timeData.ProduceTimeSeconds,
			RecallTimeSeconds:      timeData.RecallTimeSeconds,
			VocabTimeSeconds:       timeData.VocabTimeSeconds,
			GrammarTimeSeconds:     timeData.GrammarTimeSeconds,
		},
	})
}

// missingActivities lists the five-phase activities that still block the score
// page, in flow order. "no_data" means never started; "incomplete" means
// started but not finished (the frontend labels the button accordingly).
func missingActivities(c *models.PageCompletion, s *models.UserStoryScoreSummary) []MissingActivity {
	var missing []MissingActivity
	add := func(activity, display, route string, started, done bool) {
		if done {
			return
		}
		reason := "incomplete"
		if !started {
			reason = "no_data"
		}
		missing = append(missing, MissingActivity{Activity: activity, DisplayName: display, Route: route, Reason: reason})
	}

	if c.IdentifyTotal > 0 {
		add("identify", "Identify", "identify", s.IdentifyCorrect+s.IdentifyIncorrect > 0, c.IdentifyComplete())
	}
	add("translation", "Translation", "translate", c.TranslationCompleted, c.TranslateComplete())
	if c.ProduceTotal > 0 {
		add("produce", "Produce", "produce", s.ProduceSubmitted > 0, c.ProduceComplete())
	}
	if c.RecallTotal > 0 {
		add("recall", "Recall", "recall", s.RecallCorrect+s.RecallIncorrect > 0, c.RecallComplete())
	}
	return missing
}

func (h *Handler) writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}

func getVocabAndGrammarCount(story models.Story) struct{ VocabCount, GrammarCount int64 } {
	counts := struct{ VocabCount, GrammarCount int64 }{}
	for _, line := range story.Content.Lines {
		counts.VocabCount += int64(len(line.Vocabulary))
		counts.GrammarCount += int64(len(line.Grammar))
	}
	return counts
}
