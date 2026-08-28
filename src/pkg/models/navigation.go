package models

import (
	"context"
	"errors"

	"glossias/src/pkg/generated/db"
)

// PageCompletion is a user's progress through every skippable phase of one
// story, fetched in a single query. Each *Complete method encodes the rule for
// whether navigation may skip that phase.
type PageCompletion struct {
	IdentifyTotal        int
	IdentifyCorrect      int
	TranslationCompleted bool
	RecallTotal          int
	RecallCorrect        int
	ProduceTotal         int
	ProduceSubmitted     int
}

// GetUserStoryPageCompletion loads the user's phase progress for a story in one
// round trip. It does not check story access; callers gate on GetStoryData.
func GetUserStoryPageCompletion(ctx context.Context, userID string, storyID int) (*PageCompletion, error) {
	if queries == nil {
		return nil, errors.New("database not initialized")
	}

	row, err := queries.GetUserStoryPageCompletion(ctx, db.GetUserStoryPageCompletionParams{
		StoryID: int32(storyID),
		UserID:  userID,
	})
	if err != nil {
		return nil, err
	}

	return &PageCompletion{
		IdentifyTotal:        int(row.IdentifyTotal),
		IdentifyCorrect:      int(row.IdentifyCorrect),
		TranslationCompleted: row.TranslationCompleted,
		RecallTotal:          int(row.RecallTotal),
		RecallCorrect:        int(row.RecallCorrect),
		ProduceTotal:         int(row.ProduceTotal),
		ProduceSubmitted:     int(row.ProduceSubmitted),
	}, nil
}

// IdentifyComplete: every target-word occurrence picked correctly. A story with
// no target vocabulary authored is never complete, so the student still visits
// the page (matching identifyProgress).
func (c *PageCompletion) IdentifyComplete() bool {
	return c.IdentifyTotal > 0 && c.IdentifyCorrect >= c.IdentifyTotal
}

// TranslateComplete: the student finished the translate phase for this story.
func (c *PageCompletion) TranslateComplete() bool {
	return c.TranslationCompleted
}

// RecallComplete: every recall sentence placed correctly at least once; a story
// with no sentences is never complete (matching recallCompleted).
func (c *PageCompletion) RecallComplete() bool {
	return c.RecallTotal > 0 && c.RecallCorrect >= c.RecallTotal
}

// ProduceComplete: every produce segment has a submission; no segments counts
// as complete (matching produceCompleted).
func (c *PageCompletion) ProduceComplete() bool {
	return c.ProduceSubmitted >= c.ProduceTotal
}
