package handlers

import (
	"glossias/src/apis/types"
	"glossias/src/pkg/models"
	"reflect"
	"testing"
)

func intPtr(n int) *int { return &n }

func TestProduceSlot(t *testing.T) {
	// Hebrew text so rune offsets differ from byte offsets.
	lines := []string{
		"הילד רואה את הכלב",
		"הכלב רץ אל הילד",
	}

	t.Run("authored line, reference found: exact range on that line", func(t *testing.T) {
		got := produceSlot(lines, models.ProduceSegment{ReferenceHebrew: "רץ אל", LineStart: intPtr(2), LineEnd: intPtr(2)})
		want := &types.ProduceSlot{LineIndex: 1, LineEnd: 1, Exact: true, Start: 5, End: 10}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("slot = %+v, want %+v", got, want)
		}
	})

	t.Run("authored line beats a match elsewhere", func(t *testing.T) {
		// "הכלב" appears in both lines; the authored line is 1.
		got := produceSlot(lines, models.ProduceSegment{ReferenceHebrew: "הכלב", LineStart: intPtr(1), LineEnd: intPtr(1)})
		want := &types.ProduceSlot{LineIndex: 0, LineEnd: 0, Exact: true, Start: 13, End: 17}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("slot = %+v, want %+v", got, want)
		}
	})

	t.Run("authored line, paraphrased reference: whole line marked", func(t *testing.T) {
		got := produceSlot(lines, models.ProduceSegment{ReferenceHebrew: "הכלב רץ לילד", LineStart: intPtr(2), LineEnd: intPtr(2)})
		want := &types.ProduceSlot{LineIndex: 1, LineEnd: 1, Exact: false}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("slot = %+v, want %+v", got, want)
		}
	})

	t.Run("authored range spanning two lines, reference found on the second: exact range on that line", func(t *testing.T) {
		got := produceSlot(lines, models.ProduceSegment{ReferenceHebrew: "רץ אל", LineStart: intPtr(1), LineEnd: intPtr(2)})
		want := &types.ProduceSlot{LineIndex: 1, LineEnd: 1, Exact: true, Start: 5, End: 10}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("slot = %+v, want %+v", got, want)
		}
	})

	t.Run("authored range, reference not found on any single line: whole range marked", func(t *testing.T) {
		got := produceSlot(lines, models.ProduceSegment{ReferenceHebrew: "הכלב רואה את הכלב רץ", LineStart: intPtr(1), LineEnd: intPtr(2)})
		want := &types.ProduceSlot{LineIndex: 0, LineEnd: 1, Exact: false}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("slot = %+v, want %+v", got, want)
		}
	})

	t.Run("authored range out of bounds or inverted: nil", func(t *testing.T) {
		if got := produceSlot(lines, models.ProduceSegment{ReferenceHebrew: "הכלב", LineStart: intPtr(2), LineEnd: intPtr(3)}); got != nil {
			t.Errorf("expected nil, got %+v", got)
		}
		if got := produceSlot(lines, models.ProduceSegment{ReferenceHebrew: "הכלב", LineStart: intPtr(0), LineEnd: intPtr(1)}); got != nil {
			t.Errorf("expected nil, got %+v", got)
		}
		if got := produceSlot(lines, models.ProduceSegment{ReferenceHebrew: "הכלב", LineStart: intPtr(2), LineEnd: intPtr(1)}); got != nil {
			t.Errorf("expected nil, got %+v", got)
		}
	})

	t.Run("no authored line: first verbatim match, surrounding whitespace ignored", func(t *testing.T) {
		got := produceSlot(lines, models.ProduceSegment{ReferenceHebrew: " הכלב רץ אל הילד "})
		want := &types.ProduceSlot{LineIndex: 1, LineEnd: 1, Exact: true, Start: 0, End: 15}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("slot = %+v, want %+v", got, want)
		}
	})

	t.Run("no authored line, absent or empty reference: nil", func(t *testing.T) {
		if got := produceSlot(lines, models.ProduceSegment{ReferenceHebrew: "שלום"}); got != nil {
			t.Errorf("expected nil, got %+v", got)
		}
		if got := produceSlot(lines, models.ProduceSegment{ReferenceHebrew: "  "}); got != nil {
			t.Errorf("expected nil for blank reference, got %+v", got)
		}
	})
}

func TestSecondsLeft(t *testing.T) {
	cases := map[int]int{0: 90, 30: 60, 90: 0, 500: 0}
	for elapsed, want := range cases {
		if got := secondsLeft(elapsed); got != want {
			t.Errorf("secondsLeft(%d) = %d, want %d", elapsed, got, want)
		}
	}
}

func TestProduceStartViews(t *testing.T) {
	segments := []models.ProduceSegment{
		{ID: 1, SegmentOrder: 1},
		{ID: 2, SegmentOrder: 2},
	}
	starts := []models.ProduceAttemptStart{
		{SegmentID: 2, ElapsedSeconds: 20},
		{SegmentID: 1, ElapsedSeconds: 1000},
		{SegmentID: 99, ElapsedSeconds: 0}, // deleted segment
	}

	t.Run("started segments in order, floored at zero", func(t *testing.T) {
		got := produceStartViews(segments, nil, starts)
		want := []types.ProduceAttemptStartView{
			{SegmentID: 1, SecondsLeft: 0},
			{SegmentID: 2, SecondsLeft: 70},
		}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("views = %+v, want %+v", got, want)
		}
	})

	t.Run("submitted segments are omitted", func(t *testing.T) {
		subs := []models.ProduceSubmission{{SegmentID: 1}}
		got := produceStartViews(segments, subs, starts)
		want := []types.ProduceAttemptStartView{{SegmentID: 2, SecondsLeft: 70}}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("views = %+v, want %+v", got, want)
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
