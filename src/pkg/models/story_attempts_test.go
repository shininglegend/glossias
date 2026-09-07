package models

import (
	"encoding/json"
	"testing"

	"glossias/src/pkg/generated/db"
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
