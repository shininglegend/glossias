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

// StoryImage represents an image file attached to a story
type StoryImage struct {
	ID         int    `json:"id"`
	StoryID    int    `json:"storyId"`
	FilePath   string `json:"filePath"`
	FileBucket string `json:"fileBucket"`
	Label      string `json:"label"`
}

// CreateStoryImage creates a new story image record
func CreateStoryImage(ctx context.Context, storyID int, filePath, fileBucket, label string) (*StoryImage, error) {
	result, err := queries.CreateStoryImage(ctx, db.CreateStoryImageParams{
		StoryID:    pgtype.Int4{Int32: int32(storyID), Valid: true},
		FilePath:   filePath,
		FileBucket: fileBucket,
		Label:      label,
	})
	if err != nil {
		return nil, err
	}

	return &StoryImage{
		ID:         int(result.ImageID),
		StoryID:    int(result.StoryID.Int32),
		FilePath:   result.FilePath,
		FileBucket: result.FileBucket,
		Label:      result.Label,
	}, nil
}

// GetStoryImage retrieves a story image by ID
func GetStoryImage(ctx context.Context, imageID int) (*StoryImage, error) {
	result, err := queries.GetStoryImage(ctx, int32(imageID))
	if err == sql.ErrNoRows || err == pgx.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	return &StoryImage{
		ID:         int(result.ImageID),
		StoryID:    int(result.StoryID.Int32),
		FilePath:   result.FilePath,
		FileBucket: result.FileBucket,
		Label:      result.Label,
	}, nil
}

// GetStoryImagesByLabel retrieves all images for a story with a specific label
func GetStoryImagesByLabel(ctx context.Context, storyID int, label string) ([]StoryImage, error) {
	results, err := queries.GetStoryImagesByLabel(ctx, db.GetStoryImagesByLabelParams{
		StoryID: pgtype.Int4{Int32: int32(storyID), Valid: true},
		Label:   label,
	})
	if err != nil {
		return nil, err
	}

	images := make([]StoryImage, 0, len(results))
	for _, result := range results {
		images = append(images, StoryImage{
			ID:         int(result.ImageID),
			StoryID:    int(result.StoryID.Int32),
			FilePath:   result.FilePath,
			FileBucket: result.FileBucket,
			Label:      result.Label,
		})
	}

	return images, nil
}

// GetAllStoryImages retrieves all images for a story
func GetAllStoryImages(ctx context.Context, storyID int) ([]StoryImage, error) {
	results, err := queries.GetAllStoryImages(ctx, pgtype.Int4{Int32: int32(storyID), Valid: true})
	if err != nil {
		return nil, err
	}

	images := make([]StoryImage, 0, len(results))
	for _, result := range results {
		images = append(images, StoryImage{
			ID:         int(result.ImageID),
			StoryID:    int(result.StoryID.Int32),
			FilePath:   result.FilePath,
			FileBucket: result.FileBucket,
			Label:      result.Label,
		})
	}

	return images, nil
}

// UpdateStoryImage updates an existing story image
func UpdateStoryImage(ctx context.Context, imageID int, storyID int, filePath, fileBucket, label string) (*StoryImage, error) {
	result, err := queries.UpdateStoryImage(ctx, db.UpdateStoryImageParams{
		ImageID:    int32(imageID),
		StoryID:    pgtype.Int4{Int32: int32(storyID), Valid: true},
		FilePath:   filePath,
		FileBucket: fileBucket,
		Label:      label,
	})
	if err == sql.ErrNoRows || err == pgx.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	return &StoryImage{
		ID:         int(result.ImageID),
		StoryID:    int(result.StoryID.Int32),
		FilePath:   result.FilePath,
		FileBucket: result.FileBucket,
		Label:      result.Label,
	}, nil
}

// deleteStoryImagesFromStorage deletes image files from Supabase storage
func deleteStoryImagesFromStorage(storyImages []StoryImage) error {
	if storageClient == nil {
		return errors.New("storage client not initialized")
	}

	for _, img := range storyImages {
		err := storageRetry(func() error {
			_, removeErr := storageClient.RemoveFile(img.FileBucket, []string{img.FilePath})
			return removeErr
		})
		if err != nil {
			return fmt.Errorf("failed to delete file from storage: %w", err)
		}
	}
	return nil
}

// DeleteStoryImage deletes a story image
func DeleteStoryImage(ctx context.Context, imageID int) error {
	// Get image details before deletion
	img, err := GetStoryImage(ctx, imageID)
	if err != nil {
		return err
	}

	// Delete from Supabase storage first
	if err := deleteStoryImagesFromStorage([]StoryImage{*img}); err != nil {
		return err
	}

	// Delete from database
	err = queries.DeleteStoryImage(ctx, int32(imageID))
	if err == sql.ErrNoRows || err == pgx.ErrNoRows {
		return ErrNotFound
	}
	return err
}

// DeleteStoryImages deletes all images for a story
func DeleteStoryImages(ctx context.Context, storyID int) error {
	// Get all images for the story before deletion
	images, err := GetAllStoryImages(ctx, storyID)
	if err != nil {
		return err
	}

	// Delete from Supabase storage first
	if err := deleteStoryImagesFromStorage(images); err != nil {
		return err
	}

	// Delete from database
	return queries.DeleteStoryImages(ctx, pgtype.Int4{Int32: int32(storyID), Valid: true})
}

// DeleteStoryImagesByLabel deletes all images for a story with a specific label
func DeleteStoryImagesByLabel(ctx context.Context, storyID int, label string) error {
	// Get all images for the story with the label before deletion
	images, err := GetStoryImagesByLabel(ctx, storyID, label)
	if err != nil {
		return err
	}

	// Delete from Supabase storage first
	if err := deleteStoryImagesFromStorage(images); err != nil {
		return err
	}

	// Delete from database
	return queries.DeleteStoryImagesByLabel(ctx, db.DeleteStoryImagesByLabelParams{
		StoryID: pgtype.Int4{Int32: int32(storyID), Valid: true},
		Label:   label,
	})
}

// GetStoryImagesByLabelGlobally returns all images with a specific label across all stories
func GetStoryImagesByLabelGlobally(ctx context.Context, label string) ([]StoryImage, error) {
	results, err := queries.GetStoryImagesByLabelGlobally(ctx, label)
	if err != nil {
		return nil, err
	}

	images := make([]StoryImage, 0, len(results))
	for _, result := range results {
		images = append(images, StoryImage{
			ID:         int(result.ImageID),
			StoryID:    int(result.StoryID.Int32),
			FilePath:   result.FilePath,
			FileBucket: result.FileBucket,
			Label:      result.Label,
		})
	}

	return images, nil
}

// GetSignedImageURL generates a signed URL for a specific image file
func GetSignedImageURL(ctx context.Context, imageID int, userID string, expiresInSeconds int) (string, error) {
	if storageClient == nil {
		return "", errors.New("storage client not initialized")
	}

	// Get image record
	img, err := GetStoryImage(ctx, imageID)
	if err != nil {
		return "", err
	}

	// Check user can access this story's course
	story, err := queries.GetStory(ctx, int32(img.StoryID))
	if err != nil {
		return "", err
	}

	if story.CourseID.Valid {
		canAccess := CanUserAccessCourse(ctx, userID, story.CourseID.Int32)
		if !canAccess {
			return "", errors.New("access denied")
		}
	}

	// Generate signed URL from Supabase with retry
	var signedURL string
	err = storageRetry(func() error {
		result, signErr := storageClient.CreateSignedUrl(img.FileBucket, img.FilePath, expiresInSeconds)
		if signErr == nil {
			signedURL = result.SignedURL
		}
		return signErr
	})
	if err != nil {
		return "", err
	}

	return signedURL, nil
}

// GetSignedImageURLsForStory generates signed URLs for all images in a story with optional label filter
func GetSignedImageURLsForStory(ctx context.Context, storyID int, userID string, label string, expiresInSeconds int) (map[int]string, error) {
	if storageClient == nil {
		return nil, errors.New("storage client not initialized")
	}

	// Check user can access this story's course
	story, err := queries.GetStory(ctx, int32(storyID))
	if err != nil {
		if err == sql.ErrNoRows || err == pgx.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, err
	}

	if story.CourseID.Valid {
		canAccess := CanUserAccessCourse(ctx, userID, story.CourseID.Int32)
		if !canAccess {
			return nil, errors.New("access denied")
		}
	}

	// Get images
	var images []StoryImage
	if label != "" {
		images, err = GetStoryImagesByLabel(ctx, storyID, label)
	} else {
		images, err = GetAllStoryImages(ctx, storyID)
	}
	if err != nil {
		return nil, err
	}

	// Generate signed URLs with retry
	signedURLs := make(map[int]string)
	for _, img := range images {
		var signedURL string
		err := storageRetry(func() error {
			result, signErr := storageClient.CreateSignedUrl(img.FileBucket, img.FilePath, expiresInSeconds)
			if signErr == nil {
				signedURL = result.SignedURL
			}
			return signErr
		})
		if err != nil {
			return nil, err
		}
		signedURLs[img.ID] = signedURL
	}

	return signedURLs, nil
}
