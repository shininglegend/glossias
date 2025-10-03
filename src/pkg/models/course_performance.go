package models

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"

	"github.com/jackc/pgx/v5/pgtype"
)

// GetStoryCourseID retrieves the course ID for a given story
func GetStoryCourseID(ctx context.Context, storyID int32) (int32, error) {
	story, err := queries.GetStory(ctx, storyID)
	if err != nil {
		return 0, err
	}
	return story.CourseID.Int32, nil
}

// CourseStudentPerformance represents performance data for one student in one story
type CourseStudentPerformance struct {
	UserID                 string  `json:"user_id"`
	UserName               string  `json:"user_name"`
	Email                  string  `json:"email"`
	StoryID                int32   `json:"story_id"`
	StoryTitle             string  `json:"story_title"`
	VocabCorrect           int64   `json:"vocab_correct"`
	VocabIncorrect         int64   `json:"vocab_incorrect"`
	VocabAccuracy          float64 `json:"vocab_accuracy"`
	GrammarCorrect         int64   `json:"grammar_correct"`
	GrammarIncorrect       int64   `json:"grammar_incorrect"`
	GrammarAccuracy        float64 `json:"grammar_accuracy"`
	TranslationCompleted   bool    `json:"translation_completed"`
	RequestedLines         []int32 `json:"requested_lines"`
	VocabTimeSeconds       int32   `json:"vocab_time_seconds"`
	GrammarTimeSeconds     int32   `json:"grammar_time_seconds"`
	TranslationTimeSeconds int32   `json:"translation_time_seconds"`
	VideoTimeSeconds       int32   `json:"video_time_seconds"`
	TotalTimeSeconds       int32   `json:"total_time_seconds"`
}

// convertToInt32 safely converts any to int32
func convertToInt32(v any) int32 {
	if v == nil {
		return 0
	}
	switch val := v.(type) {
	case int32:
		return val
	case int64:
		return int32(val)
	case int:
		return int32(val)
	default:
		fmt.Println("ERROR: Unknown type in convertToInt32:", fmt.Sprintf("%T", v))
		return 0
	}
}

// calculateAccuracy calculates accuracy score for vocab/grammar exercises where students must retry until correct.
// If not all items completed, score = (100 - (correctCount / totalItems * 100)) * (incorrectCount / (correctCount + incorrectCount))
// If all items completed, score = max(0, 100 - (incorrectCount / totalItems * 10))
func calculateAccuracy(correctCount, incorrectCount, totalItems int64) float64 {
	totalAttempted := correctCount + incorrectCount

	// If no attempts made, score is 0
	if totalAttempted == 0 {
		return 0
	}

	// If student hasn't completed all items, use partial completion formula
	if correctCount < totalItems {
		if incorrectCount == 0 {
			return float64(correctCount) / float64(totalItems) * 100
		}
		// Formula: (100 - (correctCount / totalItems * 100)) * (incorrectCount / totalAttempted)
		completionPenalty := 100 - (float64(correctCount)/float64(totalItems))*100
		errorRate := float64(incorrectCount) / float64(totalAttempted)
		return completionPenalty * errorRate
	}

	// If completed with no mistakes, perfect score
	if incorrectCount == 0 {
		return 100
	}

	// Calculate penalty: 10 points per mistake per item
	penalty := (float64(incorrectCount) / float64(totalItems)) * 10
	// Floor at 0
	accuracy := max(100-penalty, 0)

	return accuracy
}

// GetStoryStudentPerformance retrieves performance data for all students in a specific story
func GetStoryStudentPerformance(ctx context.Context, storyID int32) ([]CourseStudentPerformance, error) {
	// Get total vocab and grammar items for this story
	totalVocab, err := queries.CountStoryVocabItems(ctx, pgtype.Int4{Int32: storyID, Valid: true})
	if err != nil {
		return nil, err
	}
	totalGrammar, err := queries.CountStoryGrammarItems(ctx, pgtype.Int4{Int32: storyID, Valid: true})
	if err != nil {
		return nil, err
	}

	rows, err := queries.GetStoryStudentPerformance(ctx, storyID)
	if err != nil {
		return nil, err
	}

	results := make([]CourseStudentPerformance, len(rows))
	for i, row := range rows {
		// Parse requested_lines from PostgreSQL array
		var requestedLines []int32
		if row.RequestedLines != nil {
			switch v := row.RequestedLines.(type) {
			case []int32:
				requestedLines = v
			case []int64:
				requestedLines = make([]int32, len(v))
				for j, val := range v {
					requestedLines[j] = int32(val)
				}
			case []any:
				requestedLines = make([]int32, 0, len(v))
				for _, item := range v {
					switch val := item.(type) {
					case int32:
						requestedLines = append(requestedLines, val)
					case int64:
						requestedLines = append(requestedLines, int32(val))
					case int:
						requestedLines = append(requestedLines, int32(val))
					}
				}
			case []byte:
				var lines []int32
				if err := json.Unmarshal(v, &lines); err == nil {
					requestedLines = lines
				}
			}
			slices.Sort(requestedLines)
		}

		// Calculate vocab accuracy using the new formula
		vocabAccuracy := calculateAccuracy(row.VocabCorrect, row.VocabIncorrect, totalVocab)

		// Calculate grammar accuracy using the new formula
		grammarAccuracy := calculateAccuracy(row.GrammarCorrect, row.GrammarIncorrect, totalGrammar)

		results[i] = CourseStudentPerformance{
			UserID:                 row.UserID,
			UserName:               row.UserName,
			Email:                  row.Email,
			StoryID:                storyID,
			StoryTitle:             row.StoryTitle.String,
			VocabCorrect:           row.VocabCorrect,
			VocabIncorrect:         row.VocabIncorrect,
			VocabAccuracy:          vocabAccuracy,
			GrammarCorrect:         row.GrammarCorrect,
			GrammarIncorrect:       row.GrammarIncorrect,
			GrammarAccuracy:        grammarAccuracy,
			TranslationCompleted:   row.TranslationCompleted,
			RequestedLines:         requestedLines,
			VocabTimeSeconds:       convertToInt32(row.VocabTimeSeconds),
			GrammarTimeSeconds:     convertToInt32(row.GrammarTimeSeconds),
			TranslationTimeSeconds: convertToInt32(row.TranslationTimeSeconds),
			VideoTimeSeconds:       convertToInt32(row.VideoTimeSeconds),
			TotalTimeSeconds:       convertToInt32(row.TotalTimeSeconds),
		}
	}

	return results, nil
}
