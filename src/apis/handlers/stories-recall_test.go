package handlers

import (
	"glossias/src/pkg/models"
	"reflect"
	"testing"
)

func TestShuffledRecallCards(t *testing.T) {
	sentences := []models.RecallSentence{
		{ID: 1, SequenceOrder: 1, HebrewText: "א", ImageURL: "img-1"},
		{ID: 2, SequenceOrder: 2, HebrewText: "ב", ImageURL: "img-2"},
		{ID: 3, SequenceOrder: 3, HebrewText: "ג"},
	}
	reverse := func(n int, swap func(i, j int)) {
		for i, j := 0, n-1; i < j; i, j = i+1, j-1 {
			swap(i, j)
		}
	}

	cards := shuffledRecallCards(sentences, reverse)

	if got := []int{cards[0].ID, cards[1].ID, cards[2].ID}; !reflect.DeepEqual(got, []int{3, 2, 1}) {
		t.Errorf("order = %v, want reversed", got)
	}
	if cards[2].ImageURL != "img-1" || cards[2].HebrewText != "א" {
		t.Errorf("card fields not carried over: %+v", cards[2])
	}
	// The payload type has no position field, so SequenceOrder cannot leak;
	// this guards against someone adding one later.
	if _, has := reflect.TypeOf(cards[0]).FieldByName("SequenceOrder"); has {
		t.Error("RecallCard must not expose SequenceOrder")
	}
}

func TestRecallAttempts(t *testing.T) {
	cases := []struct {
		name    string
		summary models.AnswerSummary
		count   int
		want    int
	}{
		{"no sentences", models.AnswerSummary{CorrectCount: 5}, 0, 0},
		{"no attempts", models.AnswerSummary{}, 5, 0},
		{"one attempt", models.AnswerSummary{CorrectCount: 3, IncorrectCount: 2}, 5, 1},
		{"three attempts", models.AnswerSummary{CorrectCount: 11, IncorrectCount: 4}, 5, 3},
	}
	for _, tc := range cases {
		if got := recallAttempts(tc.summary, tc.count); got != tc.want {
			t.Errorf("%s: got %d, want %d", tc.name, got, tc.want)
		}
	}
}

func TestRecallCompleted(t *testing.T) {
	sentences := []models.RecallSentence{{ID: 1}, {ID: 2}, {ID: 3}}

	if recallCompleted(nil, []int{1, 2, 3}) {
		t.Error("a story with no sentences is never complete")
	}
	if recallCompleted(sentences, nil) {
		t.Error("no correct answers should not be complete")
	}
	if recallCompleted(sentences, []int{1, 2, 2}) {
		t.Error("missing a sentence should not be complete")
	}
	// Correct placements can come from different attempts; stale IDs from a
	// sentence since deleted are ignored.
	if !recallCompleted(sentences, []int{3, 1, 99, 2, 1}) {
		t.Error("expected complete once every sentence has been placed correctly")
	}
}
