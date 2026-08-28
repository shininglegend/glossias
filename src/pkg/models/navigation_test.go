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
	}}, nil)
	SetDB(mockDB)
	t.Cleanup(func() { SetDB(struct{}{}) })

	c, err := GetUserStoryPageCompletion(context.Background(), "user-1", 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
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

func TestPageCompletionEmptyStory(t *testing.T) {
	var c PageCompletion
	if !c.ProduceComplete() {
		t.Error("produce with no segments authored should count as complete")
	}
	if c.IdentifyComplete() || c.RecallComplete() || c.TranslateComplete() {
		t.Error("identify, recall and translate must not be skipped on an empty story")
	}
}
