package models

import (
	"context"
	"glossias/src/pkg/generated/db"
)

// GetUserGrammarScores retrieves grammar scores for a user and story
func GetUserGrammarScores(ctx context.Context, userID string, storyID int) (map[int]bool, error) {
	summary, err := GetUserStoryGrammarSummary(ctx, userID, int32(storyID))
	if err != nil {
		return nil, err
	}

	result := make(map[int]bool)
	totalAttempted := summary.CorrectCount + summary.IncorrectCount
	if totalAttempted > 0 {
		// Use line 0 to represent overall accuracy for the story
		result[0] = float64(summary.CorrectCount)/float64(totalAttempted) >= 0.5
	}

	return result, nil
}

// GetUserVocabScores retrieves vocabulary scores for a user and story
func GetUserVocabScores(ctx context.Context, userID string, storyID int) (map[int]bool, error) {
	summary, err := GetUserStoryVocabSummary(ctx, userID, int32(storyID))
	if err != nil {
		return nil, err
	}

	result := make(map[int]bool)
	totalAttempted := summary.CorrectCount + summary.IncorrectCount
	if totalAttempted > 0 {
		// Use line 0 to represent overall accuracy for the story
		result[0] = float64(summary.CorrectCount)/float64(totalAttempted) >= 0.5
	}

	return result, nil
}

func CheckAllVocabCompleteForLine(ctx context.Context, userID string, storyID, lineNumber int) (bool, error) {
	lineNumber = lineNumber + 1 // Convert to 1-based index for DB
	return queries.CheckAllVocabCompleteForLineForUser(ctx, db.CheckAllVocabCompleteForLineForUserParams{
		UserID:     userID,
		StoryID:    convertToPGInt(storyID),
		LineNumber: convertToPGInt(lineNumber),
	})
}

// UserStoryVocabSummary represents vocabulary summary data
type UserStoryVocabSummary struct {
	CorrectCount   int64
	IncorrectCount int64
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
		CorrectCount:   result.CorrectCount,
		IncorrectCount: result.IncorrectCount,
	}, nil
}

// UserStoryGrammarSummary represents grammar summary data
type UserStoryGrammarSummary struct {
	CorrectCount   int64
	IncorrectCount int64
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
		CorrectCount:   result.CorrectCount,
		IncorrectCount: result.IncorrectCount,
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
		StoryID: convertToPGInt(storyID),
	})
	if err != nil {
		return nil, err
	}
	// fmt.Println("Time spent: ", result)

	// Convert any to int, handling potential type conversions
	convertToInt := func(v any) int {
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

// CountStoryVocabItems returns the total number of vocabulary items for a story (cached)
func CountStoryVocabItems(ctx context.Context, storyID int32) (int64, error) {
	if cacheInstance == nil || keyBuilder == nil {
		// No cache available, query directly
		return queries.CountStoryVocabItems(ctx, convertToPGInt(storyID))
	}

	cacheKey := keyBuilder.StoryVocabCount(int(storyID))

	var count int64
	err := cacheInstance.GetOrSetJSON(cacheKey, &count, func() (any, error) {
		return queries.CountStoryVocabItems(ctx, convertToPGInt(storyID))
	})

	return count, err
}
