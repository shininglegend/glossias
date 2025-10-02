package models

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
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

// GetStoryStudentPerformance retrieves performance data for all students in a specific story
func GetStoryStudentPerformance(ctx context.Context, storyID int32) ([]CourseStudentPerformance, error) {
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

		results[i] = CourseStudentPerformance{
			UserID:                 row.UserID,
			UserName:               row.UserName,
			Email:                  row.Email,
			StoryID:                storyID,
			StoryTitle:             row.StoryTitle.String,
			VocabCorrect:           row.VocabCorrect,
			VocabIncorrect:         row.VocabIncorrect,
			VocabAccuracy:          float64(row.VocabAccuracy),
			GrammarCorrect:         row.GrammarCorrect,
			GrammarIncorrect:       row.GrammarIncorrect,
			GrammarAccuracy:        float64(row.GrammarAccuracy),
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
