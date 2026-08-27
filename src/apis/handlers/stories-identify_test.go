package handlers

import (
	"glossias/src/pkg/models"
	"reflect"
	"testing"
)

func TestBuildIdentifyLines(t *testing.T) {
	// Hebrew text so rune offsets differ from byte offsets.
	lines := []models.StoryLine{
		{Text: "הילד רואה את הכלב"}, // no targets
		{Text: "הכלב רץ אל הילד"},   // two targets, one at the very end
		{Text: "כלב כלב"},           // same word twice on one line
	}
	occ := []models.TargetVocabularyOccurrence{
		{TargetVocabID: 7, LineNumber: 2, Position: [2]int{0, 4}},   // הכלב
		{TargetVocabID: 9, LineNumber: 2, Position: [2]int{11, 15}}, // הילד
		{TargetVocabID: 7, LineNumber: 3, Position: [2]int{4, 7}},   // second כלב (out of order on purpose)
		{TargetVocabID: 7, LineNumber: 3, Position: [2]int{0, 3}},   // first כלב
		{TargetVocabID: 9, LineNumber: 3, Position: [2]int{2, 5}},   // overlaps — must be dropped
		{TargetVocabID: 9, LineNumber: 1, Position: [2]int{50, 60}}, // out of range — must be dropped
	}

	got := buildIdentifyLines(lines, occ)

	if len(got) != 3 {
		t.Fatalf("expected 3 lines, got %d", len(got))
	}

	if want := "הילד רואה את הכלב"; len(got[0].Text) != 1 || got[0].Text[0].Type != "text" || got[0].Text[0].Text != want {
		t.Errorf("line 0: expected single text segment %q, got %+v", want, got[0].Text)
	}
	if len(got[0].TargetVocabIDs) != 0 {
		t.Errorf("line 0: expected no target IDs, got %v", got[0].TargetVocabIDs)
	}

	wantTypes := []string{"target", "text", "target"}
	if len(got[1].Text) != 3 {
		t.Fatalf("line 1: expected 3 segments, got %+v", got[1].Text)
	}
	for i, seg := range got[1].Text {
		if seg.Type != wantTypes[i] {
			t.Errorf("line 1 seg %d: type %q, want %q", i, seg.Type, wantTypes[i])
		}
	}
	if got[1].Text[0].Text != "הכלב" || got[1].Text[0].TargetVocabID != 7 {
		t.Errorf("line 1 seg 0: %+v", got[1].Text[0])
	}
	if got[1].Text[2].Text != "הילד" || got[1].Text[2].TargetVocabID != 9 {
		t.Errorf("line 1 seg 2: %+v", got[1].Text[2])
	}
	if !reflect.DeepEqual(got[1].TargetVocabIDs, []int{7, 9}) {
		t.Errorf("line 1: target IDs %v, want [7 9]", got[1].TargetVocabIDs)
	}

	// Reassembled text must equal the original for every line.
	for i, line := range got {
		var joined string
		for _, seg := range line.Text {
			joined += seg.Text
		}
		if joined != lines[i].Text {
			t.Errorf("line %d: reassembled %q != original %q", i, joined, lines[i].Text)
		}
	}

	if !reflect.DeepEqual(got[2].TargetVocabIDs, []int{7}) {
		t.Errorf("line 2: target IDs %v, want [7] (deduped)", got[2].TargetVocabIDs)
	}
	targetCount := 0
	for _, seg := range got[2].Text {
		if seg.Type == "target" {
			targetCount++
		}
	}
	if targetCount != 2 {
		t.Errorf("line 2: expected 2 target segments, got %d", targetCount)
	}
}
