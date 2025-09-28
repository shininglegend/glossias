package models

import (
	"context"
	"glossias/src/pkg/generated/db"
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
