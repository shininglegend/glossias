package models

import (
	"context"

	"glossias/src/pkg/generated/db"

	"github.com/jackc/pgx/v5/pgtype"
)

type VocabItem struct {
	LineNumber int
	Position   int
}

// GetLinesWithoutVocabForUser returns line and position pairs that have not been correctly completed by the user
func GetLinesWithoutVocabForUser(ctx context.Context, userID string, storyID int) ([]VocabItem, error) {
	rows, err := queries.GetIncompleteVocabForUser(ctx, db.GetIncompleteVocabForUserParams{
		StoryID: pgtype.Int4{Int32: int32(storyID), Valid: true},
		UserID:  userID,
	})
	if err != nil {
		return nil, err
	}

	vocabItems := make([]VocabItem, len(rows))
	for i, row := range rows {
		vocabItems[i] = VocabItem{
			LineNumber: int(row.LineNumber.Int32),
			Position:   int(row.PositionStart),
		}
	}

	return vocabItems, nil
}
