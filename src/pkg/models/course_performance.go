package models

import (
	"context"
	"glossias/src/pkg/generated/db"
	"slices"
	"strings"
)

// GetStoryCourseID retrieves the course ID for a given story
func GetStoryCourseID(ctx context.Context, storyID int32) (int32, error) {
	story, err := queries.GetStory(ctx, storyID)
	if err != nil {
		return 0, err
	}
	return story.CourseID.Int32, nil
}

// CourseStudentPerformance is one student's performance on one story: the
// instructor-facing equivalent of the score page, on the same five-phase
// categories. Legacy vocab/grammar counts stay for stories authored before the
// five-phase flow.
type CourseStudentPerformance struct {
	UserID     string `json:"user_id"`
	UserName   string `json:"user_name"`
	Email      string `json:"email"`
	StoryID    int32  `json:"story_id"`
	StoryTitle string `json:"story_title"`

	OverallAccuracy float64 `json:"overall_accuracy"`

	IdentifyCorrect   int     `json:"identify_correct"`
	IdentifyIncorrect int     `json:"identify_incorrect"`
	IdentifyAccuracy  float64 `json:"identify_accuracy"`

	TranslationCompleted bool    `json:"translation_completed"`
	RequestedLines       []int32 `json:"requested_lines"`

	ProduceSubmitted int     `json:"produce_submitted"`
	ProduceTotal     int     `json:"produce_total"`
	ProduceGraded    int     `json:"produce_graded"`
	ProduceScore     float64 `json:"produce_score"`

	RecallCorrect   int     `json:"recall_correct"`
	RecallIncorrect int     `json:"recall_incorrect"`
	RecallAttempts  int     `json:"recall_attempts"`
	RecallAccuracy  float64 `json:"recall_accuracy"`

	VocabCorrect     int     `json:"vocab_correct"`
	VocabIncorrect   int     `json:"vocab_incorrect"`
	VocabAccuracy    float64 `json:"vocab_accuracy"`
	GrammarCorrect   int     `json:"grammar_correct"`
	GrammarIncorrect int     `json:"grammar_incorrect"`
	GrammarAccuracy  float64 `json:"grammar_accuracy"`

	VideoTimeSeconds       int32 `json:"video_time_seconds"`
	IdentifyTimeSeconds    int32 `json:"identify_time_seconds"`
	TranslationTimeSeconds int32 `json:"translation_time_seconds"`
	ProduceTimeSeconds     int32 `json:"produce_time_seconds"`
	RecallTimeSeconds      int32 `json:"recall_time_seconds"`
	VocabTimeSeconds       int32 `json:"vocab_time_seconds"`
	GrammarTimeSeconds     int32 `json:"grammar_time_seconds"`
	TotalTimeSeconds       int32 `json:"total_time_seconds"`
}

// CalculateScoreWithRetriesAllowed calculates a score for vocab/grammar exercises where students must retry until correct.
// If all items are complete, then final score is correct answers / (correct + incorrect answers).
// If only some items are incorrect (ie, total possible != correct), final score is (correct / (correct + incorrect)) * (correct / total)
// It takes the number of correct answers and incorrect answers by the student and the total number of possible answers for this story.
func CalculateScoreWithRetriesAllowed(correctCount, incorrectCount, totalPossible int64) float64 {
	// Convert to float64
	var correct, incorrect, possible float64 = float64(correctCount), float64(incorrectCount), float64(totalPossible)
	// total attempts
	totalAttempted := correct + (incorrect * .5) // incorrect answers count as half an attempt

	// If no attempts made or none correct, score is 0
	if totalAttempted == 0 || correct == 0 {
		return 0
	}

	// If total possible is 0, score is arbitrary 100
	if possible == 0 {
		return 100
	}

	// Calculation
	accuracy := (correct / totalAttempted) * (correct / possible) * 100

	// Floor at 0, cap at 100
	accuracy = min(max(accuracy, 0), 100)

	return accuracy
}

// GetStoryStudentPerformance retrieves performance data for all students in a
// specific story, best overall score first. Two round trips: the story's phase
// totals, then one aggregated row per student.
// status filters by course status: "active", "future", "past", or "" for all.
func GetStoryStudentPerformance(ctx context.Context, storyID int32, status string) ([]CourseStudentPerformance, error) {
	totals, err := GetStoryPhaseTotals(ctx, int(storyID))
	if err != nil {
		return nil, err
	}

	rows, err := queries.GetStoryStudentPerformance(ctx, db.GetStoryStudentPerformanceParams{
		StoryID: storyID,
		Status:  status,
	})
	if err != nil {
		return nil, err
	}

	results := make([]CourseStudentPerformance, len(rows))
	for i, row := range rows {
		summary := UserStoryScoreSummary{
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
		}
		scores := ComputePhaseScores(summary, *totals)

		requestedLines := slices.Clone(row.RequestedLines)
		slices.Sort(requestedLines)

		results[i] = CourseStudentPerformance{
			UserID:     row.UserID,
			UserName:   row.UserName,
			Email:      row.Email,
			StoryID:    storyID,
			StoryTitle: row.StoryTitle.String,

			OverallAccuracy: scores.Overall,

			IdentifyCorrect:   summary.IdentifyCorrect,
			IdentifyIncorrect: summary.IdentifyIncorrect,
			IdentifyAccuracy:  scores.IdentifyAccuracy,

			TranslationCompleted: row.TranslationCompleted,
			RequestedLines:       requestedLines,

			ProduceSubmitted: summary.ProduceSubmitted,
			ProduceTotal:     totals.ProduceTotal,
			ProduceGraded:    summary.ProduceGraded,
			ProduceScore:     scores.ProduceScore,

			RecallCorrect:   summary.RecallCorrect,
			RecallIncorrect: summary.RecallIncorrect,
			RecallAttempts:  scores.RecallAttempts,
			RecallAccuracy:  scores.RecallAccuracy,

			VocabCorrect:     summary.VocabCorrect,
			VocabIncorrect:   summary.VocabIncorrect,
			VocabAccuracy:    scores.VocabAccuracy,
			GrammarCorrect:   summary.GrammarCorrect,
			GrammarIncorrect: summary.GrammarIncorrect,
			GrammarAccuracy:  scores.GrammarAccuracy,

			VideoTimeSeconds:       row.VideoTimeSeconds,
			IdentifyTimeSeconds:    row.IdentifyTimeSeconds,
			TranslationTimeSeconds: row.TranslationTimeSeconds,
			ProduceTimeSeconds:     row.ProduceTimeSeconds,
			RecallTimeSeconds:      row.RecallTimeSeconds,
			VocabTimeSeconds:       row.VocabTimeSeconds,
			GrammarTimeSeconds:     row.GrammarTimeSeconds,
			TotalTimeSeconds:       row.TotalTimeSeconds,
		}
	}

	// Best overall first; ties broken by least time (faster is better when
	// scores match), then email for a stable order.
	slices.SortStableFunc(results, func(a, b CourseStudentPerformance) int {
		if a.OverallAccuracy != b.OverallAccuracy {
			if a.OverallAccuracy > b.OverallAccuracy {
				return -1
			}
			return 1
		}
		if a.TotalTimeSeconds != b.TotalTimeSeconds {
			return int(a.TotalTimeSeconds - b.TotalTimeSeconds)
		}
		return strings.Compare(a.Email, b.Email)
	})

	return results, nil
}
