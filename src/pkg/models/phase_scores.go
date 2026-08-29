package models

import (
	"context"
	"errors"
	"glossias/src/pkg/generated/db"
)

// PhaseTotals is how many scorable items each phase of a story has: the
// denominators for CalculateScoreWithRetriesAllowed. IdentifyTotal counts
// target-word occurrences (one picture quiz each), matching PageCompletion.
type PhaseTotals struct {
	VocabTotal    int
	GrammarTotal  int
	IdentifyTotal int
	ProduceTotal  int
	RecallTotal   int
}

// GetStoryPhaseTotals loads a story's per-phase item counts in one round trip.
func GetStoryPhaseTotals(ctx context.Context, storyID int) (*PhaseTotals, error) {
	if queries == nil {
		return nil, errors.New("database not initialized")
	}
	row, err := queries.GetStoryPhaseTotals(ctx, int32(storyID))
	if err != nil {
		return nil, err
	}
	return &PhaseTotals{
		VocabTotal:    int(row.VocabTotal),
		GrammarTotal:  int(row.GrammarTotal),
		IdentifyTotal: int(row.IdentifyTotal),
		ProduceTotal:  int(row.ProduceTotal),
		RecallTotal:   int(row.RecallTotal),
	}, nil
}

// UserStoryScoreSummary is one student's raw answer counts for one story across
// every phase, legacy (vocab/grammar) and current (identify/produce/recall).
// Produce reflects the latest submission per segment; ProduceAverageScore is
// over graded segments only, so ProduceGraded tells "pending" from "scored 0".
type UserStoryScoreSummary struct {
	VocabCorrect        int
	VocabIncorrect      int
	GrammarCorrect      int
	GrammarIncorrect    int
	IdentifyCorrect     int
	IdentifyIncorrect   int
	RecallCorrect       int
	RecallIncorrect     int
	ProduceSubmitted    int
	ProduceGraded       int
	ProduceAverageScore float64
}

// GetUserStoryScoreSummary loads every answer count the score page needs in one
// round trip. It does not check story access; callers gate on GetStoryData.
func GetUserStoryScoreSummary(ctx context.Context, userID string, storyID int) (*UserStoryScoreSummary, error) {
	if queries == nil {
		return nil, errors.New("database not initialized")
	}
	row, err := queries.GetUserStoryScoreSummary(ctx, db.GetUserStoryScoreSummaryParams{
		UserID:  userID,
		StoryID: int32(storyID),
	})
	if err != nil {
		return nil, err
	}
	return &UserStoryScoreSummary{
		VocabCorrect:        int(row.VocabCorrect),
		VocabIncorrect:      int(row.VocabIncorrect),
		GrammarCorrect:      int(row.GrammarCorrect),
		GrammarIncorrect:    int(row.GrammarIncorrect),
		IdentifyCorrect:     int(row.IdentifyCorrect),
		IdentifyIncorrect:   int(row.IdentifyIncorrect),
		RecallCorrect:       int(row.RecallCorrect),
		RecallIncorrect:     int(row.RecallIncorrect),
		ProduceSubmitted:    int(row.ProduceSubmitted),
		ProduceGraded:       int(row.ProduceGraded),
		ProduceAverageScore: row.ProduceAverageScore,
	}, nil
}

// PhaseScores are the derived percentages (0–100) shown to students and
// instructors. The same rule produces the score page and the admin report so
// the two never disagree.
type PhaseScores struct {
	VocabAccuracy    float64
	GrammarAccuracy  float64
	IdentifyAccuracy float64
	RecallAccuracy   float64
	// RecallAttempts is how many full orderings the student submitted:
	// every attempt logs one row per sentence.
	RecallAttempts int
	// ProduceScore is the AI average over graded segments; meaningful only
	// when ProduceGraded > 0.
	ProduceScore float64
	// Overall averages the phases the story actually has (identify, recall,
	// and produce once graded). A story authored before the five-phase flow,
	// with none of those, falls back to the legacy vocab+grammar score.
	Overall float64
}

// ComputePhaseScores derives every percentage from raw counts and totals.
func ComputePhaseScores(s UserStoryScoreSummary, t PhaseTotals) PhaseScores {
	var ps PhaseScores
	if t.VocabTotal > 0 {
		ps.VocabAccuracy = CalculateScoreWithRetriesAllowed(int64(s.VocabCorrect), int64(s.VocabIncorrect), int64(t.VocabTotal))
	}
	if t.GrammarTotal > 0 {
		ps.GrammarAccuracy = CalculateScoreWithRetriesAllowed(int64(s.GrammarCorrect), int64(s.GrammarIncorrect), int64(t.GrammarTotal))
	}
	if t.IdentifyTotal > 0 {
		ps.IdentifyAccuracy = CalculateScoreWithRetriesAllowed(int64(s.IdentifyCorrect), int64(s.IdentifyIncorrect), int64(t.IdentifyTotal))
	}
	if t.RecallTotal > 0 {
		ps.RecallAccuracy = CalculateScoreWithRetriesAllowed(int64(s.RecallCorrect), int64(s.RecallIncorrect), int64(t.RecallTotal))
		ps.RecallAttempts = (s.RecallCorrect + s.RecallIncorrect) / t.RecallTotal
	}
	if s.ProduceGraded > 0 {
		ps.ProduceScore = s.ProduceAverageScore
	}

	var parts []float64
	if t.IdentifyTotal > 0 {
		parts = append(parts, ps.IdentifyAccuracy)
	}
	if t.RecallTotal > 0 {
		parts = append(parts, ps.RecallAccuracy)
	}
	if t.ProduceTotal > 0 && s.ProduceGraded > 0 {
		parts = append(parts, ps.ProduceScore)
	}
	switch {
	case len(parts) > 0:
		var sum float64
		for _, p := range parts {
			sum += p
		}
		ps.Overall = sum / float64(len(parts))
	case t.VocabTotal+t.GrammarTotal > 0:
		ps.Overall = CalculateScoreWithRetriesAllowed(
			int64(s.VocabCorrect+s.GrammarCorrect),
			int64(s.VocabIncorrect+s.GrammarIncorrect),
			int64(t.VocabTotal+t.GrammarTotal),
		)
	}
	return ps
}
