package models

import (
	"math"
	"testing"
)

func approx(a, b float64) bool { return math.Abs(a-b) < 0.01 }

func TestComputePhaseScores(t *testing.T) {
	fivePhase := PhaseTotals{IdentifyTotal: 10, ProduceTotal: 2, RecallTotal: 5}

	t.Run("perfect run averages identify, recall and produce", func(t *testing.T) {
		s := UserStoryScoreSummary{IdentifyCorrect: 10, RecallCorrect: 5, ProduceSubmitted: 2, ProduceGraded: 2, ProduceAverageScore: 80}
		ps := ComputePhaseScores(s, fivePhase)
		if !approx(ps.IdentifyAccuracy, 100) || !approx(ps.RecallAccuracy, 100) || !approx(ps.ProduceScore, 80) {
			t.Fatalf("phase scores = %+v", ps)
		}
		if !approx(ps.Overall, (100+100+80)/3.0) {
			t.Errorf("overall = %v, want %v", ps.Overall, (100+100+80)/3.0)
		}
		if ps.RecallAttempts != 1 {
			t.Errorf("recall attempts = %d, want 1", ps.RecallAttempts)
		}
	})

	t.Run("ungraded produce is left out of the overall, not scored 0", func(t *testing.T) {
		s := UserStoryScoreSummary{IdentifyCorrect: 10, RecallCorrect: 5, ProduceSubmitted: 2}
		ps := ComputePhaseScores(s, fivePhase)
		if ps.ProduceScore != 0 {
			t.Errorf("produce score = %v, want 0 while ungraded", ps.ProduceScore)
		}
		if !approx(ps.Overall, 100) {
			t.Errorf("overall = %v, want 100 (produce pending excluded)", ps.Overall)
		}
	})

	t.Run("recall retries count as attempts and lower accuracy", func(t *testing.T) {
		// Two full orderings: first 2/5 right, second 5/5.
		s := UserStoryScoreSummary{IdentifyCorrect: 10, RecallCorrect: 7, RecallIncorrect: 3, ProduceGraded: 2, ProduceAverageScore: 50}
		ps := ComputePhaseScores(s, fivePhase)
		if ps.RecallAttempts != 2 {
			t.Errorf("recall attempts = %d, want 2", ps.RecallAttempts)
		}
		want := CalculateScoreWithRetriesAllowed(7, 3, 5)
		if !approx(ps.RecallAccuracy, want) {
			t.Errorf("recall accuracy = %v, want %v", ps.RecallAccuracy, want)
		}
	})

	t.Run("legacy story falls back to vocab+grammar", func(t *testing.T) {
		legacy := PhaseTotals{VocabTotal: 4, GrammarTotal: 2}
		s := UserStoryScoreSummary{VocabCorrect: 4, VocabIncorrect: 2, GrammarCorrect: 2}
		ps := ComputePhaseScores(s, legacy)
		want := CalculateScoreWithRetriesAllowed(6, 2, 6)
		if !approx(ps.Overall, want) {
			t.Errorf("overall = %v, want %v", ps.Overall, want)
		}
		if ps.IdentifyAccuracy != 0 || ps.RecallAccuracy != 0 || ps.RecallAttempts != 0 {
			t.Errorf("new-phase scores should be zero for a legacy story: %+v", ps)
		}
	})

	t.Run("story with no content at all scores zero", func(t *testing.T) {
		ps := ComputePhaseScores(UserStoryScoreSummary{}, PhaseTotals{})
		if ps.Overall != 0 {
			t.Errorf("overall = %v, want 0", ps.Overall)
		}
	})

	t.Run("mixed-generation story with only identify authored", func(t *testing.T) {
		mixed := PhaseTotals{VocabTotal: 8, IdentifyTotal: 6}
		s := UserStoryScoreSummary{VocabCorrect: 8, IdentifyCorrect: 6, IdentifyIncorrect: 2}
		ps := ComputePhaseScores(s, mixed)
		if !approx(ps.Overall, CalculateScoreWithRetriesAllowed(6, 2, 6)) {
			t.Errorf("overall should follow identify alone, got %v", ps.Overall)
		}
	})
}
