package handlers

import (
	"encoding/json"
	"glossias/src/apis/types"
	"glossias/src/auth"
	"glossias/src/pkg/models"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
)

// ScoreData is the student's result for one attempt at a story: one accuracy
// per five-phase activity (Identify, Produce, Recall) and a per-phase time
// breakdown. Archived vocab/grammar results are never shown, even when the
// student has leftover scores from those pages.
type ScoreData struct {
	StoryTitle       string  `json:"story_title"`
	TotalTimeSeconds int     `json:"total_time_seconds"`
	OverallAccuracy  float64 `json:"overall_accuracy"` // Percentage (0-100)

	IdentifyAccuracy       float64 `json:"identify_accuracy"` // Percentage (0-100)
	IdentifyCorrectCount   int     `json:"identify_correct_count"`
	IdentifyIncorrectCount int     `json:"identify_incorrect_count"`
	IdentifyTotal          int     `json:"identify_total"`

	ProduceScore             float64                        `json:"produce_score"` // AI average over graded segments (0-100)
	ProduceSegmentsSubmitted int                            `json:"produce_segments_submitted"`
	ProduceSegmentsGraded    int                            `json:"produce_segments_graded"`
	ProduceTotal             int                            `json:"produce_total"`
	ProduceSegments          []models.AttemptProduceSegment `json:"produce_segments"`

	RecallAccuracy       float64 `json:"recall_accuracy"` // Percentage (0-100)
	RecallCorrectCount   int     `json:"recall_correct_count"`
	RecallIncorrectCount int     `json:"recall_incorrect_count"`
	RecallAttempts       int     `json:"recall_attempts"`
	RecallTotal          int     `json:"recall_total"`

	VideoTimeSeconds       int `json:"video_time_seconds"`
	IdentifyTimeSeconds    int `json:"identify_time_seconds"`
	TranslationTimeSeconds int `json:"translation_time_seconds"`
	ProduceTimeSeconds     int `json:"produce_time_seconds"`
	RecallTimeSeconds      int `json:"recall_time_seconds"`

	// AttemptNumber is the attempt shown; Attempts lists every one the
	// student can switch to. Archived is false only for a finished attempt
	// still waiting on Produce grading, which is served live and frozen on a
	// later visit.
	AttemptNumber int            `json:"attempt_number"`
	Attempts      []ScoreAttempt `json:"attempts"`
	Archived      bool           `json:"archived"`
}

// ScoreAttempt is one entry on the Score page's attempt picker.
type ScoreAttempt struct {
	Number      int        `json:"number"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
	Pending     bool       `json:"pending,omitempty"`
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

// GetScoresData serves GET /api/stories/{id}/scores[?attempt=N].
//
// Finishing a story is what archives it: when the live rows are complete (and
// Produce grading has settled) this handler freezes them as the next attempt
// and wipes the exercises, so the student's next visit to Identify starts a
// fresh run without being asked. The page then shows the requested attempt,
// defaulting to the newest, with every archived attempt selectable.
//
// A phase blocks the page only if the story has content for it (a story
// missing Identify words, Produce segments or Recall sentences degrades to
// fewer cards, not a wall). Legacy vocab/grammar pages never block.
func (h *Handler) GetScoresData(w http.ResponseWriter, r *http.Request) {
	storyID := mux.Vars(r)["id"]
	id, err := strconv.Atoi(storyID)
	if err != nil {
		h.sendError(w, "Invalid story ID format", http.StatusBadRequest)
		return
	}

	requested := 0
	if raw := r.URL.Query().Get("attempt"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n < 1 {
			h.sendError(w, "Invalid attempt parameter", http.StatusBadRequest)
			return
		}
		requested = n
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

	livePending := models.StoryExercisesComplete(completion)
	var justArchived *models.AttemptScoreSnapshot
	justArchivedNumber := 0
	if models.ReadyToArchive(completion, summary) {
		justArchivedNumber, justArchived, err = models.ArchiveCompletedAttempt(r.Context(), userID, int32(id), title, summary)
		if err != nil {
			h.log.Error("Failed to archive completed story attempt", "error", err, "storyID", id, "userID", userID)
			h.sendError(w, "Internal server error", http.StatusInternalServerError)
			return
		}
		h.log.Info("Story attempt archived", "userID", userID, "storyID", id, "attempt", justArchivedNumber)
		livePending = false
	}

	attempts, err := models.ListUserStoryAttempts(r.Context(), userID, int32(id))
	if err != nil {
		h.log.Error("Failed to list story attempts", "error", err, "storyID", id, "userID", userID)
		h.sendError(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	current := attempts[len(attempts)-1].Number
	picker := make([]ScoreAttempt, 0, len(attempts))
	for _, a := range attempts[:len(attempts)-1] {
		picker = append(picker, ScoreAttempt{Number: a.Number, CompletedAt: a.CompletedAt})
	}
	if livePending {
		picker = append(picker, ScoreAttempt{Number: current, Pending: true})
	}

	if len(picker) == 0 {
		if requested != 0 {
			h.sendError(w, "Attempt not found", http.StatusNotFound)
			return
		}
		h.writeJSON(w, types.APIResponse{
			Success: true,
			Data: IncompleteDataResponse{
				Complete:          false,
				StoryTitle:        title,
				MissingActivities: missingActivities(completion, summary),
				Message:           "Please complete the missing activities to view your scores",
			},
		})
		return
	}
	if requested == 0 {
		requested = picker[len(picker)-1].Number
	}

	var snap *models.AttemptScoreSnapshot
	switch {
	case justArchived != nil && requested == justArchivedNumber:
		snap = justArchived
	case livePending && requested == current:
		snap, err = models.BuildLiveAttemptScore(r.Context(), userID, id, title, summary)
	case requested < current:
		snap, err = models.GetAttemptScore(r.Context(), userID, id, requested)
	}
	if err != nil {
		h.log.Error("Failed to load attempt score", "error", err, "storyID", id, "userID", userID, "attempt", requested)
		h.sendError(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	if snap == nil {
		h.sendError(w, "Attempt not found", http.StatusNotFound)
		return
	}

	data := scoreDataFromSnapshot(*snap)
	data.AttemptNumber = requested
	data.Attempts = picker
	data.Archived = requested < current
	h.writeJSON(w, types.APIResponse{Success: true, Data: data})
}

// scoreDataFromSnapshot projects the frozen payload onto the student API,
// dropping the archived vocab/grammar fields the snapshot keeps for admins.
func scoreDataFromSnapshot(s models.AttemptScoreSnapshot) ScoreData {
	b, _ := json.Marshal(s)
	var d ScoreData
	_ = json.Unmarshal(b, &d)
	if d.ProduceSegments == nil {
		d.ProduceSegments = []models.AttemptProduceSegment{}
	}
	return d
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
