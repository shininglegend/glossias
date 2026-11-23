// glossias/src/pkg/models/edit.go
package models

import (
	"context"
	"errors"

	"glossias/src/pkg/generated/db"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

var (
	ErrInvalidStoryID    = errors.New("invalid story ID")
	ErrInvalidLineNumber = errors.New("invalid line number")
)

// touchStoryRevision bumps the story's last_revision in one statement. A
// missing story is pgx.ErrNoRows, as the GetStory + UpdateStory pair it
// replaces returned.
func touchStoryRevision(ctx context.Context, storyID int) error {
	n, err := queries.TouchStoryRevision(ctx, int32(storyID))
	if err != nil {
		return err
	}
	if n == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

// EditStoryText updates only the text content of story lines
func EditStoryText(ctx context.Context, storyID int, lines []StoryLine) error {
	err := withTransaction(ctx, func(txCtx context.Context) error {
		// Delete existing lines using SQLC
		if err := queries.DeleteAllStoryLines(txCtx, int32(storyID)); err != nil {
			return err
		}

		// Insert updated lines in one COPY
		if len(lines) > 0 {
			rows := make([]db.BulkCreateStoryLinesParams, 0, len(lines))
			for _, line := range lines {
				rows = append(rows, db.BulkCreateStoryLinesParams{
					StoryID:    int32(storyID),
					LineNumber: int32(line.LineNumber),
					Text:       line.Text,
				})
			}
			if _, err := queries.BulkCreateStoryLines(txCtx, rows); err != nil {
				return err
			}
		}

		return touchStoryRevision(txCtx, storyID)
	})

	// Invalidate cache after successful edit
	if err == nil {
		// Get all users who might have this story cached - for now, invalidate for all users
		// In a production system, you might want to track which users have accessed this story
		InvalidateStoryMetadata(storyID)
	}

	return err
}

// EditStoryMetadata updates the story's metadata fields
func EditStoryMetadata(ctx context.Context, storyID int, metadata StoryMetadata) error {
	err := withTransaction(ctx, func(txCtx context.Context) error {
		// Update main story table using SQLC
		courseID := pgtype.Int4{Valid: false}
		if metadata.CourseID != nil {
			courseID = pgtype.Int4{Int32: int32(*metadata.CourseID), Valid: true}
		}

		err := queries.UpdateStory(txCtx, db.UpdateStoryParams{
			StoryID:    int32(storyID),
			WeekNumber: int32(metadata.WeekNumber),
			DayLetter:  metadata.DayLetter,
			VideoUrl:   pgtype.Text{String: metadata.VideoURL, Valid: metadata.VideoURL != ""},
			AuthorID:   metadata.Author.ID,
			AuthorName: metadata.Author.Name,
			CourseID:   courseID,
		})
		if err != nil {
			return err
		}
		if err := ensureOwnerLinked(txCtx, int32(storyID), metadata.CourseID); err != nil {
			return err
		}

		// Update titles using SQLC
		if err := queries.DeleteStoryTitles(txCtx, int32(storyID)); err != nil {
			return err
		}
		if err := upsertStoryTitles(txCtx, storyID, metadata.Title); err != nil {
			return err
		}

		// Update description using SQLC
		if err := queries.DeleteStoryDescriptions(txCtx, int32(storyID)); err != nil {
			return err
		}

		if metadata.Description.Text != "" || metadata.Language != "" {
			err := queries.UpsertStoryDescription(txCtx, db.UpsertStoryDescriptionParams{
				StoryID:         int32(storyID),
				LanguageCode:    metadata.Language,
				DescriptionText: metadata.Description.Text,
			})
			if err != nil {
				return err
			}
		}

		// Update grammar points
		if err := ClearStoryGrammarPoints(txCtx, storyID); err != nil {
			return err
		}
		if len(metadata.GrammarPoints) > 0 {
			p := db.InsertGrammarPointsParams{
				StoryID:      int32(storyID),
				Names:        make([]string, 0, len(metadata.GrammarPoints)),
				Descriptions: make([]string, 0, len(metadata.GrammarPoints)),
			}
			for _, gp := range metadata.GrammarPoints {
				p.Names = append(p.Names, gp.Name)
				p.Descriptions = append(p.Descriptions, gp.Description)
			}
			if err := queries.InsertGrammarPoints(txCtx, p); err != nil {
				return err
			}
		}

		return nil
	})

	// Invalidate cache after successful edit
	if err == nil {
		InvalidateStoryMetadata(storyID)
	}

	return err
}

// AddLineAnnotations updates grammar points, vocabulary, and footnotes for a specific line
func AddLineAnnotations(ctx context.Context, storyID int, lineNumber int, line StoryLine) error {
	err := withTransaction(ctx, func(txCtx context.Context) error {
		// Verify line exists
		exists, err := queries.LineExists(txCtx, db.LineExistsParams{
			StoryID:    int32(storyID),
			LineNumber: int32(lineNumber),
		})
		if err != nil {
			return err
		}
		if !exists {
			return ErrInvalidLineNumber
		}

		// Vocabulary, grammar and footnotes (with references): dedup checks
		// and inserts, a constant number of statements. Audio is not part of
		// line annotations.
		batch := newAnnotationBatch(storyID)
		batch.addAnnotations(lineNumber, line)
		if err := batch.write(txCtx); err != nil {
			return err
		}

		return touchStoryRevision(txCtx, storyID)
	})

	// Invalidate cache after successful edit
	if err == nil {
		InvalidateStoryMetadata(storyID)
		// Also invalidate line-specific cache
		if cacheInstance != nil && keyBuilder != nil {
			lineKey := keyBuilder.LineAnnotations(storyID, lineNumber)
			_ = cacheInstance.Delete(lineKey)
		}
	}

	return err
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

	err = withTransaction(ctx, func(txCtx context.Context) error {
		// Delete all annotations
		if err := queries.DeleteAllStoryAnnotations(txCtx, pgtype.Int4{Int32: int32(storyID), Valid: true}); err != nil {
			return err
		}
		if err := queries.DeleteAllVocabularyForStory(txCtx, pgtype.Int4{Int32: int32(storyID), Valid: true}); err != nil {
			return err
		}
		if err := queries.DeleteAllGrammarForStory(txCtx, pgtype.Int4{Int32: int32(storyID), Valid: true}); err != nil {
			return err
		}

		// Update last revision timestamp
		return queries.UpdateStoryRevision(txCtx, int32(storyID))
	})

	// Invalidate cache after successful clear
	if err == nil {
		InvalidateStoryMetadata(storyID)
	}

	return err
}

// ClearLineAnnotations removes all annotations from a specific line
func ClearLineAnnotations(ctx context.Context, storyID int, lineNumber int) error {
	err := withTransaction(ctx, func(txCtx context.Context) error {
		// Verify line exists
		exists, err := queries.LineExists(txCtx, db.LineExistsParams{
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
		if err := queries.DeleteLineFootnoteReferences(txCtx, db.DeleteLineFootnoteReferencesParams{
			StoryID:    pgtype.Int4{Int32: int32(storyID), Valid: true},
			LineNumber: pgtype.Int4{Int32: int32(lineNumber), Valid: true},
		}); err != nil {
			return err
		}

		// Delete line-specific annotations
		if err := queries.DeleteAllLineAnnotations(txCtx, db.DeleteAllLineAnnotationsParams{
			StoryID:    pgtype.Int4{Int32: int32(storyID), Valid: true},
			LineNumber: pgtype.Int4{Int32: int32(lineNumber), Valid: true},
		}); err != nil {
			return err
		}
		if err := queries.DeleteLineVocabulary(txCtx, db.DeleteLineVocabularyParams{
			StoryID:    pgtype.Int4{Int32: int32(storyID), Valid: true},
			LineNumber: pgtype.Int4{Int32: int32(lineNumber), Valid: true},
		}); err != nil {
			return err
		}
		if err := queries.DeleteLineGrammar(txCtx, db.DeleteLineGrammarParams{
			StoryID:    pgtype.Int4{Int32: int32(storyID), Valid: true},
			LineNumber: pgtype.Int4{Int32: int32(lineNumber), Valid: true},
		}); err != nil {
			return err
		}

		// Update last revision timestamp
		return queries.UpdateStoryRevision(txCtx, int32(storyID))
	})

	// Invalidate cache after successful clear
	if err == nil {
		InvalidateStoryMetadata(storyID)
		// Also invalidate line-specific cache
		if cacheInstance != nil && keyBuilder != nil {
			lineKey := keyBuilder.LineAnnotations(storyID, lineNumber)
			_ = cacheInstance.Delete(lineKey)
		}
	}

	return err
}

// UpdateVocabularyAnnotation updates a vocabulary annotation at a specific position
func UpdateVocabularyAnnotation(ctx context.Context, storyID int, lineNumber int, position [2]int, vocab VocabularyItem) error {
	err := withTransaction(ctx, func(txCtx context.Context) error {
		// Update the vocabulary item using SQLC
		err := queries.UpdateVocabularyByPosition(txCtx, db.UpdateVocabularyByPositionParams{
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
		return queries.UpdateStoryRevision(txCtx, int32(storyID))
	})

	// Invalidate cache after successful update
	if err == nil {
		InvalidateStoryMetadata(storyID)
		// Also invalidate line-specific cache
		if cacheInstance != nil && keyBuilder != nil {
			lineKey := keyBuilder.LineAnnotations(storyID, lineNumber)
			_ = cacheInstance.Delete(lineKey)
		}
	}

	return err
}

// UpdateGrammarAnnotation updates a grammar annotation at a specific position
func UpdateGrammarAnnotation(ctx context.Context, storyID int, lineNumber int, position [2]int, grammar GrammarItem) error {
	err := withTransaction(ctx, func(txCtx context.Context) error {
		// Update the grammar item using SQLC
		grammarPointID := pgtype.Int4{Valid: false}
		if grammar.GrammarPointID != nil {
			grammarPointID = pgtype.Int4{Int32: int32(*grammar.GrammarPointID), Valid: true}
		}
		err := queries.UpdateGrammarByPosition(txCtx, db.UpdateGrammarByPositionParams{
			StoryID:        pgtype.Int4{Int32: int32(storyID), Valid: true},
			LineNumber:     pgtype.Int4{Int32: int32(lineNumber), Valid: true},
			PositionStart:  int32(position[0]),
			PositionEnd:    int32(position[1]),
			GrammarPointID: grammarPointID,
			Text:           grammar.Text,
		})

		if err != nil {
			return err
		}

		// Update last revision timestamp
		return queries.UpdateStoryRevision(txCtx, int32(storyID))
	})

	// Invalidate cache after successful update
	if err == nil {
		InvalidateStoryMetadata(storyID)
		// Also invalidate line-specific cache
		if cacheInstance != nil && keyBuilder != nil {
			lineKey := keyBuilder.LineAnnotations(storyID, lineNumber)
			_ = cacheInstance.Delete(lineKey)
		}
	}

	return err
}

// UpdateVocabularyByWord updates the lexical form of all vocabulary items with a specific word
func UpdateVocabularyByWord(ctx context.Context, storyID int, lineNumber int, word string, newLexicalForm string) error {
	err := withTransaction(ctx, func(txCtx context.Context) error {
		// Update the vocabulary item by word using SQLC
		err := queries.UpdateVocabularyByWord(txCtx, db.UpdateVocabularyByWordParams{
			StoryID:     pgtype.Int4{Int32: int32(storyID), Valid: true},
			LineNumber:  pgtype.Int4{Int32: int32(lineNumber), Valid: true},
			Word:        word,
			LexicalForm: newLexicalForm,
		})
		if err != nil {
			return err
		}

		// Update last revision timestamp
		return queries.UpdateStoryRevision(txCtx, int32(storyID))
	})

	// Invalidate cache after successful update
	if err == nil {
		InvalidateStoryMetadata(storyID)
		// Also invalidate line-specific cache
		if cacheInstance != nil && keyBuilder != nil {
			lineKey := keyBuilder.LineAnnotations(storyID, lineNumber)
			_ = cacheInstance.Delete(lineKey)
		}
	}

	return err
}

// UpdateFootnoteAnnotation updates a footnote and its references
func UpdateFootnoteAnnotation(ctx context.Context, storyID int, footnoteID int, footnote Footnote) error {
	err := withTransaction(ctx, func(txCtx context.Context) error {
		// Update the footnote text using SQLC
		err := queries.UpdateFootnote(txCtx, db.UpdateFootnoteParams{
			ID:           int32(footnoteID),
			StoryID:      pgtype.Int4{Int32: int32(storyID), Valid: true},
			FootnoteText: footnote.Text,
		})
		if err != nil {
			return err
		}

		// Delete existing references using SQLC
		if err := queries.DeleteFootnoteReferences(txCtx, int32(footnoteID)); err != nil {
			return err
		}

		// Insert new references in one statement
		if len(footnote.References) > 0 {
			if err := queries.InsertFootnoteReferences(txCtx, db.InsertFootnoteReferencesParams{
				FootnoteID:     int32(footnoteID),
				ReferenceTexts: footnote.References,
			}); err != nil {
				return err
			}
		}

		// Update last revision timestamp
		return queries.UpdateStoryRevision(txCtx, int32(storyID))
	})

	// Invalidate cache after successful update
	if err == nil {
		InvalidateStoryMetadata(storyID)
	}

	return err
}
