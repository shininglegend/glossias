package models

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

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
// ImagePath/ImageBucket and AudioPath/AudioBucket are the source of truth for
// the card's picture and optional audio override; ImageURL/AudioURL are signed
// read URLs filled in on demand by SignRecallSentenceURLs and are never
// persisted. An empty audio path means the student hears the story-line
// narration that makes up HebrewText.
type RecallSentence struct {
	ID            int    `json:"id"`
	StoryID       int    `json:"storyId"`
	SequenceOrder int    `json:"sequenceOrder"`
	HebrewText    string `json:"hebrewText"`
	TargetVocabID *int   `json:"targetVocabId,omitempty"`
	ImagePath     string `json:"imagePath,omitempty"`
	ImageBucket   string `json:"imageBucket,omitempty"`
	ImageURL      string `json:"imageUrl,omitempty"`
	AudioPath     string `json:"audioPath,omitempty"`
	AudioBucket   string `json:"audioBucket,omitempty"`
	AudioURL      string `json:"audioUrl,omitempty"`
	// StoryAudioURLs are signed story-line clips for this sentence when no
	// override is uploaded. Filled on demand, never persisted.
	StoryAudioURLs []string `json:"storyAudioUrls,omitempty"`
}

func recallSentenceFromRow(
	id, storyID, sequenceOrder int32,
	hebrewText string,
	targetVocabID pgtype.Int4,
	imagePath, imageBucket, audioPath, audioBucket pgtype.Text,
) RecallSentence {
	sentence := RecallSentence{
		ID:            int(id),
		StoryID:       int(storyID),
		SequenceOrder: int(sequenceOrder),
		HebrewText:    hebrewText,
		ImagePath:     imagePath.String,
		ImageBucket:   imageBucket.String,
		AudioPath:     audioPath.String,
		AudioBucket:   audioBucket.String,
	}
	if targetVocabID.Valid {
		targetID := int(targetVocabID.Int32)
		sentence.TargetVocabID = &targetID
	}
	return sentence
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
		sentences = append(sentences, recallSentenceFromRow(
			row.ID, row.StoryID, row.SequenceOrder, row.HebrewText,
			row.TargetVocabID, row.ImagePath, row.ImageBucket, row.AudioPath, row.AudioBucket,
		))
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

	sentence := recallSentenceFromRow(
		row.ID, row.StoryID, row.SequenceOrder, row.HebrewText,
		row.TargetVocabID, row.ImagePath, row.ImageBucket, row.AudioPath, row.AudioBucket,
	)
	return &sentence, nil
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
		AudioPath:     optionalText(sentence.AudioPath),
		AudioBucket:   optionalText(sentence.AudioBucket),
	})
	if err != nil {
		return nil, err
	}

	saved := recallSentenceFromRow(
		row.ID, row.StoryID, row.SequenceOrder, row.HebrewText,
		row.TargetVocabID, row.ImagePath, row.ImageBucket, row.AudioPath, row.AudioBucket,
	)

	InvalidateStoryContentReadiness(saved.StoryID)
	return &saved, nil
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

// SaveRecallPick scores one sequential pick: the student claims sentenceID
// occurs at 1-based position in the story. It logs a single correct or
// incorrect answer row and returns whether the pick was right.
func SaveRecallPick(ctx context.Context, userID string, storyID, sentenceID, position int) (bool, error) {
	if queries == nil {
		return false, errors.New("database not initialized")
	}

	sentences, err := GetStoryRecallSentences(ctx, storyID)
	if err != nil {
		return false, err
	}
	if len(sentences) == 0 {
		return false, ErrNotFound
	}

	var found *RecallSentence
	for i := range sentences {
		if sentences[i].ID == sentenceID {
			found = &sentences[i]
			break
		}
	}
	if found == nil {
		return false, fmt.Errorf("%w: sentence %d does not belong to story %d", ErrInvalidRecallOrder, sentenceID, storyID)
	}
	if position < 1 || position > len(sentences) {
		return false, fmt.Errorf("%w: position %d is out of range", ErrInvalidRecallOrder, position)
	}

	correct := found.SequenceOrder == position
	if correct {
		err = queries.SaveRecallCorrectAnswer(ctx, db.SaveRecallCorrectAnswerParams{
			UserID:           userID,
			StoryID:          int32(storyID),
			RecallSentenceID: int32(sentenceID),
		})
	} else {
		err = queries.SaveRecallIncorrectAnswer(ctx, db.SaveRecallIncorrectAnswerParams{
			UserID:           userID,
			StoryID:          int32(storyID),
			RecallSentenceID: int32(sentenceID),
			SelectedPosition: int32(position),
		})
	}
	if err != nil {
		return false, err
	}
	return correct, nil
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

// GetUserRecallCorrectSentenceIDs returns the ID of every sentence the user
// has placed correctly, one entry per correct answer row (so a sentence placed
// correctly on two attempts appears twice). Callers judging completion should
// treat it as a set.
func GetUserRecallCorrectSentenceIDs(ctx context.Context, userID string, storyID int) ([]int, error) {
	if queries == nil {
		return nil, errors.New("database not initialized")
	}

	rows, err := queries.GetUserRecallCorrectAnswers(ctx, db.GetUserRecallCorrectAnswersParams{
		UserID:  userID,
		StoryID: int32(storyID),
	})
	if err != nil {
		return nil, err
	}

	ids := make([]int, 0, len(rows))
	for _, row := range rows {
		ids = append(ids, int(row.RecallSentenceID))
	}
	return ids, nil
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

// RecallSentenceCoveredLines is the story lines whose narration makes up
// hebrew, in story order. A sentence contained in a line uses that line; a
// join of several lines uses every line whose text appears in the sentence.
func RecallSentenceCoveredLines(hebrew string, lines []StoryLine) []int {
	text := strings.TrimSpace(hebrew)
	if text == "" {
		return nil
	}

	var contained []int
	for _, line := range lines {
		if strings.Contains(line.Text, text) {
			contained = append(contained, line.LineNumber)
		}
	}
	if len(contained) > 0 {
		return contained
	}

	var parts []int
	for _, line := range lines {
		lineText := strings.TrimSpace(line.Text)
		if lineText != "" && strings.Contains(text, lineText) {
			parts = append(parts, line.LineNumber)
		}
	}
	return parts
}

// RecallSentenceHasAudio is true when the sentence has an uploaded override or
// every covered story line has narration.
func RecallSentenceHasAudio(s RecallSentence, lines []StoryLine, linesWithAudio map[int]bool) bool {
	if s.AudioPath != "" && s.AudioBucket != "" {
		return true
	}
	covered := RecallSentenceCoveredLines(s.HebrewText, lines)
	if len(covered) == 0 {
		return false
	}
	for _, n := range covered {
		if !linesWithAudio[n] {
			return false
		}
	}
	return true
}

// SignCompleteLineAudioURLs signs each "complete" narration file for a story,
// keyed by 1-based line number.
func SignCompleteLineAudioURLs(ctx context.Context, storyID, expiresInSeconds int) (map[int]string, error) {
	files, err := GetStoryAudioFilesByLabel(ctx, storyID, "complete")
	if err != nil {
		return nil, err
	}
	urls := make(map[int]string, len(files))
	for _, file := range files {
		url, err := GetSignedURLForPath(ctx, file.FileBucket, file.FilePath, expiresInSeconds)
		if err != nil {
			return nil, err
		}
		urls[file.LineNumber] = url
	}
	return urls, nil
}

// AttachRecallStoryAudioURLs fills StoryAudioURLs from signed line clips.
func AttachRecallStoryAudioURLs(sentences []RecallSentence, lines []StoryLine, audioURLs map[int]string) {
	for i := range sentences {
		var urls []string
		for _, lineNumber := range RecallSentenceCoveredLines(sentences[i].HebrewText, lines) {
			if url, ok := audioURLs[lineNumber]; ok {
				urls = append(urls, url)
			}
		}
		sentences[i].StoryAudioURLs = urls
	}
}
