// glossias/src/pkg/models/save.go
package models

import (
	"context"

	"glossias/src/pkg/generated/db"

	"github.com/jackc/pgx/v5/pgtype"
	_ "github.com/lib/pq"
)

func SaveNewStory(ctx context.Context, story *Story) error {
	return withTransaction(func() error {
		// Create story using SQLC
		result, err := queries.CreateStory(ctx, db.CreateStoryParams{
			WeekNumber:   int32(story.Metadata.WeekNumber),
			DayLetter:    story.Metadata.DayLetter,
			GrammarPoint: pgtype.Text{String: story.Metadata.GrammarPoint, Valid: story.Metadata.GrammarPoint != ""},
			AuthorID:     story.Metadata.Author.ID,
			AuthorName:   story.Metadata.Author.Name,
			CourseID:     pgtype.Int4{Int32: 0, Valid: false}, // Default null course
		})
		if err != nil {
			return err
		}

		story.Metadata.StoryID = int(result.StoryID)
		return saveStoryComponents(ctx, story)
	})
}

func SaveStoryData(ctx context.Context, storyID int, story *Story) error {
	exists, err := queries.StoryExists(ctx, int32(storyID))
	if err != nil {
		return err
	}
	if !exists {
		return ErrNotFound
	}

	return withTransaction(func() error {
		// Update story using SQLC
		err := queries.UpdateStory(ctx, db.UpdateStoryParams{
			StoryID:      int32(storyID),
			WeekNumber:   int32(story.Metadata.WeekNumber),
			DayLetter:    story.Metadata.DayLetter,
			GrammarPoint: pgtype.Text{String: story.Metadata.GrammarPoint, Valid: story.Metadata.GrammarPoint != ""},
			AuthorID:     story.Metadata.Author.ID,
			AuthorName:   story.Metadata.Author.Name,
			CourseID:     pgtype.Int4{Int32: 0, Valid: false},
		})
		if err != nil {
			return err
		}

		if err := saveStoryComponents(ctx, story); err != nil {
			return err
		}

		return saveStoryComponents(ctx, story)
	})
}

func saveStoryComponents(ctx context.Context, story *Story) error {
	// Save titles using SQLC
	for lang, title := range story.Metadata.Title {
		if err := queries.UpsertStoryTitle(ctx, db.UpsertStoryTitleParams{
			StoryID:      int32(story.Metadata.StoryID),
			LanguageCode: lang,
			Title:        title,
		}); err != nil {
			return err
		}
	}

	// Save description using SQLC
	if story.Metadata.Description.Text != "" || story.Metadata.Description.Language != "" {
		if err := queries.UpsertStoryDescription(ctx, db.UpsertStoryDescriptionParams{
			StoryID:         int32(story.Metadata.StoryID),
			LanguageCode:    story.Metadata.Description.Language,
			DescriptionText: story.Metadata.Description.Text,
		}); err != nil {
			return err
		}
	}

	return saveLines(ctx, story.Metadata.StoryID, story.Content.Lines)
}

func saveLines(ctx context.Context, storyID int, lines []StoryLine) error {
	for _, line := range lines {
		if err := saveLine(ctx, storyID, &line); err != nil {
			return err
		}
	}
	return nil
}

func saveLine(ctx context.Context, storyID int, line *StoryLine) error {
	// Save line using SQLC
	audioFile := pgtype.Text{String: "", Valid: false}
	if line.AudioFile != nil {
		audioFile = pgtype.Text{String: *line.AudioFile, Valid: true}
	}
	err := queries.UpsertStoryLine(ctx, db.UpsertStoryLineParams{
		StoryID:    int32(storyID),
		LineNumber: int32(line.LineNumber),
		Text:       line.Text,
		AudioFile:  audioFile,
	})
	if err != nil {
		return err
	}

	// Save vocabulary
	for _, v := range line.Vocabulary {
		if err := dedupVocabularyInsert(ctx, storyID, line.LineNumber, v); err != nil {
			return err
		}
	}

	// Save grammar items
	for _, g := range line.Grammar {
		if err := dedupGrammarInsert(ctx, storyID, line.LineNumber, g); err != nil {
			return err
		}
	}

	// Save footnotes
	for _, f := range line.Footnotes {
		if err := dedupFootnoteInsert(ctx, storyID, line.LineNumber, f); err != nil {
			return err
		}
	}

	return nil
}
