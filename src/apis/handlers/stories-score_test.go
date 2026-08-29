package handlers

import (
	"context"
	"encoding/json"
	"glossias/src/auth"
	"glossias/src/pkg/database"
	"glossias/src/pkg/models"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
	"github.com/jackc/pgx/v5/pgtype"
)

// stubScoreDB wires an uncoursed story 2 with the given page-completion row
// (GetUserStoryPageCompletion column order) and score-summary row
// (GetUserStoryScoreSummary column order).
func stubScoreDB(t *testing.T, completion, summary []any) {
	t.Helper()
	mockDB := database.NewMockDBTX()
	mockDB.StubQuery("name: GetStory :one", [][]any{{
		int32(2), int32(1), "A", pgtype.Text{}, pgtype.Timestamp{}, "author", "Author", pgtype.Int4{},
	}}, nil)
	mockDB.StubQuery("GetUserStoryPageCompletion", [][]any{completion}, nil)
	mockDB.StubQuery("GetUserStoryScoreSummary", [][]any{summary}, nil)
	mockDB.StubQuery("GetUserStoryTimeTracking", [][]any{{
		int64(0), int64(0), int64(150), int64(105), int64(210), int64(300), int64(135),
	}}, nil)
	models.SetDB(mockDB)
	t.Cleanup(func() { models.SetDB(struct{}{}) })
}

func scoreRequest() *http.Request {
	req := httptest.NewRequest("GET", "/api/stories/2/scores", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "2"})
	return req.WithContext(context.WithValue(req.Context(), auth.UserIDKey, "user-1"))
}

// Completion column order: identify total/correct, translate, recall
// total/correct, produce total/submitted.
// Summary column order: vocab c/i, grammar c/i, identify c/i, recall c/i,
// produce submitted/graded/avg.
var (
	fullStoryDone = []any{int32(6), int32(6), true, int32(5), int32(5), int32(2), int32(2)}
	fullSummary   = []any{int32(0), int32(0), int32(0), int32(0), int32(6), int32(2), int32(7), int32(3), int32(2), int32(2), float64(85)}
)

func TestGetScoresDataComplete(t *testing.T) {
	h := NewHandler(slog.New(slog.DiscardHandler), nil)
	stubScoreDB(t, fullStoryDone, fullSummary)

	// Budget: 10 for the uncached GetStoryData load (cached in production)
	// + completion + summary + time tracking.
	rr := assertQueryBudget(t, 13, h.GetScoresData, scoreRequest())
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rr.Code, rr.Body.String())
	}

	var resp struct {
		Data ScoreData `json:"data"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	d := resp.Data

	if d.IdentifyCorrectCount != 6 || d.IdentifyIncorrectCount != 2 || d.IdentifyTotal != 6 {
		t.Errorf("identify counts = %d/%d of %d", d.IdentifyCorrectCount, d.IdentifyIncorrectCount, d.IdentifyTotal)
	}
	if d.RecallAttempts != 2 {
		t.Errorf("recall attempts = %d, want 2 (10 rows / 5 sentences)", d.RecallAttempts)
	}
	if d.ProduceScore != 85 || d.ProduceSegmentsGraded != 2 {
		t.Errorf("produce = %v graded %d, want 85 / 2", d.ProduceScore, d.ProduceSegmentsGraded)
	}
	wantIdentify := models.CalculateScoreWithRetriesAllowed(6, 2, 6)
	wantRecall := models.CalculateScoreWithRetriesAllowed(7, 3, 5)
	wantOverall := (wantIdentify + wantRecall + 85) / 3
	if diff := d.OverallAccuracy - wantOverall; diff > 0.01 || diff < -0.01 {
		t.Errorf("overall = %v, want %v", d.OverallAccuracy, wantOverall)
	}
	// Five-phase breakdown feeds the total: 150+105+210+300+135.
	if d.TotalTimeSeconds != 900 {
		t.Errorf("total time = %d, want 900", d.TotalTimeSeconds)
	}
	// Time row column order: vocab, grammar, translation, video, identify, produce, recall.
	if d.VideoTimeSeconds != 105 || d.IdentifyTimeSeconds != 210 || d.RecallTimeSeconds != 135 {
		t.Errorf("video/identify/recall time = %d/%d/%d, want 105/210/135", d.VideoTimeSeconds, d.IdentifyTimeSeconds, d.RecallTimeSeconds)
	}
}

func TestGetScoresDataIncomplete(t *testing.T) {
	h := NewHandler(slog.New(slog.DiscardHandler), nil)

	tests := []struct {
		name       string
		completion []any
		summary    []any
		want       map[string]string // activity -> reason
	}{
		{
			name:       "fresh story lists every phase as not started",
			completion: []any{int32(6), int32(0), false, int32(5), int32(0), int32(2), int32(0)},
			summary:    []any{int32(0), int32(0), int32(0), int32(0), int32(0), int32(0), int32(0), int32(0), int32(0), int32(0), float64(0)},
			want:       map[string]string{"identify": "no_data", "translation": "no_data", "produce": "no_data", "recall": "no_data"},
		},
		{
			name:       "started phases are incomplete, not no_data",
			completion: []any{int32(6), int32(4), true, int32(5), int32(3), int32(2), int32(1)},
			summary:    []any{int32(0), int32(0), int32(0), int32(0), int32(4), int32(1), int32(3), int32(2), int32(1), int32(0), float64(0)},
			want:       map[string]string{"identify": "incomplete", "produce": "incomplete", "recall": "incomplete"},
		},
		{
			name: "story with no produce or recall content only blocks on what it has",
			// Mixed-generation: identify authored, produce/recall not yet.
			completion: []any{int32(6), int32(6), false, int32(0), int32(0), int32(0), int32(0)},
			summary:    []any{int32(0), int32(0), int32(0), int32(0), int32(6), int32(0), int32(0), int32(0), int32(0), int32(0), float64(0)},
			want:       map[string]string{"translation": "no_data"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stubScoreDB(t, tt.completion, tt.summary)
			rr := assertQueryBudget(t, 13, h.GetScoresData, scoreRequest())
			if rr.Code != http.StatusOK {
				t.Fatalf("status = %d, body = %s", rr.Code, rr.Body.String())
			}
			var resp struct {
				Data IncompleteDataResponse `json:"data"`
			}
			if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
				t.Fatalf("unmarshal: %v", err)
			}
			if resp.Data.Complete {
				t.Fatalf("expected incomplete response, got %s", rr.Body.String())
			}
			got := map[string]string{}
			for _, m := range resp.Data.MissingActivities {
				got[m.Activity] = m.Reason
				if m.Route == "" || m.DisplayName == "" {
					t.Errorf("activity %q missing route/display name", m.Activity)
				}
			}
			if len(got) != len(tt.want) {
				t.Errorf("missing = %v, want %v", got, tt.want)
			}
			for k, v := range tt.want {
				if got[k] != v {
					t.Errorf("%s reason = %q, want %q", k, got[k], v)
				}
			}
		})
	}
}

func TestGetScoresDataLegacyStoryIsNotBlocked(t *testing.T) {
	// A pre-five-phase story: no identify/produce/recall content, translation
	// done. Legacy vocab/grammar never block; the page renders with their score.
	h := NewHandler(slog.New(slog.DiscardHandler), nil)
	stubScoreDB(t,
		[]any{int32(0), int32(0), true, int32(0), int32(0), int32(0), int32(0)},
		[]any{int32(3), int32(1), int32(2), int32(0), int32(0), int32(0), int32(0), int32(0), int32(0), int32(0), float64(0)},
	)
	rr := assertQueryBudget(t, 13, h.GetScoresData, scoreRequest())
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rr.Code, rr.Body.String())
	}
	var resp struct {
		Data ScoreData `json:"data"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.Data.VocabCorrectCount != 3 || resp.Data.GrammarCorrectCount != 2 {
		t.Errorf("legacy counts = %d/%d, want 3/2", resp.Data.VocabCorrectCount, resp.Data.GrammarCorrectCount)
	}
	if resp.Data.IdentifyTotal != 0 || resp.Data.RecallTotal != 0 || resp.Data.ProduceTotal != 0 {
		t.Errorf("legacy story should report zero five-phase totals: %+v", resp.Data)
	}
}
