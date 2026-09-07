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
	// CompletedAttempts is how many archived runs the student has; the live
	// rows above are always attempt CompletedAttempts+1.
	CompletedAttempts int
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

	return pageCompletionFromRow(row.IdentifyTotal, row.IdentifyCorrect, row.TranslationCompleted, row.RecallTotal, row.RecallCorrect, row.ProduceTotal, row.ProduceSubmitted, row.CompletedAttempts), nil
}

// GetUserStoriesPageCompletion loads phase progress for many stories in one
// round trip. Stories with no row (should not happen) are omitted.
func GetUserStoriesPageCompletion(ctx context.Context, userID string, storyIDs []int) (map[int]*PageCompletion, error) {
	if queries == nil {
		return nil, errors.New("database not initialized")
	}
	if len(storyIDs) == 0 {
		return map[int]*PageCompletion{}, nil
	}

	ids := make([]int32, len(storyIDs))
	for i, id := range storyIDs {
		ids[i] = int32(id)
	}

	rows, err := queries.GetUserStoriesPageCompletion(ctx, db.GetUserStoriesPageCompletionParams{
		UserID:   userID,
		StoryIds: ids,
	})
	if err != nil {
		return nil, err
	}

	out := make(map[int]*PageCompletion, len(rows))
	for _, row := range rows {
		out[int(row.StoryID)] = pageCompletionFromRow(row.IdentifyTotal, row.IdentifyCorrect, row.TranslationCompleted, row.RecallTotal, row.RecallCorrect, row.ProduceTotal, row.ProduceSubmitted, row.CompletedAttempts)
	}
	return out, nil
}

func pageCompletionFromRow(identifyTotal, identifyCorrect int32, translationCompleted bool, recallTotal, recallCorrect, produceTotal, produceSubmitted, completedAttempts int32) *PageCompletion {
	return &PageCompletion{
		IdentifyTotal:        int(identifyTotal),
		IdentifyCorrect:      int(identifyCorrect),
		TranslationCompleted: translationCompleted,
		RecallTotal:          int(recallTotal),
		RecallCorrect:        int(recallCorrect),
		ProduceTotal:         int(produceTotal),
		ProduceSubmitted:     int(produceSubmitted),
		CompletedAttempts:    int(completedAttempts),
	}
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

// LaterPhaseStarted is true once the student has stored progress past Watch.
// Empty Produce (submitted=0, total=0) does not count: ProduceComplete would
// be true on a story they have not actually started.
func (c *PageCompletion) LaterPhaseStarted() bool {
	return c.IdentifyCorrect > 0 || c.TranslationCompleted || c.ProduceSubmitted > 0 || c.RecallCorrect > 0
}

// FlowComplete is true when every skippable phase may be skipped, so list
// resume and the story-card CTA can send the student to Score.
func (c *PageCompletion) FlowComplete() bool {
	return c.IdentifyComplete() && c.TranslateComplete() && c.ProduceComplete() && c.RecallComplete()
}
