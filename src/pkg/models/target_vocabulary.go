package models

import (
	"context"
	"database/sql"
	"errors"

	"glossias/src/pkg/generated/db"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

// TargetWordsPerStory is the number of target vocabulary words each story must
// have for the Identify and Recall phases.
const TargetWordsPerStory = 5

// MinTargetWordOccurrences is how many times a target word must appear in the
// story text. The Identify phase pauses on every occurrence, so a word seen
// only once gives the student no second look.
const MinTargetWordOccurrences = 2

// TargetVocabulary is one of a story's target words, with the pronunciation
// audio and picture used by the Identify phase.
//
// The path/bucket pairs are the source of truth for which asset belongs to this
// word; AudioURL and ImageURL are signed read URLs filled in on demand by
// SignTargetVocabularyURLs and are never persisted.
type TargetVocabulary struct {
	ID               int    `json:"id"`
	StoryID          int    `json:"storyId"`
	LexicalForm      string `json:"lexicalForm"`
	AudioPath        string `json:"audioPath,omitempty"`
	AudioBucket      string `json:"audioBucket,omitempty"`
	CorrectImagePath string `json:"correctImagePath,omitempty"`
	ImageBucket      string `json:"imageBucket,omitempty"`
	AudioURL         string `json:"audioUrl,omitempty"`
	ImageURL         string `json:"imageUrl,omitempty"`
}

// TargetVocabularyOccurrence is one appearance of a target word in the story
// text, matched to a vocabulary_items row by lexical form.
type TargetVocabularyOccurrence struct {
	TargetVocabID int    `json:"targetVocabId"`
	LexicalForm   string `json:"lexicalForm"`
	VocabItemID   int    `json:"vocabItemId"`
	LineNumber    int    `json:"lineNumber"`
	Word          string `json:"word"`
	Position      [2]int `json:"position"` // [start, end]
}

// GetStoryTargetVocabulary returns a story's target words, ordered by creation.
func GetStoryTargetVocabulary(ctx context.Context, storyID int) ([]TargetVocabulary, error) {
	if queries == nil {
		return nil, errors.New("database not initialized")
	}

	rows, err := queries.GetStoryTargetVocabulary(ctx, int32(storyID))
	if err != nil {
		return nil, err
	}

	words := make([]TargetVocabulary, 0, len(rows))
	for _, row := range rows {
		words = append(words, TargetVocabulary{
			ID:               int(row.ID),
			StoryID:          int(row.StoryID),
			LexicalForm:      row.LexicalForm,
			AudioPath:        row.AudioPath.String,
			AudioBucket:      row.AudioBucket.String,
			CorrectImagePath: row.CorrectImagePath.String,
			ImageBucket:      row.ImageBucket.String,
		})
	}

	return words, nil
}

// GetTargetVocabulary retrieves a single target word by ID.
func GetTargetVocabulary(ctx context.Context, id int) (*TargetVocabulary, error) {
	if queries == nil {
		return nil, errors.New("database not initialized")
	}

	row, err := queries.GetTargetVocabulary(ctx, int32(id))
	if errors.Is(err, sql.ErrNoRows) || errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	return &TargetVocabulary{
		ID:               int(row.ID),
		StoryID:          int(row.StoryID),
		LexicalForm:      row.LexicalForm,
		AudioPath:        row.AudioPath.String,
		AudioBucket:      row.AudioBucket.String,
		CorrectImagePath: row.CorrectImagePath.String,
		ImageBucket:      row.ImageBucket.String,
	}, nil
}

// CreateTargetVocabulary adds a target word to a story.
func CreateTargetVocabulary(ctx context.Context, word TargetVocabulary) (*TargetVocabulary, error) {
	if queries == nil {
		return nil, errors.New("database not initialized")
	}

	row, err := queries.CreateTargetVocabulary(ctx, db.CreateTargetVocabularyParams{
		StoryID:          int32(word.StoryID),
		LexicalForm:      word.LexicalForm,
		AudioPath:        optionalText(word.AudioPath),
		AudioBucket:      optionalText(word.AudioBucket),
		CorrectImagePath: optionalText(word.CorrectImagePath),
		ImageBucket:      optionalText(word.ImageBucket),
	})
	if err != nil {
		return nil, asDuplicate(err)
	}
	InvalidateStoryContentReadiness(int(row.StoryID))

	return &TargetVocabulary{
		ID:               int(row.ID),
		StoryID:          int(row.StoryID),
		LexicalForm:      row.LexicalForm,
		AudioPath:        row.AudioPath.String,
		AudioBucket:      row.AudioBucket.String,
		CorrectImagePath: row.CorrectImagePath.String,
		ImageBucket:      row.ImageBucket.String,
	}, nil
}

// UpdateTargetVocabulary replaces the lexical form and assets of a target word.
func UpdateTargetVocabulary(ctx context.Context, word TargetVocabulary) (*TargetVocabulary, error) {
	if queries == nil {
		return nil, errors.New("database not initialized")
	}

	row, err := queries.UpdateTargetVocabulary(ctx, db.UpdateTargetVocabularyParams{
		ID:               int32(word.ID),
		LexicalForm:      word.LexicalForm,
		AudioPath:        optionalText(word.AudioPath),
		AudioBucket:      optionalText(word.AudioBucket),
		CorrectImagePath: optionalText(word.CorrectImagePath),
		ImageBucket:      optionalText(word.ImageBucket),
	})
	if errors.Is(err, sql.ErrNoRows) || errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, asDuplicate(err)
	}
	InvalidateStoryContentReadiness(int(row.StoryID))

	return &TargetVocabulary{
		ID:               int(row.ID),
		StoryID:          int(row.StoryID),
		LexicalForm:      row.LexicalForm,
		AudioPath:        row.AudioPath.String,
		AudioBucket:      row.AudioBucket.String,
		CorrectImagePath: row.CorrectImagePath.String,
		ImageBucket:      row.ImageBucket.String,
	}, nil
}

// DeleteTargetVocabulary removes a single target word.
func DeleteTargetVocabulary(ctx context.Context, storyID, id int) error {
	if queries == nil {
		return errors.New("database not initialized")
	}
	if err := queries.DeleteTargetVocabulary(ctx, int32(id)); err != nil {
		return err
	}
	InvalidateStoryContentReadiness(storyID)
	return nil
}

// DeleteStoryTargetVocabulary removes every target word for a story.
func DeleteStoryTargetVocabulary(ctx context.Context, storyID int) error {
	if queries == nil {
		return errors.New("database not initialized")
	}
	if err := queries.DeleteStoryTargetVocabulary(ctx, int32(storyID)); err != nil {
		return err
	}
	InvalidateStoryContentReadiness(storyID)
	return nil
}

// CountStoryTargetVocabulary returns how many target words a story has.
func CountStoryTargetVocabulary(ctx context.Context, storyID int) (int, error) {
	if queries == nil {
		return 0, errors.New("database not initialized")
	}

	count, err := queries.CountStoryTargetVocabulary(ctx, int32(storyID))
	if err != nil {
		return 0, err
	}
	return int(count), nil
}

// GetTargetVocabularyOccurrences returns every place a target word appears in
// the story text, ordered by line and position.
func GetTargetVocabularyOccurrences(ctx context.Context, storyID int) ([]TargetVocabularyOccurrence, error) {
	if queries == nil {
		return nil, errors.New("database not initialized")
	}

	rows, err := queries.GetTargetVocabularyOccurrences(ctx, int32(storyID))
	if err != nil {
		return nil, err
	}

	occurrences := make([]TargetVocabularyOccurrence, 0, len(rows))
	for _, row := range rows {
		occurrences = append(occurrences, TargetVocabularyOccurrence{
			TargetVocabID: int(row.TargetVocabID),
			LexicalForm:   row.LexicalForm,
			VocabItemID:   int(row.VocabItemID),
			LineNumber:    int(row.LineNumber.Int32),
			Word:          row.Word,
			Position:      [2]int{int(row.PositionStart), int(row.PositionEnd)},
		})
	}

	return occurrences, nil
}

// LexicalFormCount is an annotated lexical form in a story and how many times it
// appears. The target-vocabulary editor lists these so an author can only pick
// words that already meet MinTargetWordOccurrences.
type LexicalFormCount struct {
	LexicalForm string `json:"lexicalForm"`
	Occurrences int    `json:"occurrences"`
}

// GetStoryLexicalFormCounts returns every annotated lexical form in a story with
// its occurrence count, ordered by lexical form.
func GetStoryLexicalFormCounts(ctx context.Context, storyID int) ([]LexicalFormCount, error) {
	if queries == nil {
		return nil, errors.New("database not initialized")
	}

	rows, err := queries.GetStoryLexicalFormCounts(ctx, pgtype.Int4{Int32: int32(storyID), Valid: true})
	if err != nil {
		return nil, err
	}

	counts := make([]LexicalFormCount, 0, len(rows))
	for _, row := range rows {
		counts = append(counts, LexicalFormCount{
			LexicalForm: row.LexicalForm,
			Occurrences: int(row.Occurrences),
		})
	}

	return counts, nil
}

// optionalText maps an empty string to a NULL text column.
func optionalText(s string) pgtype.Text {
	return pgtype.Text{String: s, Valid: s != ""}
}
