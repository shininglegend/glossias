package models

import (
	"context"
	"glossias/src/pkg/generated/db"
	"slices"

	"github.com/jackc/pgx/v5/pgtype"
)

// SaveVocabScore saves a vocabulary score for a user
func SaveVocabScore(ctx context.Context, userID string, storyID int, lineNumber int, correct bool, incorrectAnswer string) error {
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
	for _, item := range vocabItems {
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
	}

	return nil
}

// SaveGrammarScore saves a grammar score for a user (single line)
func SaveGrammarScore(ctx context.Context, userID string, storyID int, lineNumber int, correct bool, selectedLine int, selectedPositions []int) error {
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
		if correct {
			err := queries.SaveGrammarScore(ctx, db.SaveGrammarScoreParams{
				UserID:         userID,
				StoryID:        int32(storyID),
				LineNumber:     int32(lineNumber),
				GrammarPointID: item.GrammarPointID.Int32,
			})
			if err != nil {
				return err
			}
		} else if selectedLine > 0 && len(selectedPositions) > 0 {
			positions := make([]int32, len(selectedPositions))
			for i, pos := range selectedPositions {
				positions[i] = int32(pos)
			}
			err := queries.SaveGrammarIncorrectAnswer(ctx, db.SaveGrammarIncorrectAnswerParams{
				UserID:            userID,
				StoryID:           int32(storyID),
				LineNumber:        int32(lineNumber),
				GrammarPointID:    item.GrammarPointID.Int32,
				SelectedLine:      int32(selectedLine),
				SelectedPositions: positions,
			})
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// SaveGrammarScoresForPoint saves grammar scores for multiple lines of the same grammar point
func SaveGrammarScoresForPoint(ctx context.Context, userID string, storyID int, grammarPointID int, lineScores map[int]bool, incorrectAnswers map[int]struct {
	SelectedLine      int
	SelectedPositions []int
}) error {
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
					if correct {
						err := queries.SaveGrammarScore(ctx, db.SaveGrammarScoreParams{
							UserID:         userID,
							StoryID:        int32(storyID),
							LineNumber:     int32(lineNumber),
							GrammarPointID: item.GrammarPointID.Int32,
						})
						if err != nil {
							return err
						}
					} else {
						if incorrectAnswer, exists := incorrectAnswers[lineNumber]; exists && incorrectAnswer.SelectedLine > 0 && len(incorrectAnswer.SelectedPositions) > 0 {
							positions := make([]int32, len(incorrectAnswer.SelectedPositions))
							for i, pos := range incorrectAnswer.SelectedPositions {
								positions[i] = int32(pos)
							}
							err := queries.SaveGrammarIncorrectAnswer(ctx, db.SaveGrammarIncorrectAnswerParams{
								UserID:            userID,
								StoryID:           int32(storyID),
								LineNumber:        int32(lineNumber),
								GrammarPointID:    item.GrammarPointID.Int32,
								SelectedLine:      int32(incorrectAnswer.SelectedLine),
								SelectedPositions: positions,
							})
							if err != nil {
								return err
							}
						}
					}
				}
			}
		}
		return nil
	})
}

// SaveCorrectGrammarItems saves correct grammar scores
func SaveCorrectGrammarItems(ctx context.Context, userID string, storyID, grammarPointID int, grammarItemsMap map[int][]GrammarItem, correctItems map[int][]int) error {
	return withTransaction(func() error {
		for lineNumber, itemIndices := range correctItems {
			grammarItems, err := queries.GetGrammarItems(ctx, db.GetGrammarItemsParams{
				StoryID:    pgtype.Int4{Int32: int32(storyID), Valid: true},
				LineNumber: pgtype.Int4{Int32: int32(lineNumber), Valid: true},
			})
			if err != nil {
				return err
			}

			// Save correct scores for matching items
			itemIndex := 0
			for _, dbItem := range grammarItems {
				if dbItem.GrammarPointID.Valid && int(dbItem.GrammarPointID.Int32) == grammarPointID {
					// Check if this item index is in the correct list
					if slices.Contains(itemIndices, itemIndex) {
						err := queries.SaveGrammarScore(ctx, db.SaveGrammarScoreParams{
							UserID:         userID,
							StoryID:        int32(storyID),
							LineNumber:     int32(lineNumber),
							GrammarPointID: dbItem.GrammarPointID.Int32,
						})
						if err != nil {
							return err
						}
					}
					itemIndex++
				}
			}
		}
		return nil
	})
}

// SaveCorrectAnswers saves correct answer data
func SaveCorrectAnswers(ctx context.Context, userID string, storyID, grammarPointID int, correctAnswers []struct {
	LineNumber int
	Position   [2]int
	Text       string
}) error {
	return withTransaction(func() error {
		for _, correctAnswer := range correctAnswers {
			err := queries.SaveGrammarScore(ctx, db.SaveGrammarScoreParams{
				UserID:         userID,
				StoryID:        int32(storyID),
				LineNumber:     int32(correctAnswer.LineNumber),
				GrammarPointID: int32(grammarPointID),
			})
			if err != nil {
				return err
			}
		}
		return nil
	})
}

// SaveIncorrectAnswers saves incorrect answer data for wrong user clicks
func SaveIncorrectAnswers(ctx context.Context, userID string, storyID, grammarPointID int, incorrectAnswers []struct {
	LineNumber int
	Position   int
}) error {
	return withTransaction(func() error {
		for _, incorrectAnswer := range incorrectAnswers {
			err := queries.SaveGrammarIncorrectAnswer(ctx, db.SaveGrammarIncorrectAnswerParams{
				UserID:            userID,
				StoryID:           int32(storyID),
				LineNumber:        int32(incorrectAnswer.LineNumber),
				GrammarPointID:    int32(grammarPointID),
				SelectedLine:      int32(incorrectAnswer.LineNumber),
				SelectedPositions: []int32{int32(incorrectAnswer.Position)},
			})
			if err != nil {
				return err
			}
		}
		return nil
	})
}
