// glossias/src/pkg/models/edit.go
package models

import (
	"context"
	"errors"

	"glossias/src/pkg/generated/db"

	"github.com/jackc/pgx/v5/pgtype"
)

var (
	ErrInvalidStoryID    = errors.New("invalid story ID")
	ErrInvalidLineNumber = errors.New("invalid line number")
)

// EditStoryText updates only the text content of story lines
func EditStoryText(ctx context.Context, storyID int, lines []StoryLine) error {
	return withTransaction(func() error {
		// Delete existing lines using SQLC
		if err := queries.DeleteAllStoryLines(ctx, int32(storyID)); err != nil {
			return err
		}

		// Insert updated lines using SQLC
		for _, line := range lines {
			err := queries.UpsertStoryLine(ctx, db.UpsertStoryLineParams{
				StoryID:    int32(storyID),
				LineNumber: int32(line.LineNumber),
				Text:       line.Text,
				AudioFile:  pgtype.Text{String: "", Valid: false},
			})
			if err != nil {
				return err
			}
		}

		// Fetch existing story metadata
		existingStory, err := queries.GetStory(ctx, int32(storyID))
		if err != nil {
			return err
		}
		// Update last revision timestamp using SQLC, preserving existing metadata
		err = queries.UpdateStory(ctx, db.UpdateStoryParams{
			StoryID:      int32(storyID),
			WeekNumber:   existingStory.WeekNumber,
			DayLetter:    existingStory.DayLetter,
			GrammarPoint: existingStory.GrammarPoint,
			AuthorID:     existingStory.AuthorID,
			AuthorName:   existingStory.AuthorName,
			CourseID:     existingStory.CourseID,
		})
		if err != nil {
			return err
		}

		return nil
	})
}

// EditStoryMetadata updates the story's metadata fields
func EditStoryMetadata(ctx context.Context, storyID int, metadata StoryMetadata) error {
	return withTransaction(func() error {
		// Update main story table using SQLC
		courseID := pgtype.Int4{Valid: false}
		if metadata.CourseID != nil {
			courseID = pgtype.Int4{Int32: int32(*metadata.CourseID), Valid: true}
		}

		err := queries.UpdateStory(ctx, db.UpdateStoryParams{
			StoryID:      int32(storyID),
			WeekNumber:   int32(metadata.WeekNumber),
			DayLetter:    metadata.DayLetter,
			GrammarPoint: pgtype.Text{String: metadata.GrammarPoint, Valid: metadata.GrammarPoint != ""},
			AuthorID:     metadata.Author.ID,
			AuthorName:   metadata.Author.Name,
			CourseID:     courseID,
		})
		if err != nil {
			return err
		}

		// Update titles using SQLC
		if err := queries.DeleteStoryTitles(ctx, int32(storyID)); err != nil {
			return err
		}
		for lang, title := range metadata.Title {
			err := queries.UpsertStoryTitle(ctx, db.UpsertStoryTitleParams{
				StoryID:      int32(storyID),
				LanguageCode: lang,
				Title:        title,
			})
			if err != nil {
				return err
			}
		}

		// Update description using SQLC
		if err := queries.DeleteStoryDescriptions(ctx, int32(storyID)); err != nil {
			return err
		}

		if metadata.Description.Text != "" || metadata.Description.Language != "" {
			err := queries.UpsertStoryDescription(ctx, db.UpsertStoryDescriptionParams{
				StoryID:         int32(storyID),
				LanguageCode:    metadata.Description.Language,
				DescriptionText: metadata.Description.Text,
			})
			if err != nil {
				return err
			}
		}

		return nil
	})
}

// AddLineAnnotations updates grammar points, vocabulary, and footnotes for a specific line
func AddLineAnnotations(ctx context.Context, storyID int, lineNumber int, line StoryLine) error {
	return withTransaction(func() error {
		// Verify line exists
		exists, err := queries.LineExists(ctx, db.LineExistsParams{
			StoryID:    int32(storyID),
			LineNumber: int32(lineNumber),
		})
		if err != nil {
			return err
		}
		if !exists {
			return ErrInvalidLineNumber
		}

		// Insert vocabulary items
		for _, v := range line.Vocabulary {
			if err := dedupVocabularyInsert(ctx, storyID, lineNumber, v); err != nil {
				return err
			}
		}

		// Insert grammar items
		for _, g := range line.Grammar {
			if err := dedupGrammarInsert(ctx, storyID, lineNumber, g); err != nil {
				return err
			}
		}

		// Insert footnotes and their references
		// Insert footnotes
		for _, f := range line.Footnotes {
			if err := dedupFootnoteInsert(ctx, storyID, lineNumber, f); err != nil {
				return err
			}
		}

		// Update last revision timestamp - get existing values first
		story, err := queries.GetStory(ctx, int32(storyID))
		if err != nil {
			return err
		}

		err = queries.UpdateStory(ctx, db.UpdateStoryParams{
			StoryID:      story.StoryID,
			WeekNumber:   story.WeekNumber,
			DayLetter:    story.DayLetter,
			GrammarPoint: story.GrammarPoint,
			AuthorID:     story.AuthorID,
			AuthorName:   story.AuthorName,
			CourseID:     story.CourseID,
		})
		if err != nil {
			return err
		}

		return nil
	})
}

// ClearStoryAnnotations removes all annotations from a story while preserving the text and metadata
func ClearStoryAnnotations(ctx context.Context, storyID int) error {
	// Verify story exists first
	exists, err := queries.StoryExists(ctx, int32(storyID))
	if err != nil {
		return err
	}
	if !exists {
		return ErrNotFound
	}

	return withTransaction(func() error {
		// Delete all annotations
		if err := queries.DeleteAllStoryAnnotations(ctx, pgtype.Int4{Int32: int32(storyID), Valid: true}); err != nil {
			return err
		}
		if err := queries.DeleteAllVocabularyForStory(ctx, pgtype.Int4{Int32: int32(storyID), Valid: true}); err != nil {
			return err
		}
		if err := queries.DeleteAllGrammarForStory(ctx, pgtype.Int4{Int32: int32(storyID), Valid: true}); err != nil {
			return err
		}

		// Update last revision timestamp
		return queries.UpdateStoryRevision(ctx, int32(storyID))
	})
}

// ClearLineAnnotations removes all annotations from a specific line
func ClearLineAnnotations(ctx context.Context, storyID int, lineNumber int) error {
	return withTransaction(func() error {
		// Verify line exists
		exists, err := queries.LineExists(ctx, db.LineExistsParams{
			StoryID:    int32(storyID),
			LineNumber: int32(lineNumber),
		})
		if err != nil {
			return err
		}
		if !exists {
			return ErrInvalidLineNumber
		}

		// Delete footnote references first, then footnotes
		if err := queries.DeleteLineFootnoteReferences(ctx, db.DeleteLineFootnoteReferencesParams{
			StoryID:    pgtype.Int4{Int32: int32(storyID), Valid: true},
			LineNumber: pgtype.Int4{Int32: int32(lineNumber), Valid: true},
		}); err != nil {
			return err
		}

		// Delete line-specific annotations
		if err := queries.DeleteAllLineAnnotations(ctx, db.DeleteAllLineAnnotationsParams{
			StoryID:    pgtype.Int4{Int32: int32(storyID), Valid: true},
			LineNumber: pgtype.Int4{Int32: int32(lineNumber), Valid: true},
		}); err != nil {
			return err
		}
		if err := queries.DeleteLineVocabulary(ctx, db.DeleteLineVocabularyParams{
			StoryID:    pgtype.Int4{Int32: int32(storyID), Valid: true},
			LineNumber: pgtype.Int4{Int32: int32(lineNumber), Valid: true},
		}); err != nil {
			return err
		}
		if err := queries.DeleteLineGrammar(ctx, db.DeleteLineGrammarParams{
			StoryID:    pgtype.Int4{Int32: int32(storyID), Valid: true},
			LineNumber: pgtype.Int4{Int32: int32(lineNumber), Valid: true},
		}); err != nil {
			return err
		}

		// Update last revision timestamp
		return queries.UpdateStoryRevision(ctx, int32(storyID))
	})
}

// UpdateVocabularyAnnotation updates a specific vocabulary annotation by position
// UpdateVocabularyAnnotation updates a vocabulary annotation at a specific position
func UpdateVocabularyAnnotation(ctx context.Context, storyID int, lineNumber int, position [2]int, vocab VocabularyItem) error {
	return withTransaction(func() error {
		// Update the vocabulary item using SQLC
		err := queries.UpdateVocabularyByPosition(ctx, db.UpdateVocabularyByPositionParams{
			StoryID:       pgtype.Int4{Int32: int32(storyID), Valid: true},
			LineNumber:    pgtype.Int4{Int32: int32(lineNumber), Valid: true},
			PositionStart: int32(position[0]),
			PositionEnd:   int32(position[1]),
			Word:          vocab.Word,
			LexicalForm:   vocab.LexicalForm,
		})

		if err != nil {
			return err
		}

		// Update last revision timestamp
		return queries.UpdateStoryRevision(ctx, int32(storyID))
	})
}

// UpdateGrammarAnnotation updates a specific grammar annotation by position
// UpdateGrammarAnnotation updates a grammar annotation at a specific position
func UpdateGrammarAnnotation(ctx context.Context, storyID int, lineNumber int, position [2]int, grammar GrammarItem) error {
	return withTransaction(func() error {
		// Update the grammar item using SQLC
		err := queries.UpdateGrammarByPosition(ctx, db.UpdateGrammarByPositionParams{
			StoryID:       pgtype.Int4{Int32: int32(storyID), Valid: true},
			LineNumber:    pgtype.Int4{Int32: int32(lineNumber), Valid: true},
			PositionStart: int32(position[0]),
			PositionEnd:   int32(position[1]),
			Text:          grammar.Text,
		})

		if err != nil {
			return err
		}

		// Update last revision timestamp
		return queries.UpdateStoryRevision(ctx, int32(storyID))
	})
}

// UpdateVocabularyByWord updates a vocabulary item's lexical form by matching the word
// UpdateVocabularyByWord updates the lexical form of all vocabulary items with a specific word
func UpdateVocabularyByWord(ctx context.Context, storyID int, lineNumber int, word string, newLexicalForm string) error {
	return withTransaction(func() error {
		// Update the vocabulary item by word using SQLC
		err := queries.UpdateVocabularyByWord(ctx, db.UpdateVocabularyByWordParams{
			StoryID:     pgtype.Int4{Int32: int32(storyID), Valid: true},
			LineNumber:  pgtype.Int4{Int32: int32(lineNumber), Valid: true},
			Word:        word,
			LexicalForm: newLexicalForm,
		})
		if err != nil {
			return err
		}

		// Update last revision timestamp
		return queries.UpdateStoryRevision(ctx, int32(storyID))
	})
}

// UpdateFootnoteAnnotation updates a specific footnote by ID
// UpdateFootnoteAnnotation updates a footnote and its references
func UpdateFootnoteAnnotation(ctx context.Context, storyID int, footnoteID int, footnote Footnote) error {
	return withTransaction(func() error {
		// Update the footnote text using SQLC
		err := queries.UpdateFootnote(ctx, db.UpdateFootnoteParams{
			ID:           int32(footnoteID),
			StoryID:      pgtype.Int4{Int32: int32(storyID), Valid: true},
			FootnoteText: footnote.Text,
		})
		if err != nil {
			return err
		}

		// Delete existing references using SQLC
		if err := queries.DeleteFootnoteReferences(ctx, int32(footnoteID)); err != nil {
			return err
		}

		// Insert new references using SQLC
		for _, ref := range footnote.References {
			err := queries.CreateFootnoteReference(ctx, db.CreateFootnoteReferenceParams{
				FootnoteID: int32(footnoteID),
				Reference:  ref,
			})
			if err != nil {
				return err
			}
		}

		// Update last revision timestamp
		return queries.UpdateStoryRevision(ctx, int32(storyID))
	})
}
