// glossias/src/pkg/models/delete.go
package models

import (
	"context"

	"github.com/jackc/pgx/v5/pgtype"
)

// Delete removes a story and all its associated data from the database
func Delete(ctx context.Context, storyID int) error {
	// Verify story exists first
	exists, err := queries.StoryExists(ctx, int32(storyID))
	if err != nil {
		return err
	}
	if !exists {
		return ErrNotFound
	}

	return withTransaction(func() error {
		// Delete in proper order to respect foreign key relationships
		// Though CASCADE would handle this, we're explicit for control
		if err := deleteFootnoteData(ctx, storyID); err != nil {
			return err
		}

		if err := deleteAnnotations(ctx, storyID); err != nil {
			return err
		}

		if err := deleteAudioFiles(ctx, storyID); err != nil {
			return err
		}

		if err := deleteStoryContent(ctx, storyID); err != nil {
			return err
		}

		if err := deleteMetadata(ctx, storyID); err != nil {
			return err
		}

		if err := deleteStoryGrammarPoints(ctx, storyID); err != nil {
			return err
		}

		// Finally delete the story itself using SQLC
		if err := queries.DeleteStory(ctx, int32(storyID)); err != nil {
			return err
		}

		return nil
	})
}

// deleteFootnoteData removes footnotes and their references
func deleteFootnoteData(ctx context.Context, storyID int) error {
	// Due to CASCADE, we only need to delete footnotes
	return queries.DeleteAllStoryAnnotations(ctx, pgtype.Int4{Int32: int32(storyID), Valid: true})
}

// deleteAnnotations removes vocabulary and grammar items
func deleteAnnotations(ctx context.Context, storyID int) error {
	if err := queries.DeleteAllVocabularyForStory(ctx, pgtype.Int4{Int32: int32(storyID), Valid: true}); err != nil {
		return err
	}
	return queries.DeleteAllGrammarForStory(ctx, pgtype.Int4{Int32: int32(storyID), Valid: true})
}

// deleteStoryContent removes the story lines using SQLC
func deleteStoryContent(ctx context.Context, storyID int) error {
	return queries.DeleteAllStoryLines(ctx, int32(storyID))
}

// deleteAudioFiles removes audio files using SQLC
func deleteAudioFiles(ctx context.Context, storyID int) error {
	return queries.DeleteStoryAudioFiles(ctx, pgtype.Int4{Int32: int32(storyID), Valid: true})
}

// deleteMetadata removes titles and descriptions using SQLC
func deleteMetadata(ctx context.Context, storyID int) error {
	if err := queries.DeleteStoryTitles(ctx, int32(storyID)); err != nil {
		return err
	}
	return queries.DeleteStoryDescriptions(ctx, int32(storyID))
}

// deleteStoryGrammarPoints removes story grammar point associations
func deleteStoryGrammarPoints(ctx context.Context, storyID int) error {
	return queries.ClearStoryGrammarPoints(ctx, int32(storyID))
}
