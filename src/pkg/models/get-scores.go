package models

import (
	"context"
	"glossias/src/pkg/generated/db"

	"github.com/jackc/pgx/v5/pgtype"
)

// GetUserGrammarScores retrieves grammar scores for a user and story
func GetUserGrammarScores(ctx context.Context, userID string, storyID int) (map[int]bool, error) {
	scores, err := queries.GetUserGrammarScores(ctx, db.GetUserGrammarScoresParams{
		UserID:  userID,
		StoryID: int32(storyID),
	})
	if err != nil {
		return nil, err
	}

	// Calculate overall accuracy from all attempts
	totalAttempts := len(scores)
	correctAttempts := 0
	for _, score := range scores {
		if score.Correct {
			correctAttempts++
		}
	}

	result := make(map[int]bool)
	if totalAttempts > 0 {
		// Use line 0 to represent overall accuracy for the story
		result[0] = float64(correctAttempts)/float64(totalAttempts) >= 0.5
	}

	return result, nil
}

// GetUserVocabScores retrieves vocabulary scores for a user and story
func GetUserVocabScores(ctx context.Context, userID string, storyID int) (map[int]bool, error) {
	scores, err := queries.GetUserVocabScores(ctx, db.GetUserVocabScoresParams{
		UserID:  userID,
		StoryID: int32(storyID),
	})
	if err != nil {
		return nil, err
	}

	// Calculate overall accuracy from all attempts
	totalAttempts := len(scores)
	correctAttempts := 0
	for _, score := range scores {
		if score.Correct {
			correctAttempts++
		}
	}

	result := make(map[int]bool)
	if totalAttempts > 0 {
		// Use line 0 to represent overall accuracy for the story
		result[0] = float64(correctAttempts)/float64(totalAttempts) >= 0.5
	}

	return result, nil
}

// UserStoryVocabSummary represents vocabulary summary data
type UserStoryVocabSummary struct {
	TotalAttempts    int64
	CorrectAnswers   int64
	IncorrectAnswers int64
}

// GetUserStoryVocabSummary retrieves vocabulary summary for a user and story
func GetUserStoryVocabSummary(ctx context.Context, userID string, storyID int32) (*UserStoryVocabSummary, error) {
	result, err := queries.GetUserStoryVocabSummary(ctx, db.GetUserStoryVocabSummaryParams{
		UserID:  userID,
		StoryID: storyID,
	})
	if err != nil {
		return nil, err
	}

	return &UserStoryVocabSummary{
		TotalAttempts:    result.TotalAttempts,
		CorrectAnswers:   result.CorrectAnswers,
		IncorrectAnswers: result.IncorrectAnswers,
	}, nil
}

// UserStoryGrammarSummary represents grammar summary data
type UserStoryGrammarSummary struct {
	TotalAttempts    int64
	CorrectAnswers   int64
	IncorrectAnswers int64
}

// GetUserStoryGrammarSummary retrieves grammar summary for a user and story
func GetUserStoryGrammarSummary(ctx context.Context, userID string, storyID int32) (*UserStoryGrammarSummary, error) {
	result, err := queries.GetUserStoryGrammarSummary(ctx, db.GetUserStoryGrammarSummaryParams{
		UserID:  userID,
		StoryID: storyID,
	})
	if err != nil {
		return nil, err
	}

	return &UserStoryGrammarSummary{
		TotalAttempts:    result.TotalAttempts,
		CorrectAnswers:   result.CorrectAnswers,
		IncorrectAnswers: result.IncorrectAnswers,
	}, nil
}

// UserStoryTimeTracking represents time tracking data
type UserStoryTimeTracking struct {
	VocabTimeSeconds       int
	GrammarTimeSeconds     int
	TranslationTimeSeconds int
	VideoTimeSeconds       int
}

// GetUserStoryTimeTracking retrieves time tracking summary for a user and story
func GetUserStoryTimeTracking(ctx context.Context, userID string, storyID int32) (*UserStoryTimeTracking, error) {
	result, err := queries.GetUserStoryTimeTracking(ctx, db.GetUserStoryTimeTrackingParams{
		UserID:  userID,
		StoryID: pgtype.Int4{Int32: storyID, Valid: true},
	})
	if err != nil {
		return nil, err
	}

	// Convert interface{} to int, handling potential type conversions
	convertToInt := func(v interface{}) int {
		if v == nil {
			return 0
		}
		switch val := v.(type) {
		case int:
			return val
		case int32:
			return int(val)
		case int64:
			return int(val)
		default:
			return 0
		}
	}

	return &UserStoryTimeTracking{
		VocabTimeSeconds:       convertToInt(result.VocabTimeSeconds),
		GrammarTimeSeconds:     convertToInt(result.GrammarTimeSeconds),
		TranslationTimeSeconds: convertToInt(result.TranslationTimeSeconds),
		VideoTimeSeconds:       convertToInt(result.VideoTimeSeconds),
	}, nil
}
