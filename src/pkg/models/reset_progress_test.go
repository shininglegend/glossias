package models

import (
	"context"
	"errors"
	"testing"

	"glossias/src/pkg/database"
)

func TestResetUserStoryProgress_InvalidPhase(t *testing.T) {
	SetDB(database.NewMockDBTX())
	defer SetDB(struct{}{})

	_, err := ResetUserStoryProgress(context.Background(), "u1", 1, ResetPhase("nope"))
	if !errors.Is(err, ErrInvalidResetPhase) {
		t.Fatalf("expected ErrInvalidResetPhase, got %v", err)
	}
}

func TestResetUserStoryProgress_AllUsesBatchQuery(t *testing.T) {
	mockDB := database.NewMockDBTX()
	mockDB.StubQuery("ResetUserStoryAnswers", [][]any{{
		int64(1), int64(2), int64(3), int64(4), int64(1), int64(5), int64(6), int64(2), int64(2), int64(7), int64(8),
	}}, nil)
	SetDB(mockDB)
	defer SetDB(struct{}{})

	ctx, _ := database.WithQueryCounter(context.Background())
	res, err := ResetUserStoryProgress(ctx, "u1", 1, ResetAll)
	if err != nil {
		t.Fatal(err)
	}
	if got := database.QueryCount(ctx); got > 2 {
		t.Errorf("whole-story reset made %d queries, want <= 2 (batch CTE + time rows)", got)
	}
	if res.Deleted["recall_correct_answers"] != 7 || res.Deleted["identify_correct_answers"] != 5 {
		t.Errorf("counts not mapped from batch row: %v", res.Deleted)
	}
	if _, ok := res.Deleted["time_tracking"]; !ok {
		t.Errorf("expected time_tracking key: %v", res.Deleted)
	}
}

func TestResetUserStoryProgress_SinglePhaseTouchesOnlyItsTables(t *testing.T) {
	SetDB(database.NewMockDBTX())
	defer SetDB(struct{}{})

	res, err := ResetUserStoryProgress(context.Background(), "u1", 1, ResetProduce)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"produce_submissions", "produce_attempt_starts", "time_tracking"} {
		if _, ok := res.Deleted[want]; !ok {
			t.Errorf("missing %s in %v", want, res.Deleted)
		}
	}
	for _, notWant := range []string{"recall_correct_answers", "vocab_correct_answers", "translation_requests"} {
		if _, ok := res.Deleted[notWant]; ok {
			t.Errorf("produce reset must not touch %s: %v", notWant, res.Deleted)
		}
	}
}

func TestPhaseSpecsCoverEveryPhase(t *testing.T) {
	for _, p := range ResetPhases {
		if p == ResetAll {
			continue
		}
		spec, ok := phaseSpecs[p]
		if !ok {
			t.Errorf("phase %q has no spec", p)
			continue
		}
		if spec.timePhase == "" {
			t.Errorf("phase %q has no time-tracking phase", p)
		}
		if got := PhaseFromRoute("/stories/7/" + spec.timePhase); got != spec.timePhase {
			t.Errorf("PhaseFromRoute does not produce %q for its own route (got %q), so reset would miss the time rows", spec.timePhase, got)
		}
	}
}
