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
	"time"

	"github.com/gorilla/mux"
	"github.com/jackc/pgx/v5/pgtype"
)

// stubScoreDB wires an uncoursed story 2 with the given page-completion row
// (GetUserStoryPageCompletion column order) and score-summary row
// (GetUserStoryScoreSummary column order). The archive queries are left
// unstubbed: ArchiveStoryAttempt then fails with ErrNoRows, so a test that
// expects an archive must call stubArchive.
func stubScoreDB(t *testing.T, completion, summary []any) *database.MockDBTX {
	t.Helper()
	mockDB := database.NewMockDBTX()
	mockDB.StubQuery("CanUserAccessStory", [][]any{{true}}, nil)
	mockDB.StubQuery("name: GetStory :one", [][]any{{
		int32(2), int32(1), "A", pgtype.Text{}, pgtype.Timestamp{}, "author", "Author", pgtype.Int4{},
	}}, nil)
	mockDB.StubQuery("GetUserStoryPageCompletion", [][]any{completion}, nil)
	mockDB.StubQuery("GetUserStoryScoreSummary", [][]any{summary}, nil)
	mockDB.StubQuery("GetUserStoryTimeTracking", [][]any{{
		int64(0), int64(0), int64(150), int64(105), int64(210), int64(300), int64(135),
	}}, nil)
	// GetStoryPhaseTotals column order: vocab, grammar, identify, produce,
	// recall — the same authored totals the completion row carries.
	mockDB.StubQuery("GetStoryPhaseTotals", [][]any{{int32(0), int32(0), completion[0], completion[5], completion[3]}}, nil)
	models.SetDB(mockDB)
	t.Cleanup(func() { models.SetDB(struct{}{}) })
	return mockDB
}

// stubArchive makes the archive succeed as attempt `number` and lists the
// archived attempts 1..number afterwards.
func stubArchive(mockDB *database.MockDBTX, number int) {
	mockDB.StubQuery("ArchiveStoryAttempt", [][]any{{int32(number)}}, nil)
	mockDB.StubQuery("ResetUserStoryAnswers", [][]any{{
		int64(0), int64(0), int64(0), int64(0), int64(1), int64(6), int64(2), int64(2), int64(2), int64(7), int64(3),
	}}, nil)
	stubCompletedAttempts(mockDB, number)
}

func stubCompletedAttempts(mockDB *database.MockDBTX, count int) {
	rows := make([][]any, 0, count)
	for i := range count {
		rows = append(rows, []any{int32(i + 1), pgtype.Timestamptz{Time: time.Date(2026, 9, 1+i, 0, 0, 0, 0, time.UTC), Valid: true}})
	}
	mockDB.StubQuery("ListUserStoryCompletedAttempts", rows, nil)
}

func scoreRequest(query string) *http.Request {
	req := httptest.NewRequest("GET", "/api/stories/2/scores"+query, nil)
	req = mux.SetURLVars(req, map[string]string{"id": "2"})
	return req.WithContext(context.WithValue(req.Context(), auth.UserIDKey, "user-1"))
}

func decodeScore(t *testing.T, rr *httptest.ResponseRecorder) ScoreData {
	t.Helper()
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rr.Code, rr.Body.String())
	}
	var resp struct {
		Data ScoreData `json:"data"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	return resp.Data
}

// Completion column order: identify total/correct, translate, recall
// total/correct, produce total/submitted, completed attempts.
// Summary column order: vocab c/i, grammar c/i, identify c/i, recall c/i,
// produce submitted/graded/avg/pending.
var (
	fullStoryDone = []any{int32(6), int32(6), true, int32(5), int32(5), int32(2), int32(2), int32(0)}
	freshStory    = []any{int32(6), int32(0), false, int32(5), int32(0), int32(2), int32(0), int32(0)}
	fullSummary   = []any{int32(0), int32(0), int32(0), int32(0), int32(6), int32(2), int32(7), int32(3), int32(2), int32(2), float64(85), int32(0)}
	emptySummary  = []any{int32(0), int32(0), int32(0), int32(0), int32(0), int32(0), int32(0), int32(0), int32(0), int32(0), float64(0), int32(0)}
)

// Budget for the archive path: 10 for the uncached GetStoryData load (cached
// in production) + completion + summary + the five live-score loads (time,
// totals, authored segments, submissions, translation) + the archive insert +
// the two-statement exercise wipe + the attempt list.
const archiveBudget = 23

func TestGetScoresDataArchivesOnCompletion(t *testing.T) {
	h := NewHandler(slog.New(slog.DiscardHandler), nil)
	mockDB := stubScoreDB(t, fullStoryDone, fullSummary)
	stubArchive(mockDB, 1)

	d := decodeScore(t, assertQueryBudget(t, archiveBudget, h.GetScoresData, scoreRequest("")))

	if !d.Archived || d.AttemptNumber != 1 || len(d.Attempts) != 1 || d.Attempts[0].Number != 1 || d.Attempts[0].Pending {
		t.Errorf("attempt metadata = archived %v, number %d, attempts %+v", d.Archived, d.AttemptNumber, d.Attempts)
	}
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

	// The snapshot must be bound as a JSON string (the simple protocol sends
	// []byte as bytea, which jsonb rejects) and carry the same numbers.
	archives := mockDB.Calls("ArchiveStoryAttempt")
	if len(archives) != 1 {
		t.Fatalf("archive calls = %d, want 1", len(archives))
	}
	payload, ok := archives[0].Args[2].(string)
	if !ok {
		t.Fatalf("snapshot bound as %T, want string", archives[0].Args[2])
	}
	var snap models.AttemptScoreSnapshot
	if err := json.Unmarshal([]byte(payload), &snap); err != nil {
		t.Fatalf("snapshot is not JSON: %v", err)
	}
	if snap.StoryTitle != d.StoryTitle || snap.OverallAccuracy != d.OverallAccuracy || len(snap.ProduceSegments) != 0 {
		t.Errorf("snapshot = %+v", snap)
	}
	// The wipe rides in the same request: answers and exercise time rows go,
	// video time stays.
	if n := len(mockDB.Calls("ResetUserStoryAnswers")); n != 1 {
		t.Errorf("answer wipes = %d, want 1", n)
	}
	wipes := mockDB.Calls("DeleteUserStoryTimeTrackingByPhases")
	if len(wipes) != 1 {
		t.Fatalf("time wipes = %d, want 1", len(wipes))
	}
	phases, _ := wipes[0].Args[2].([]string)
	for _, p := range phases {
		if p == "video" {
			t.Errorf("archive must keep video time: %v", phases)
		}
	}
	if len(mockDB.Calls("DELETE FROM story_attempts")) != 0 {
		t.Error("archive must not delete earlier attempts")
	}
}

func TestGetScoresDataHoldsLiveWhileProduceGradingPends(t *testing.T) {
	h := NewHandler(slog.New(slog.DiscardHandler), nil)
	pending := []any{int32(0), int32(0), int32(0), int32(0), int32(6), int32(2), int32(7), int32(3), int32(2), int32(0), float64(0), int32(2)}
	mockDB := stubScoreDB(t, fullStoryDone, pending)
	stubCompletedAttempts(mockDB, 1)

	d := decodeScore(t, assertQueryBudget(t, 20, h.GetScoresData, scoreRequest("")))

	if d.Archived {
		t.Error("an attempt awaiting grades must not be frozen yet")
	}
	if d.AttemptNumber != 2 || len(d.Attempts) != 2 || !d.Attempts[1].Pending || d.Attempts[0].Pending {
		t.Errorf("attempts = number %d, %+v", d.AttemptNumber, d.Attempts)
	}
	if d.ProduceSegmentsGraded != 0 || d.IdentifyCorrectCount != 6 {
		t.Errorf("live counts not served: %+v", d)
	}
	if n := len(mockDB.Calls("ArchiveStoryAttempt")); n != 0 {
		t.Errorf("archive calls = %d, want 0", n)
	}
	if n := len(mockDB.Calls("ResetUserStoryAnswers")); n != 0 {
		t.Errorf("answers wiped while grading pending: %d calls", n)
	}
}

func TestGetScoresDataServesArchivedAttempts(t *testing.T) {
	h := NewHandler(slog.New(slog.DiscardHandler), nil)
	mockDB := stubScoreDB(t, freshStory, emptySummary)
	stubCompletedAttempts(mockDB, 2)
	frozen, _ := json.Marshal(models.AttemptScoreSnapshot{
		StoryTitle: "A", OverallAccuracy: 77, IdentifyTotal: 6, IdentifyCorrectCount: 6,
		VocabCorrectCount: 3, // archived-section field: must not leak to the student
	})
	mockDB.StubQuery("GetUserStoryAttemptSnapshotByNumber", [][]any{{
		frozen, pgtype.Timestamptz{Valid: true}, int64(9), int32(2),
	}}, nil)

	// Budget: story load + completion + summary + attempt list + one snapshot.
	t.Run("defaults to the newest attempt", func(t *testing.T) {
		d := decodeScore(t, assertQueryBudget(t, 15, h.GetScoresData, scoreRequest("")))
		if d.AttemptNumber != 2 || !d.Archived || d.OverallAccuracy != 77 {
			t.Errorf("got attempt %d archived %v overall %v", d.AttemptNumber, d.Archived, d.OverallAccuracy)
		}
		if len(d.Attempts) != 2 || d.Attempts[0].CompletedAt == nil {
			t.Errorf("attempts = %+v", d.Attempts)
		}
	})
	t.Run("attempt query selects an earlier one", func(t *testing.T) {
		d := decodeScore(t, assertQueryBudget(t, 15, h.GetScoresData, scoreRequest("?attempt=1")))
		if d.AttemptNumber != 1 || !d.Archived {
			t.Errorf("got attempt %d archived %v", d.AttemptNumber, d.Archived)
		}
	})
	t.Run("archived section scores are omitted", func(t *testing.T) {
		rr := assertQueryBudget(t, 15, h.GetScoresData, scoreRequest(""))
		assertNoArchivedSectionScores(t, rr.Body.Bytes())
	})
	t.Run("the in-progress attempt has no score yet", func(t *testing.T) {
		rr := assertQueryBudget(t, 15, h.GetScoresData, scoreRequest("?attempt=3"))
		if rr.Code != http.StatusNotFound {
			t.Errorf("status = %d, want 404", rr.Code)
		}
	})
	t.Run("malformed attempt is rejected", func(t *testing.T) {
		rr := assertQueryBudget(t, 0, h.GetScoresData, scoreRequest("?attempt=first"))
		if rr.Code != http.StatusBadRequest {
			t.Errorf("status = %d, want 400", rr.Code)
		}
	})
	if n := len(mockDB.Calls("ArchiveStoryAttempt")); n != 0 {
		t.Errorf("a fresh run must not be archived: %d calls", n)
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
			completion: freshStory,
			summary:    emptySummary,
			want:       map[string]string{"identify": "no_data", "translation": "no_data", "produce": "no_data", "recall": "no_data"},
		},
		{
			name:       "started phases are incomplete, not no_data",
			completion: []any{int32(6), int32(4), true, int32(5), int32(3), int32(2), int32(1), int32(0)},
			summary:    []any{int32(0), int32(0), int32(0), int32(0), int32(4), int32(1), int32(3), int32(2), int32(1), int32(0), float64(0), int32(0)},
			want:       map[string]string{"identify": "incomplete", "produce": "incomplete", "recall": "incomplete"},
		},
		{
			name: "story with no produce or recall content only blocks on what it has",
			// Mixed-generation: identify authored, produce/recall not yet.
			completion: []any{int32(6), int32(6), false, int32(0), int32(0), int32(0), int32(0), int32(0)},
			summary:    []any{int32(0), int32(0), int32(0), int32(0), int32(6), int32(0), int32(0), int32(0), int32(0), int32(0), float64(0), int32(0)},
			want:       map[string]string{"translation": "no_data"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stubScoreDB(t, tt.completion, tt.summary)
			rr := assertQueryBudget(t, 14, h.GetScoresData, scoreRequest(""))
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
	// done. Archived vocab/grammar never block and their scores are omitted;
	// finishing Translate alone completes (and archives) the attempt.
	h := NewHandler(slog.New(slog.DiscardHandler), nil)
	mockDB := stubScoreDB(t,
		[]any{int32(0), int32(0), true, int32(0), int32(0), int32(0), int32(0), int32(0)},
		[]any{int32(3), int32(1), int32(2), int32(0), int32(0), int32(0), int32(0), int32(0), int32(0), int32(0), float64(0), int32(0)},
	)
	stubArchive(mockDB, 1)
	rr := assertQueryBudget(t, archiveBudget, h.GetScoresData, scoreRequest(""))
	d := decodeScore(t, rr)
	if d.IdentifyTotal != 0 || d.RecallTotal != 0 || d.ProduceTotal != 0 {
		t.Errorf("legacy story should report zero five-phase totals: %+v", d)
	}
	assertNoArchivedSectionScores(t, rr.Body.Bytes())
}

func TestGetScoresDataOmitsArchivedVocabAndGrammar(t *testing.T) {
	h := NewHandler(slog.New(slog.DiscardHandler), nil)
	mockDB := stubScoreDB(t, fullStoryDone, []any{
		int32(4), int32(1), int32(3), int32(2),
		int32(6), int32(2), int32(7), int32(3), int32(2), int32(2), float64(85), int32(0),
	})
	stubArchive(mockDB, 1)
	rr := assertQueryBudget(t, archiveBudget, h.GetScoresData, scoreRequest(""))
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rr.Code, rr.Body.String())
	}
	assertNoArchivedSectionScores(t, rr.Body.Bytes())
}

func assertNoArchivedSectionScores(t *testing.T, body []byte) {
	t.Helper()
	var raw map[string]any
	if err := json.Unmarshal(body, &raw); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	data, _ := raw["data"].(map[string]any)
	if data == nil {
		t.Fatal("missing data object")
	}
	for _, key := range []string{
		"vocab_accuracy", "vocab_correct_count", "vocab_incorrect_count",
		"grammar_accuracy", "grammar_correct_count", "grammar_incorrect_count",
	} {
		if _, ok := data[key]; ok {
			t.Errorf("archived section field %q should be omitted, got %v", key, data[key])
		}
	}
}
