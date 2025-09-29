package models

import (
	"context"
	"glossias/src/pkg/generated/db"

	"github.com/jackc/pgx/v5/pgtype"
)

// GetUserGrammarScores retrieves grammar scores for a user and story
func GetUserGrammarScores(ctx context.Context, userID string, storyID int) (map[int]bool, error) {
	scores, err := queries.GetUserLatestGrammarScoresByLine(ctx, db.GetUserLatestGrammarScoresByLineParams{
		UserID:  userID,
		StoryID: int32(storyID),
	})
	if err != nil {
		return nil, err
	}

	result := make(map[int]bool)
	for _, score := range scores {
		lineNum := int(score.LineNumber)
		// If line already has a score recorded, only update if this one is more recent
		if _, exists := result[lineNum]; !exists || score.Correct {
			result[lineNum] = score.Correct
		}
	}

	return result, nil
}

// GetUserVocabScores retrieves vocabulary scores for a user and story
func GetUserVocabScores(ctx context.Context, userID string, storyID int) (map[int]bool, error) {
	scores, err := queries.GetUserLatestVocabScoresByLine(ctx, db.GetUserLatestVocabScoresByLineParams{
		UserID:  userID,
		StoryID: int32(storyID),
	})
	if err != nil {
		return nil, err
	}

	result := make(map[int]bool)
	for _, score := range scores {
		lineNum := int(score.LineNumber)
		// If line already has a score recorded, only update if this one is more recent
		if _, exists := result[lineNum]; !exists || score.Correct {
			result[lineNum] = score.Correct
		}
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
