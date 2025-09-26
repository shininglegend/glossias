package models

import (
	"context"
	"glossias/src/pkg/generated/db"

	"github.com/jackc/pgx/v5/pgtype"
)

// SaveVocabScore saves a vocabulary score for a user
func SaveVocabScore(ctx context.Context, userID string, storyID int, lineNumber int, correct bool) error {
	// Get all vocabulary items for this line to save individual scores
	vocabItems, err := queries.GetVocabularyItems(ctx, db.GetVocabularyItemsParams{
		StoryID:    pgtype.Int4{Int32: int32(storyID), Valid: true},
		LineNumber: pgtype.Int4{Int32: int32(lineNumber), Valid: true},
	})
	if err != nil {
		return err
	}

	// Save score for each vocabulary item on this line
	for _, item := range vocabItems {
		err := queries.SaveVocabScore(ctx, db.SaveVocabScoreParams{
			UserID:      userID,
			StoryID:     int32(storyID),
			LineNumber:  int32(lineNumber),
			VocabItemID: item.ID,
			Correct:     correct,
		})
		if err != nil {
			return err
		}
	}

	return nil
}

// SaveGrammarScore saves a grammar score for a user (single line)
func SaveGrammarScore(ctx context.Context, userID string, storyID int, lineNumber int, correct bool) error {
	// Get all grammar items for this line
	grammarItems, err := queries.GetGrammarItems(ctx, db.GetGrammarItemsParams{
		StoryID:    pgtype.Int4{Int32: int32(storyID), Valid: true},
		LineNumber: pgtype.Int4{Int32: int32(lineNumber), Valid: true},
	})
	if err != nil {
		return err
	}

	// Save score for each grammar item on this line
	for _, item := range grammarItems {
		err := queries.SaveGrammarScore(ctx, db.SaveGrammarScoreParams{
			UserID:        userID,
			StoryID:       int32(storyID),
			LineNumber:    int32(lineNumber),
			GrammarItemID: item.ID,
			Correct:       correct,
		})
		if err != nil {
			return err
		}
	}

	return nil
}

// SaveGrammarScoresForPoint saves grammar scores for multiple lines of the same grammar point
func SaveGrammarScoresForPoint(ctx context.Context, userID string, storyID int, grammarPointID int, lineScores map[int]bool) error {
	// Use transaction to save all grammar scores atomically
	return withTransaction(func() error {
		for lineNumber, correct := range lineScores {
			// Get grammar items for this line that match the grammar point
			grammarItems, err := queries.GetGrammarItems(ctx, db.GetGrammarItemsParams{
				StoryID:    pgtype.Int4{Int32: int32(storyID), Valid: true},
				LineNumber: pgtype.Int4{Int32: int32(lineNumber), Valid: true},
			})
			if err != nil {
				return err
			}

			// Save score only for items matching the grammar point
			for _, item := range grammarItems {
				if item.GrammarPointID.Valid && int(item.GrammarPointID.Int32) == grammarPointID {
					err := queries.SaveGrammarScore(ctx, db.SaveGrammarScoreParams{
						UserID:        userID,
						StoryID:       int32(storyID),
						LineNumber:    int32(lineNumber),
						GrammarItemID: item.ID,
						Correct:       correct,
					})
					if err != nil {
						return err
					}
				}
			}
		}
		return nil
	})
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
