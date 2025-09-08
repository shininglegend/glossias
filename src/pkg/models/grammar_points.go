package models

import (
	"context"
	"database/sql"

	"glossias/src/pkg/generated/db"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

// CreateGrammarPoint creates a new grammar point
func CreateGrammarPoint(ctx context.Context, name, description string) (*GrammarPoint, error) {
	result, err := queries.CreateGrammarPoint(ctx, db.CreateGrammarPointParams{
		Name:        name,
		Description: pgtype.Text{String: description, Valid: description != ""},
	})
	if err != nil {
		return nil, err
	}

	return &GrammarPoint{
		ID:          int(result.GrammarPointID),
		Name:        result.Name,
		Description: result.Description.String,
	}, nil
}

// GetGrammarPoint retrieves a grammar point by ID
func GetGrammarPoint(ctx context.Context, grammarPointID int) (*GrammarPoint, error) {
	result, err := queries.GetGrammarPoint(ctx, int32(grammarPointID))
	if err == sql.ErrNoRows || err == pgx.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	return &GrammarPoint{
		ID:          int(result.GrammarPointID),
		Name:        result.Name,
		Description: result.Description.String,
	}, nil
}

// GetGrammarPointByName retrieves a grammar point by name
func GetGrammarPointByName(ctx context.Context, name string) (*GrammarPoint, error) {
	result, err := queries.GetGrammarPointByName(ctx, name)
	if err == sql.ErrNoRows || err == pgx.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	return &GrammarPoint{
		ID:          int(result.GrammarPointID),
		Name:        result.Name,
		Description: result.Description.String,
	}, nil
}

// ListGrammarPoints returns all grammar points
func ListGrammarPoints(ctx context.Context) ([]GrammarPoint, error) {
	results, err := queries.ListGrammarPoints(ctx)
	if err != nil {
		return nil, err
	}

	grammarPoints := make([]GrammarPoint, 0, len(results))
	for _, result := range results {
		grammarPoints = append(grammarPoints, GrammarPoint{
			ID:          int(result.GrammarPointID),
			Name:        result.Name,
			Description: result.Description.String,
		})
	}

	return grammarPoints, nil
}

// UpdateGrammarPoint updates an existing grammar point
func UpdateGrammarPoint(ctx context.Context, grammarPointID int, name, description string) (*GrammarPoint, error) {
	result, err := queries.UpdateGrammarPoint(ctx, db.UpdateGrammarPointParams{
		GrammarPointID: int32(grammarPointID),
		Name:           name,
		Description:    pgtype.Text{String: description, Valid: description != ""},
	})
	if err == sql.ErrNoRows || err == pgx.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	return &GrammarPoint{
		ID:          int(result.GrammarPointID),
		Name:        result.Name,
		Description: result.Description.String,
	}, nil
}

// DeleteGrammarPoint deletes a grammar point
func DeleteGrammarPoint(ctx context.Context, grammarPointID int) error {
	err := queries.DeleteGrammarPoint(ctx, int32(grammarPointID))
	if err == sql.ErrNoRows || err == pgx.ErrNoRows {
		return ErrNotFound
	}
	return err
}

// AddGrammarPointToStory adds a grammar point to a story
func AddGrammarPointToStory(ctx context.Context, storyID, grammarPointID int) error {
	return queries.AddGrammarPointToStory(ctx, db.AddGrammarPointToStoryParams{
		StoryID:        int32(storyID),
		GrammarPointID: int32(grammarPointID),
	})
}

// RemoveGrammarPointFromStory removes a grammar point from a story
func RemoveGrammarPointFromStory(ctx context.Context, storyID, grammarPointID int) error {
	return queries.RemoveGrammarPointFromStory(ctx, db.RemoveGrammarPointFromStoryParams{
		StoryID:        int32(storyID),
		GrammarPointID: int32(grammarPointID),
	})
}

// GetStoryGrammarPoints returns all grammar points for a story
func GetStoryGrammarPoints(ctx context.Context, storyID int) ([]GrammarPoint, error) {
	results, err := queries.GetStoryGrammarPoints(ctx, int32(storyID))
	if err != nil {
		return nil, err
	}

	grammarPoints := make([]GrammarPoint, 0, len(results))
	for _, result := range results {
		grammarPoints = append(grammarPoints, GrammarPoint{
			ID:          int(result.GrammarPointID),
			Name:        result.Name,
			Description: result.Description.String,
		})
	}

	return grammarPoints, nil
}

// GetStoriesWithGrammarPoint returns all stories that use a specific grammar point
func GetStoriesWithGrammarPoint(ctx context.Context, grammarPointID int) ([]Story, error) {
	results, err := queries.GetStoriesWithGrammarPoint(ctx, int32(grammarPointID))
	if err != nil {
		return nil, err
	}

	stories := make([]Story, 0, len(results))
	for _, result := range results {
		story := Story{
			Metadata: StoryMetadata{
				StoryID:    int(result.StoryID),
				WeekNumber: int(result.WeekNumber),
				DayLetter:  result.DayLetter,
				VideoURL:   result.VideoUrl.String,
				Author: Author{
					ID:   result.AuthorID,
					Name: result.AuthorName,
				},
			},
		}
		if result.LastRevision.Valid {
			story.Metadata.LastRevision = &result.LastRevision.Time
		}
		if result.CourseID.Valid {
			courseID := int(result.CourseID.Int32)
			story.Metadata.CourseID = &courseID
		}
		stories = append(stories, story)
	}

	return stories, nil
}

// ClearStoryGrammarPoints removes all grammar points from a story
func ClearStoryGrammarPoints(ctx context.Context, storyID int) error {
	return queries.ClearStoryGrammarPoints(ctx, int32(storyID))
}
