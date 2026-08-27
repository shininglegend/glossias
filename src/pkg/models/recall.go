package models

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"glossias/src/pkg/generated/db"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

// RecallSentencesPerStory is the number of sentences a story must have for the
// Recall sequencing exercise.
const RecallSentencesPerStory = 5

// ErrInvalidRecallOrder is returned when a submitted ordering is not a
// permutation of the story's recall sentences.
var ErrInvalidRecallOrder = errors.New("submitted ordering does not match the story's recall sentences")

// RecallSentence is one card in the Recall sequencing exercise. SequenceOrder
// is the correct position and must be withheld from student responses.
//
// ImagePath/ImageBucket are the source of truth for the card's picture; ImageURL
// is a signed read URL filled in on demand by SignRecallSentenceURLs and is
// never persisted.
type RecallSentence struct {
	ID            int    `json:"id"`
	StoryID       int    `json:"storyId"`
	SequenceOrder int    `json:"sequenceOrder"`
	HebrewText    string `json:"hebrewText"`
	TargetVocabID *int   `json:"targetVocabId,omitempty"`
	ImagePath     string `json:"imagePath,omitempty"`
	ImageBucket   string `json:"imageBucket,omitempty"`
	ImageURL      string `json:"imageUrl,omitempty"`
}

// GetStoryRecallSentences returns a story's sentences in correct order.
func GetStoryRecallSentences(ctx context.Context, storyID int) ([]RecallSentence, error) {
	if queries == nil {
		return nil, errors.New("database not initialized")
	}

	rows, err := queries.GetStoryRecallSentences(ctx, int32(storyID))
	if err != nil {
		return nil, err
	}

	sentences := make([]RecallSentence, 0, len(rows))
	for _, row := range rows {
		sentence := RecallSentence{
			ID:            int(row.ID),
			StoryID:       int(row.StoryID),
			SequenceOrder: int(row.SequenceOrder),
			HebrewText:    row.HebrewText,
			ImagePath:     row.ImagePath.String,
			ImageBucket:   row.ImageBucket.String,
		}
		if row.TargetVocabID.Valid {
			targetID := int(row.TargetVocabID.Int32)
			sentence.TargetVocabID = &targetID
		}
		sentences = append(sentences, sentence)
	}

	return sentences, nil
}

// GetRecallSentence retrieves a single sentence by ID.
func GetRecallSentence(ctx context.Context, id int) (*RecallSentence, error) {
	if queries == nil {
		return nil, errors.New("database not initialized")
	}

	row, err := queries.GetRecallSentence(ctx, int32(id))
	if errors.Is(err, sql.ErrNoRows) || errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	sentence := &RecallSentence{
		ID:            int(row.ID),
		StoryID:       int(row.StoryID),
		SequenceOrder: int(row.SequenceOrder),
		HebrewText:    row.HebrewText,
		ImagePath:     row.ImagePath.String,
		ImageBucket:   row.ImageBucket.String,
	}
	if row.TargetVocabID.Valid {
		targetID := int(row.TargetVocabID.Int32)
		sentence.TargetVocabID = &targetID
	}

	return sentence, nil
}

// UpsertRecallSentence creates or replaces the sentence at a story's given
// sequence position.
func UpsertRecallSentence(ctx context.Context, sentence RecallSentence) (*RecallSentence, error) {
	if queries == nil {
		return nil, errors.New("database not initialized")
	}

	targetVocabID := pgtype.Int4{}
	if sentence.TargetVocabID != nil {
		targetVocabID = pgtype.Int4{Int32: int32(*sentence.TargetVocabID), Valid: true}
	}

	row, err := queries.UpsertRecallSentence(ctx, db.UpsertRecallSentenceParams{
		StoryID:       int32(sentence.StoryID),
		SequenceOrder: int32(sentence.SequenceOrder),
		HebrewText:    sentence.HebrewText,
		TargetVocabID: targetVocabID,
		ImagePath:     optionalText(sentence.ImagePath),
		ImageBucket:   optionalText(sentence.ImageBucket),
	})
	if err != nil {
		return nil, err
	}

	saved := &RecallSentence{
		ID:            int(row.ID),
		StoryID:       int(row.StoryID),
		SequenceOrder: int(row.SequenceOrder),
		HebrewText:    row.HebrewText,
		ImagePath:     row.ImagePath.String,
		ImageBucket:   row.ImageBucket.String,
	}
	if row.TargetVocabID.Valid {
		targetID := int(row.TargetVocabID.Int32)
		saved.TargetVocabID = &targetID
	}

	InvalidateStoryContentReadiness(saved.StoryID)
	return saved, nil
}

// DeleteRecallSentence removes a single sentence.
func DeleteRecallSentence(ctx context.Context, storyID, id int) error {
	if queries == nil {
		return errors.New("database not initialized")
	}
	if err := queries.DeleteRecallSentence(ctx, int32(id)); err != nil {
		return err
	}
	InvalidateStoryContentReadiness(storyID)
	return nil
}

// DeleteStoryRecallSentences removes every sentence for a story.
func DeleteStoryRecallSentences(ctx context.Context, storyID int) error {
	if queries == nil {
		return errors.New("database not initialized")
	}
	if err := queries.DeleteStoryRecallSentences(ctx, int32(storyID)); err != nil {
		return err
	}
	InvalidateStoryContentReadiness(storyID)
	return nil
}

// CountStoryRecallSentences returns how many sentences a story has.
func CountStoryRecallSentences(ctx context.Context, storyID int) (int, error) {
	if queries == nil {
		return 0, errors.New("database not initialized")
	}

	count, err := queries.CountStoryRecallSentences(ctx, int32(storyID))
	if err != nil {
		return 0, err
	}
	return int(count), nil
}

// SaveRecallAttempt scores one ordering attempt: orderedSentenceIDs[i] is the
// sentence the student placed at position i+1. It logs one answer row per
// sentence and returns per-position correctness in the submitted order.
//
// The ordering must be a permutation of the story's sentences; anything else
// returns ErrInvalidRecallOrder.
func SaveRecallAttempt(ctx context.Context, userID string, storyID int, orderedSentenceIDs []int) ([]bool, error) {
	if queries == nil {
		return nil, errors.New("database not initialized")
	}

	sentences, err := GetStoryRecallSentences(ctx, storyID)
	if err != nil {
		return nil, err
	}
	if len(sentences) == 0 {
		return nil, ErrNotFound
	}
	if len(orderedSentenceIDs) != len(sentences) {
		return nil, fmt.Errorf("%w: expected %d sentences, got %d", ErrInvalidRecallOrder, len(sentences), len(orderedSentenceIDs))
	}

	correctOrder := make(map[int]int, len(sentences))
	for _, sentence := range sentences {
		correctOrder[sentence.ID] = sentence.SequenceOrder
	}

	seen := make(map[int]bool, len(orderedSentenceIDs))
	for _, id := range orderedSentenceIDs {
		if _, ok := correctOrder[id]; !ok {
			return nil, fmt.Errorf("%w: sentence %d does not belong to story %d", ErrInvalidRecallOrder, id, storyID)
		}
		if seen[id] {
			return nil, fmt.Errorf("%w: sentence %d submitted more than once", ErrInvalidRecallOrder, id)
		}
		seen[id] = true
	}

	results := make([]bool, len(orderedSentenceIDs))
	for i, id := range orderedSentenceIDs {
		position := i + 1
		correct := correctOrder[id] == position
		results[i] = correct

		if correct {
			err = queries.SaveRecallCorrectAnswer(ctx, db.SaveRecallCorrectAnswerParams{
				UserID:           userID,
				StoryID:          int32(storyID),
				RecallSentenceID: int32(id),
			})
		} else {
			err = queries.SaveRecallIncorrectAnswer(ctx, db.SaveRecallIncorrectAnswerParams{
				UserID:           userID,
				StoryID:          int32(storyID),
				RecallSentenceID: int32(id),
				SelectedPosition: int32(position),
			})
		}
		if err != nil {
			return nil, err
		}
	}

	return results, nil
}

// GetUserStoryRecallSummary returns the user's correct/incorrect counts, in the
// shape CalculateScoreWithRetriesAllowed expects.
func GetUserStoryRecallSummary(ctx context.Context, userID string, storyID int) (AnswerSummary, error) {
	if queries == nil {
		return AnswerSummary{}, errors.New("database not initialized")
	}

	row, err := queries.GetUserStoryRecallSummary(ctx, db.GetUserStoryRecallSummaryParams{
		UserID:  userID,
		StoryID: int32(storyID),
	})
	if err != nil {
		return AnswerSummary{}, err
	}

	return AnswerSummary{
		CorrectCount:   row.CorrectCount,
		IncorrectCount: row.IncorrectCount,
	}, nil
}
