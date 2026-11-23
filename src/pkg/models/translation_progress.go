package models

import (
	"context"

	"glossias/src/pkg/generated/db"
)

// SaveTranslationProgress merges lineNumbers (1-based) into the user's
// translation request for the story, creating it on the first call, and marks
// the request complete when complete is set. It is one statement, so two
// concurrent first calls cannot collide on the (user, story) unique key.
func SaveTranslationProgress(ctx context.Context, userID string, storyID int, lineNumbers []int, complete bool) error {
	lines := make([]int32, len(lineNumbers))
	for i, n := range lineNumbers {
		lines[i] = int32(n)
	}
	return queries.MergeTranslationRequestLines(ctx, db.MergeTranslationRequestLinesParams{
		UserID:   userID,
		StoryID:  int32(storyID),
		Lines:    lines,
		Complete: complete,
	})
}
