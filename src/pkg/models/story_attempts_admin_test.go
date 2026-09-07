package models

import (
	"context"
	"errors"
	"testing"
	"time"

	"glossias/src/pkg/database"

	"github.com/jackc/pgx/v5/pgtype"
)

func TestLiveAttemptStagesListsOnlyPhasesWithData(t *testing.T) {
	if got := liveAttemptStages(&AttemptScoreSnapshot{}); len(got) != 0 {
		t.Fatalf("untouched attempt stages = %+v, want none", got)
	}

	got := liveAttemptStages(&AttemptScoreSnapshot{
		VideoTimeSeconds:         120,
		IdentifyCorrectCount:     11,
		IdentifyIncorrectCount:   1,
		IdentifyTimeSeconds:      60,
		RequestedLines:           []int32{1, 2}, // started, not finished
		ProduceSegmentsSubmitted: 1,
		ProduceTotal:             2,
		RecallTimeSeconds:        30, // time only, no ordering yet
	})
	want := []AttemptStage{
		{Phase: ResetVideo, Detail: "watched", Seconds: 120},
		{Phase: ResetIdentify, Detail: "11 correct / 1 incorrect", Seconds: 60},
		{Phase: ResetTranslate, Detail: "started"},
		{Phase: ResetProduce, Detail: "1/2 submitted"},
		{Phase: ResetRecall, Detail: "0 attempts", Seconds: 30},
	}
	if len(got) != len(want) {
		t.Fatalf("stages = %+v, want %+v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("stage %d = %+v, want %+v", i, got[i], want[i])
		}
	}
}

func TestListUserStoryAttemptSummariesArchivedThenLive(t *testing.T) {
	mockDB := database.NewMockDBTX()
	when := time.Date(2026, 9, 1, 12, 0, 0, 0, time.UTC)
	mockDB.StubQuery("ListUserStoryAttemptSnapshots", [][]any{
		{int64(10), int32(1), pgtype.Timestamptz{Time: when, Valid: true}, []byte(`{"story_title":"S","overall_accuracy":63.1,"produce_segments":[]}`)},
		{int64(11), int32(2), pgtype.Timestamptz{Time: when.Add(time.Hour), Valid: true}, []byte(`{"story_title":"S","overall_accuracy":84.7}`)},
	}, nil)
	mockDB.StubQuery("GetUserStoryScoreSummary", [][]any{{
		int32(0), int32(0), int32(0), int32(0), int32(3), int32(1), int32(0), int32(0), int32(0), int32(0), float64(0), int32(0),
	}}, nil)
	mockDB.StubQuery("GetUserStoryTimeTracking", [][]any{{
		int64(0), int64(0), int64(0), int64(45), int64(90), int64(0), int64(0),
	}}, nil)
	mockDB.StubQuery("GetStoryPhaseTotals", [][]any{{int32(0), int32(0), int32(6), int32(2), int32(5)}}, nil)
	SetDB(mockDB)
	t.Cleanup(func() { SetDB(struct{}{}) })

	got, err := ListUserStoryAttemptSummaries(context.Background(), "u1", 2, "S")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 3 {
		t.Fatalf("summaries = %+v, want 2 archived + live", got)
	}
	if got[0].Number != 1 || got[0].IsCurrent || got[0].CompletedAt == nil || got[0].Score.OverallAccuracy != 63.1 || got[0].Stages != nil {
		t.Errorf("archived #1 = %+v", got[0])
	}
	if got[1].Number != 2 || got[1].Score.OverallAccuracy != 84.7 || len(got[1].Score.ProduceSegments) != 0 {
		t.Errorf("archived #2 = %+v (ProduceSegments must be non-nil)", got[1])
	}
	live := got[2]
	if live.Number != 3 || !live.IsCurrent || live.CompletedAt != nil || live.Score == nil {
		t.Fatalf("live = %+v", live)
	}
	if live.Score.IdentifyCorrectCount != 3 || live.Score.VideoTimeSeconds != 45 {
		t.Errorf("live score = %+v", live.Score)
	}
	if len(live.Stages) != 2 || live.Stages[0].Phase != ResetVideo || live.Stages[1].Phase != ResetIdentify || live.Stages[1].Seconds != 90 {
		t.Errorf("live stages = %+v, want video + identify", live.Stages)
	}
}

func TestDeleteArchivedAttemptRenumbersAscending(t *testing.T) {
	mockDB := database.NewMockDBTX()
	mockDB.StubExecRows("DELETE FROM story_attempts WHERE user_id = $1 AND story_id = $2 AND attempt_number = $3", 1)
	mockDB.StubQuery("ListUserStoryAttemptsAfter", [][]any{
		{int64(30), int32(3)},
		{int64(40), int32(4)},
	}, nil)
	SetDB(mockDB)
	t.Cleanup(func() { SetDB(struct{}{}) })

	if err := DeleteArchivedAttempt(context.Background(), "u1", 2, 2); err != nil {
		t.Fatal(err)
	}
	del := mockDB.Calls("DELETE FROM story_attempts")
	if len(del) != 1 || del[0].Args[2] != int32(2) {
		t.Fatalf("delete calls = %+v", del)
	}
	shifts := mockDB.Calls("UPDATE story_attempts SET attempt_number")
	if len(shifts) != 2 {
		t.Fatalf("renumber calls = %+v, want 2", shifts)
	}
	// Ascending: 3 -> 2 first (2 was just freed), then 4 -> 3.
	if shifts[0].Args[0] != int64(30) || shifts[0].Args[1] != int32(2) || shifts[1].Args[0] != int64(40) || shifts[1].Args[1] != int32(3) {
		t.Errorf("renumber args = %+v", shifts)
	}
}

func TestDeleteArchivedAttemptMissingIsNotFound(t *testing.T) {
	mockDB := database.NewMockDBTX()
	SetDB(mockDB)
	t.Cleanup(func() { SetDB(struct{}{}) })

	err := DeleteArchivedAttempt(context.Background(), "u1", 2, 9)
	if !errors.Is(err, ErrAttemptNotFound) {
		t.Fatalf("err = %v, want ErrAttemptNotFound", err)
	}
	if n := len(mockDB.Calls("ListUserStoryAttemptsAfter")); n != 0 {
		t.Errorf("renumber ran after a no-op delete (%d list calls)", n)
	}
}
