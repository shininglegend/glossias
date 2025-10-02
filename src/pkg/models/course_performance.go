package models

import (
	"context"
	"encoding/json"
	"glossias/src/pkg/generated/db"

	"github.com/jackc/pgx/v5/pgtype"
)

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

// GetCourseStudentPerformance retrieves performance data for all students in a course
func GetCourseStudentPerformance(ctx context.Context, courseID int32) ([]CourseStudentPerformance, error) {
	// Get all users in the course
	courseUsers, err := queries.GetUsersForCourse(ctx, courseID)
	if err != nil {
		return nil, err
	}

	// Get all stories in the course
	courseStories, err := queries.GetCourseStoriesWithTitles(ctx, db.GetCourseStoriesWithTitlesParams{
		CourseID:     pgtype.Int4{Int32: courseID, Valid: true},
		LanguageCode: "en",
	})
	if err != nil {
		return nil, err
	}

	var results []CourseStudentPerformance

	// For each user and story combination, get their performance data
	for _, user := range courseUsers {
		for _, story := range courseStories {
			performance := CourseStudentPerformance{
				UserID:     user.UserID,
				UserName:   user.Name,
				Email:      user.Email,
				StoryID:    story.StoryID,
				StoryTitle: story.Title,
			}

			// Get vocab accuracy
			vocabData, err := queries.GetUserStoryVocabSummary(ctx, db.GetUserStoryVocabSummaryParams{
				UserID:  user.UserID,
				StoryID: story.StoryID,
			})
			if err != nil {
				return nil, err
			}
			performance.VocabCorrect = vocabData.CorrectCount
			performance.VocabIncorrect = vocabData.IncorrectCount
			if vocabData.CorrectCount+vocabData.IncorrectCount > 0 {
				performance.VocabAccuracy = float64(vocabData.CorrectCount) / float64(vocabData.CorrectCount+vocabData.IncorrectCount) * 100
			}

			// Get grammar accuracy
			grammarData, err := queries.GetUserStoryGrammarSummary(ctx, db.GetUserStoryGrammarSummaryParams{
				UserID:  user.UserID,
				StoryID: story.StoryID,
			})
			if err != nil {
				return nil, err
			}
			performance.GrammarCorrect = grammarData.CorrectCount
			performance.GrammarIncorrect = grammarData.IncorrectCount
			if grammarData.CorrectCount+grammarData.IncorrectCount > 0 {
				performance.GrammarAccuracy = float64(grammarData.CorrectCount) / float64(grammarData.CorrectCount+grammarData.IncorrectCount) * 100
			}

			// Get translation status
			translationData, err := queries.GetUserTranslationStatusForStory(ctx, db.GetUserTranslationStatusForStoryParams{
				UserID:  user.UserID,
				StoryID: story.StoryID,
			})
			if err != nil {
				return nil, err
			}
			performance.TranslationCompleted = translationData.Completed

			// Parse requested_lines from PostgreSQL array
			if translationData.RequestedLines != nil {
				switch v := translationData.RequestedLines.(type) {
				case []int32:
					performance.RequestedLines = v
				case []int64:
					// Convert []int64 to []int32
					performance.RequestedLines = make([]int32, len(v))
					for i, val := range v {
						performance.RequestedLines[i] = int32(val)
					}
				case []interface{}:
					// Handle []interface{} where each element is an int
					performance.RequestedLines = make([]int32, 0, len(v))
					for _, item := range v {
						switch val := item.(type) {
						case int32:
							performance.RequestedLines = append(performance.RequestedLines, val)
						case int64:
							performance.RequestedLines = append(performance.RequestedLines, int32(val))
						case int:
							performance.RequestedLines = append(performance.RequestedLines, int32(val))
						}
					}
				case []byte:
					// Fallback: parse as JSON if it comes as bytes
					var lines []int32
					if err := json.Unmarshal(v, &lines); err == nil {
						performance.RequestedLines = lines
					}
				}
			}

			// Get time tracking data
			timeData, err := queries.GetUserStoryTimeTracking(ctx, db.GetUserStoryTimeTrackingParams{
				UserID:  user.UserID,
				StoryID: pgtype.Int4{Int32: story.StoryID, Valid: true},
			})
			if err != nil {
				return nil, err
			}

			// Convert interface{} to int32 with type assertions (handles both int32 and int64)
			if v, ok := timeData.VocabTimeSeconds.(int64); ok {
				performance.VocabTimeSeconds = int32(v)
			} else if v, ok := timeData.VocabTimeSeconds.(int32); ok {
				performance.VocabTimeSeconds = v
			}
			if v, ok := timeData.GrammarTimeSeconds.(int64); ok {
				performance.GrammarTimeSeconds = int32(v)
			} else if v, ok := timeData.GrammarTimeSeconds.(int32); ok {
				performance.GrammarTimeSeconds = v
			}
			if v, ok := timeData.TranslationTimeSeconds.(int64); ok {
				performance.TranslationTimeSeconds = int32(v)
			} else if v, ok := timeData.TranslationTimeSeconds.(int32); ok {
				performance.TranslationTimeSeconds = v
			}
			if v, ok := timeData.VideoTimeSeconds.(int64); ok {
				performance.VideoTimeSeconds = int32(v)
			} else if v, ok := timeData.VideoTimeSeconds.(int32); ok {
				performance.VideoTimeSeconds = v
			}

			// Calculate total time from components
			performance.TotalTimeSeconds = performance.VocabTimeSeconds +
				performance.GrammarTimeSeconds +
				performance.TranslationTimeSeconds +
				performance.VideoTimeSeconds

			results = append(results, performance)
		}
	}

	return results, nil
}
