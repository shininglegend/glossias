// glossias/src/pkg/models/save.go
package models

import (
	"context"

	"glossias/src/pkg/generated/db"

	"github.com/jackc/pgx/v5/pgtype"
	_ "github.com/lib/pq"
)

func SaveNewStory(ctx context.Context, story *Story) error {
	err := withTransaction(ctx, func(txCtx context.Context) error {
		// Create story using SQLC
		courseID := pgtype.Int4{Valid: false}
		if story.Metadata.CourseID != nil {
			courseID = pgtype.Int4{Int32: int32(*story.Metadata.CourseID), Valid: true}
		}

		result, err := queries.CreateStory(txCtx, db.CreateStoryParams{
			WeekNumber: int32(story.Metadata.WeekNumber),
			DayLetter:  story.Metadata.DayLetter,
			VideoUrl:   pgtype.Text{String: story.Metadata.VideoURL, Valid: story.Metadata.VideoURL != ""},
			AuthorID:   story.Metadata.Author.ID,
			AuthorName: story.Metadata.Author.Name,
			CourseID:   courseID,
		})
		if err != nil {
			return err
		}

		story.Metadata.StoryID = int(result.StoryID)
		if err := ensureOwnerLinked(txCtx, result.StoryID, story.Metadata.CourseID); err != nil {
			return err
		}
		return saveStoryComponents(txCtx, story)
	})

	// Invalidate cache after successful save
	if err == nil {
		InvalidateStoryMetadata(story.Metadata.StoryID)
	}

	return err
}

func SaveStoryData(ctx context.Context, storyID int, story *Story) error {
	exists, err := queries.StoryExists(ctx, int32(storyID))
	if err != nil {
		return err
	}
	if !exists {
		return ErrNotFound
	}

	err = withTransaction(ctx, func(txCtx context.Context) error {
		// Update story using SQLC
		courseID := pgtype.Int4{Valid: false}
		if story.Metadata.CourseID != nil {
			courseID = pgtype.Int4{Int32: int32(*story.Metadata.CourseID), Valid: true}
		}

		err := queries.UpdateStory(txCtx, db.UpdateStoryParams{
			StoryID:    int32(storyID),
			WeekNumber: int32(story.Metadata.WeekNumber),
			DayLetter:  story.Metadata.DayLetter,
			VideoUrl:   pgtype.Text{String: story.Metadata.VideoURL, Valid: story.Metadata.VideoURL != ""},
			AuthorID:   story.Metadata.Author.ID,
			AuthorName: story.Metadata.Author.Name,
			CourseID:   courseID,
		})
		if err != nil {
			return err
		}
		if err := ensureOwnerLinked(txCtx, int32(storyID), story.Metadata.CourseID); err != nil {
			return err
		}

		return saveStoryComponents(txCtx, story)
	})

	// Invalidate cache after successful save
	if err == nil {
		InvalidateStoryMetadata(storyID)
	}

	return err
}

// saveStoryComponents writes titles, description, lines and every line's
// annotations and audio with a fixed number of statements, whatever the
// story's size. ctx must be the transaction context.
func saveStoryComponents(ctx context.Context, story *Story) error {
	storyID := story.Metadata.StoryID
	if err := upsertStoryTitles(ctx, storyID, story.Metadata.Title); err != nil {
		return err
	}

	if story.Metadata.Description.Text != "" || story.Metadata.Language != "" {
		if err := queries.UpsertStoryDescription(ctx, db.UpsertStoryDescriptionParams{
			StoryID:         int32(storyID),
			LanguageCode:    story.Metadata.Language,
			DescriptionText: story.Metadata.Description.Text,
		}); err != nil {
			return err
		}
	}

	return saveLines(ctx, storyID, story.Content.Lines)
}

// upsertStoryTitles writes every language's title in one statement.
func upsertStoryTitles(ctx context.Context, storyID int, titles map[string]string) error {
	if len(titles) == 0 {
		return nil
	}
	p := db.UpsertStoryTitlesParams{
		StoryID:       int32(storyID),
		LanguageCodes: make([]string, 0, len(titles)),
		Titles:        make([]string, 0, len(titles)),
	}
	for lang, title := range titles {
		p.LanguageCodes = append(p.LanguageCodes, lang)
		p.Titles = append(p.Titles, title)
	}
	return queries.UpsertStoryTitles(ctx, p)
}

// saveLines inserts the lines (plain inserts: a line that already exists
// fails on story_lines_pkey, as before), then their annotations and audio.
func saveLines(ctx context.Context, storyID int, lines []StoryLine) error {
	if len(lines) == 0 {
		return nil
	}
	rows := make([]db.BulkCreateStoryLinesParams, 0, len(lines))
	batch := newAnnotationBatch(storyID)
	for _, line := range lines {
		rows = append(rows, db.BulkCreateStoryLinesParams{
			StoryID:    int32(storyID),
			LineNumber: int32(line.LineNumber),
			Text:       line.Text,
		})
		batch.addAnnotations(line.LineNumber, line)
		batch.addAudio(line.LineNumber, line.AudioFiles)
	}
	if _, err := queries.BulkCreateStoryLines(ctx, rows); err != nil {
		return err
	}
	return batch.write(ctx)
}
