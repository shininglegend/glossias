package models

import (
	"context"
	"testing"

	"glossias/src/pkg/database"
)

func TestGetUserStoryPageCompletion(t *testing.T) {
	mockDB := database.NewMockDBTX()
	mockDB.StubQuery("GetUserStoryPageCompletion", [][]any{{
		int32(6), int32(6), // identify total / correct
		true,               // translation completed
		int32(5), int32(3), // recall total / correct
		int32(0), int32(0), // produce total / submitted
		int32(2), // completed attempts
	}}, nil)
	SetDB(mockDB)
	t.Cleanup(func() { SetDB(struct{}{}) })

	c, err := GetUserStoryPageCompletion(context.Background(), "user-1", 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c.CompletedAttempts != 2 {
		t.Errorf("completed attempts = %d, want 2", c.CompletedAttempts)
	}

	checks := []struct {
		name string
		got  bool
		want bool
	}{
		{"identify", c.IdentifyComplete(), true},
		{"translate", c.TranslateComplete(), true},
		{"recall", c.RecallComplete(), false},
		{"produce (none authored)", c.ProduceComplete(), true},
	}
	for _, tc := range checks {
		if tc.got != tc.want {
			t.Errorf("%s complete = %v, want %v", tc.name, tc.got, tc.want)
		}
	}
}

func TestGetUserStoriesPageCompletion(t *testing.T) {
	mockDB := database.NewMockDBTX()
	mockDB.StubQuery("GetUserStoriesPageCompletion", [][]any{
		{int32(1), int32(6), int32(6), true, int32(5), int32(5), int32(2), int32(2), int32(0)},
		{int32(2), int32(4), int32(0), false, int32(5), int32(0), int32(0), int32(0), int32(1)},
	}, nil)
	SetDB(mockDB)
	t.Cleanup(func() { SetDB(struct{}{}) })

	got, err := GetUserStoriesPageCompletion(context.Background(), "user-1", []int{1, 2})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !got[1].FlowComplete() {
		t.Error("story 1 should be complete")
	}
	if got[2].LaterPhaseStarted() {
		t.Error("story 2 should not count as started")
	}
}

func TestPageCompletionEmptyStory(t *testing.T) {
	var c PageCompletion
	if !c.ProduceComplete() {
		t.Error("produce with no segments authored should count as complete")
	}
	if c.IdentifyComplete() || c.RecallComplete() || c.TranslateComplete() {
		t.Error("identify, recall and translate must not be skipped on an empty story")
	}
	if c.LaterPhaseStarted() || c.FlowComplete() {
		t.Error("empty story must not count as started or complete")
	}
}

func TestPageCompletionLaterPhaseStarted(t *testing.T) {
	zero := PageCompletion{}
	if zero.LaterPhaseStarted() {
		t.Error("zero value should not count as started")
	}
	authoredProduce := PageCompletion{ProduceTotal: 2}
	if authoredProduce.LaterPhaseStarted() {
		t.Error("authored produce with no submissions is not started")
	}
	cases := []PageCompletion{
		{IdentifyCorrect: 1},
		{TranslationCompleted: true},
		{ProduceSubmitted: 1},
		{RecallCorrect: 1},
	}
	for i := range cases {
		if !cases[i].LaterPhaseStarted() {
			t.Errorf("case %d should count as started", i)
		}
	}
}

func TestPageCompletionFlowComplete(t *testing.T) {
	done := PageCompletion{
		IdentifyTotal: 1, IdentifyCorrect: 1,
		TranslationCompleted: true,
		ProduceTotal:         1, ProduceSubmitted: 1,
		RecallTotal: 1, RecallCorrect: 1,
	}
	if !done.FlowComplete() {
		t.Error("all skippable phases done should be complete")
	}
	done.IdentifyCorrect = 0
	if done.FlowComplete() {
		t.Error("incomplete identify should not be complete")
	}
}
