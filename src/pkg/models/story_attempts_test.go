package models

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"glossias/src/pkg/database"
	"glossias/src/pkg/generated/db"

	"github.com/jackc/pgx/v5/pgtype"
)

func TestStoryExercisesComplete(t *testing.T) {
	if StoryExercisesComplete(nil) {
		t.Fatal("nil completion is not complete")
	}
	done := &PageCompletion{
		IdentifyTotal: 6, IdentifyCorrect: 6,
		TranslationCompleted: true,
		ProduceTotal:         2, ProduceSubmitted: 2,
		RecallTotal: 5, RecallCorrect: 5,
	}
	if !StoryExercisesComplete(done) {
		t.Fatal("expected complete")
	}
	done.ProduceSubmitted = 1
	if StoryExercisesComplete(done) {
		t.Fatal("incomplete produce must block")
	}
}

func TestReadyToArchiveWaitsForProduceGrading(t *testing.T) {
	done := &PageCompletion{
		IdentifyTotal: 6, IdentifyCorrect: 6,
		TranslationCompleted: true,
		ProduceTotal:         2, ProduceSubmitted: 2,
		RecallTotal: 5, RecallCorrect: 5,
	}
	if !ReadyToArchive(done, &UserStoryScoreSummary{ProduceSubmitted: 2, ProduceGraded: 2}) {
		t.Fatal("settled grades should archive")
	}
	if ReadyToArchive(done, &UserStoryScoreSummary{ProduceSubmitted: 2, ProducePending: 1}) {
		t.Fatal("a grade still in flight must hold the attempt open")
	}
	// Grading that failed (stamped, no score) is settled: nothing more will arrive.
	if !ReadyToArchive(done, &UserStoryScoreSummary{ProduceSubmitted: 2, ProduceGraded: 0, ProducePending: 0}) {
		t.Fatal("failed grades are settled and should archive")
	}
	done.RecallCorrect = 4
	if ReadyToArchive(done, &UserStoryScoreSummary{}) {
		t.Fatal("an unfinished story never archives")
	}
}

func TestListUserStoryAttemptsAppendsCurrent(t *testing.T) {
	mockDB := database.NewMockDBTX()
	SetDB(mockDB)
	t.Cleanup(func() { SetDB(struct{}{}) })

	// Never finished: the only attempt is the live one.
	got, err := ListUserStoryAttempts(context.Background(), "u1", 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Number != 1 || !got[0].IsCurrent || got[0].CompletedAt != nil {
		t.Fatalf("fresh student attempts = %+v", got)
	}

	when := time.Date(2026, 9, 1, 12, 0, 0, 0, time.UTC)
	mockDB.StubQuery("ListUserStoryCompletedAttempts", [][]any{
		{int32(1), pgtype.Timestamptz{Time: when, Valid: true}},
		{int32(2), pgtype.Timestamptz{Time: when.Add(time.Hour), Valid: true}},
	}, nil)
	got, err = ListUserStoryAttempts(context.Background(), "u1", 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 3 {
		t.Fatalf("attempts = %+v, want two archived + current", got)
	}
	for i, a := range got[:2] {
		if a.Number != i+1 || a.IsCurrent || a.CompletedAt == nil {
			t.Errorf("archived attempt %d = %+v", i+1, a)
		}
	}
	if cur := got[2]; cur.Number != 3 || !cur.IsCurrent || cur.CompletedAt != nil {
		t.Errorf("current attempt = %+v, want number 3 with no completion", cur)
	}
}

func TestArchiveCompletedAttemptBindsJSONStringAndWipesExercises(t *testing.T) {
	mockDB := database.NewMockDBTX()
	mockDB.StubQuery("GetUserStoryTimeTracking", [][]any{{
		int64(0), int64(0), int64(150), int64(105), int64(210), int64(300), int64(135),
	}}, nil)
	mockDB.StubQuery("GetStoryPhaseTotals", [][]any{{int32(0), int32(0), int32(6), int32(2), int32(5)}}, nil)
	mockDB.StubQuery("ArchiveStoryAttempt", [][]any{{int32(3)}}, nil)
	mockDB.StubQuery("ResetUserStoryAnswers", [][]any{{
		int64(0), int64(0), int64(0), int64(0), int64(1), int64(6), int64(2), int64(2), int64(2), int64(7), int64(3),
	}}, nil)
	SetDB(mockDB)
	t.Cleanup(func() { SetDB(struct{}{}) })

	summary := &UserStoryScoreSummary{
		IdentifyCorrect: 6, IdentifyIncorrect: 2,
		RecallCorrect: 7, RecallIncorrect: 3,
		ProduceSubmitted: 2, ProduceGraded: 2, ProduceAverageScore: 85,
	}
	ctx, _ := database.WithQueryCounter(context.Background())
	number, snap, err := ArchiveCompletedAttempt(ctx, "u1", 7, "Title", summary)
	if err != nil {
		t.Fatal(err)
	}
	if number != 3 || snap == nil || snap.StoryTitle != "Title" || snap.TotalTimeSeconds != 900 {
		t.Fatalf("archived = %d, %+v", number, snap)
	}
	// Five live-score loads + archive + two-statement wipe.
	if got := database.QueryCount(ctx); got > 8 {
		t.Errorf("archive made %d queries, want <= 8", got)
	}

	calls := mockDB.Calls("ArchiveStoryAttempt")
	if len(calls) != 1 {
		t.Fatalf("archive calls = %d", len(calls))
	}
	// The pool runs the simple protocol, which sends []byte as a bytea
	// literal that jsonb rejects (SQLSTATE 22P02); the snapshot must go as text.
	raw, ok := calls[0].Args[2].(string)
	if !ok {
		t.Fatalf("snapshot bound as %T, want string", calls[0].Args[2])
	}
	var decoded AttemptScoreSnapshot
	if err := json.Unmarshal([]byte(raw), &decoded); err != nil {
		t.Fatalf("snapshot is not JSON: %v", err)
	}
	if decoded.ProduceScore != 85 || decoded.IdentifyCorrectCount != 6 {
		t.Errorf("snapshot payload = %+v", decoded)
	}

	if len(mockDB.Calls("ResetUserStoryAnswers")) != 1 {
		t.Error("exercise answers were not wiped")
	}
	wipes := mockDB.Calls("DeleteUserStoryTimeTrackingByPhases")
	if len(wipes) != 1 {
		t.Fatalf("time wipes = %d, want one batched statement", len(wipes))
	}
	phases, _ := wipes[0].Args[2].([]string)
	if len(phases) != len(exerciseTimePhases) {
		t.Errorf("time phases wiped = %v", phases)
	}
	for _, p := range phases {
		if p == "video" {
			t.Error("video time must survive an archive")
		}
	}
	if len(mockDB.Calls("DELETE FROM story_attempts")) != 0 {
		t.Error("archiving must not delete earlier attempts")
	}
}

func TestApplyOfficialSnapshotOverwritesLiveRow(t *testing.T) {
	score := 91
	snap := AttemptScoreSnapshot{
		OverallAccuracy:          88,
		IdentifyAccuracy:         90,
		IdentifyCorrectCount:     6,
		IdentifyIncorrectCount:   1,
		ProduceScore:             80,
		ProduceSegmentsSubmitted: 2,
		ProduceSegmentsGraded:    2,
		RecallAccuracy:           85,
		RecallCorrectCount:       5,
		RecallIncorrectCount:     2,
		RecallAttempts:           2,
		TranslationCompleted:     true,
		RequestedLines:           []int32{1, 2},
		TotalTimeSeconds:         400,
		IdentifyTimeSeconds:      100,
		ProduceSegments: []AttemptProduceSegment{{
			SegmentOrder: 1, StudentText: "hi", AiScore: &score, AiFeedback: "Good.",
		}},
	}
	raw, err := json.Marshal(snap)
	if err != nil {
		t.Fatal(err)
	}
	p := CourseStudentPerformance{OverallAccuracy: 10, ProduceScore: 1}
	applyOfficialSnapshot(&p, raw)
	if p.OverallAccuracy != 88 || p.ProduceScore != 80 || p.IdentifyCorrect != 6 {
		t.Fatalf("snapshot not applied: %+v", p)
	}
	if p.TotalTimeSeconds != 400 || !p.TranslationCompleted {
		t.Fatalf("time/translate not applied: %+v", p)
	}
}

func TestHasLiveExerciseWork(t *testing.T) {
	empty := db.GetStoryStudentPerformanceRow{}
	if hasLiveExerciseWork(empty) {
		t.Fatal("empty row has no work")
	}
	if !hasLiveExerciseWork(db.GetStoryStudentPerformanceRow{TranslationCompleted: true}) {
		t.Fatal("translate completion counts")
	}
}
