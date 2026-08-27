package handlers

import (
	"glossias/src/apis/types"
	"glossias/src/pkg/models"
	"reflect"
	"testing"
)

func TestFindProduceSlot(t *testing.T) {
	// Hebrew text so rune offsets differ from byte offsets.
	lines := []string{
		"הילד רואה את הכלב",
		"הכלב רץ אל הילד",
	}

	t.Run("finds the reference by rune offset", func(t *testing.T) {
		got := findProduceSlot(lines, "רץ אל")
		want := &types.ProduceSlot{LineIndex: 1, Start: 5, End: 10}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("slot = %+v, want %+v", got, want)
		}
	})

	t.Run("whole line, surrounding whitespace ignored", func(t *testing.T) {
		got := findProduceSlot(lines, " הכלב רץ אל הילד ")
		want := &types.ProduceSlot{LineIndex: 1, Start: 0, End: 15}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("slot = %+v, want %+v", got, want)
		}
	})

	t.Run("nil when absent or empty", func(t *testing.T) {
		if got := findProduceSlot(lines, "שלום"); got != nil {
			t.Errorf("expected nil, got %+v", got)
		}
		if got := findProduceSlot(lines, "  "); got != nil {
			t.Errorf("expected nil for blank reference, got %+v", got)
		}
	})
}

func TestProduceCompleted(t *testing.T) {
	segments := []models.ProduceSegment{
		{ID: 1, SegmentOrder: 1, ReferenceHebrew: "א"},
		{ID: 2, SegmentOrder: 2, ReferenceHebrew: "ב"},
	}

	if !produceCompleted(nil, nil) {
		t.Error("a story with no segments has nothing to do and counts as complete")
	}
	if produceCompleted(segments, nil) {
		t.Error("no submissions should not be complete")
	}
	one := []models.ProduceSubmission{{SegmentID: 2, StudentText: "x"}}
	if produceCompleted(segments, one) {
		t.Error("one of two segments should not be complete")
	}
	both := []models.ProduceSubmission{{SegmentID: 2, StudentText: "x"}, {SegmentID: 1, StudentText: ""}}
	if !produceCompleted(segments, both) {
		t.Error("expected complete once every segment has a submission (even an empty one)")
	}
	stale := []models.ProduceSubmission{{SegmentID: 1}, {SegmentID: 99}}
	if produceCompleted(segments, stale) {
		t.Error("a submission for a deleted segment must not count for a live one")
	}
}

func TestProduceSubmissionViews(t *testing.T) {
	segments := []models.ProduceSegment{
		{ID: 1, SegmentOrder: 1, ReferenceHebrew: "א"},
		{ID: 2, SegmentOrder: 2, ReferenceHebrew: "ב"},
	}
	subs := []models.ProduceSubmission{
		{SegmentID: 2, StudentText: "two"},
		{SegmentID: 99, StudentText: "stale"},
		{SegmentID: 1, StudentText: "one"},
	}
	got := produceSubmissionViews(segments, subs)
	want := []types.ProduceSubmissionView{
		{SegmentID: 1, StudentText: "one", ReferenceHebrew: "א"},
		{SegmentID: 2, StudentText: "two", ReferenceHebrew: "ב"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("views = %+v, want %+v", got, want)
	}
}
