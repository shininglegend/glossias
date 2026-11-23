package models

import (
	"context"
	"glossias/src/pkg/generated/db"

	"github.com/jackc/pgx/v5/pgtype"
)

// SaveVocabScore saves a vocabulary score for a user
func SaveVocabScore(ctx context.Context, userID string, storyID, lineNumber, position int, correct bool, incorrectAnswer string) error {
	lineNumber = lineNumber + 1 // Convert 0-indexed to 1-indexed
	// Get all vocabulary items for this line to save individual scores
	vocabItems, err := queries.GetVocabularyItems(ctx, db.GetVocabularyItemsParams{
		StoryID:    pgtype.Int4{Int32: int32(storyID), Valid: true},
		LineNumber: pgtype.Int4{Int32: int32(lineNumber), Valid: true},
	})
	if err != nil {
		return err
	}

	// Save score for each vocabulary item on this line
	if position < 0 || position >= len(vocabItems) {
		panic("Position out of bounds for vocab items on line")
	}

	item := vocabItems[position]
	// Save score only for the specific vocab item at this position
	if correct {
		err := queries.SaveVocabScore(ctx, db.SaveVocabScoreParams{
			UserID:      userID,
			StoryID:     int32(storyID),
			LineNumber:  int32(lineNumber),
			VocabItemID: item.ID,
		})
		if err != nil {
			return err
		}
	} else if incorrectAnswer != "" {
		err := queries.SaveVocabIncorrectAnswer(ctx, db.SaveVocabIncorrectAnswerParams{
			UserID:          userID,
			StoryID:         int32(storyID),
			LineNumber:      int32(lineNumber),
			VocabItemID:     item.ID,
			IncorrectAnswer: incorrectAnswer,
		})
		if err != nil {
			return err
		}
	}

	return nil
}

// GetUserGrammarScoresByGrammarPoint retrieves user's correct grammar answers for a specific grammar point
func GetUserGrammarScoresByGrammarPoint(ctx context.Context, userID string, storyID int, grammarPointID int) ([]db.GetUserGrammarScoresByGrammarPointRow, error) {
	return queries.GetUserGrammarScoresByGrammarPoint(ctx, db.GetUserGrammarScoresByGrammarPointParams{
		UserID:         userID,
		StoryID:        int32(storyID),
		GrammarPointID: int32(grammarPointID),
	})
}

// GetUserGrammarIncorrectAnswers retrieves user's incorrect grammar answers for a specific grammar point
func GetUserGrammarIncorrectAnswers(ctx context.Context, userID string, storyID int, grammarPointID int) ([]db.GetUserGrammarIncorrectAnswersRow, error) {
	return queries.GetUserGrammarIncorrectAnswers(ctx, db.GetUserGrammarIncorrectAnswersParams{
		UserID:         userID,
		StoryID:        int32(storyID),
		GrammarPointID: int32(grammarPointID),
	})
}

// SaveSingleGrammarSelection saves a single grammar selection (correct or incorrect)
func SaveSingleGrammarSelection(ctx context.Context, userID string, storyID int, grammarPointID int, lineNumber int, position int, correct bool) error {
	if correct {
		return queries.SaveGrammarScore(ctx, db.SaveGrammarScoreParams{
			UserID:         userID,
			StoryID:        int32(storyID),
			LineNumber:     int32(lineNumber),
			GrammarPointID: int32(grammarPointID),
		})
	}

	return queries.SaveGrammarIncorrectAnswer(ctx, db.SaveGrammarIncorrectAnswerParams{
		UserID:            userID,
		StoryID:           int32(storyID),
		LineNumber:        int32(lineNumber),
		GrammarPointID:    int32(grammarPointID),
		SelectedLine:      int32(lineNumber),
		SelectedPositions: []int32{int32(position)},
	})
}

// CountFoundGrammarInstances counts how many instances of a grammar point a user has already found correctly
func CountFoundGrammarInstances(ctx context.Context, userID string, storyID int, grammarPointID int) (int, error) {
	scores, err := queries.GetUserGrammarScores(ctx, db.GetUserGrammarScoresParams{
		UserID:  userID,
		StoryID: int32(storyID),
	})

	if err != nil {
		return 0, err
	}

	// Count scores matching this grammar point
	count := 0
	for _, score := range scores {
		if int(score.GrammarPointID) == grammarPointID {
			count++
		}
	}
	return count, nil
}
