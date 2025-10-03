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

var ErrAudioFileExists = errors.New("audio file with this label already exists for this line")

// CreateAudioFile creates a new audio file record
func CreateAudioFile(ctx context.Context, storyID, lineNumber int, filePath, fileBucket, label string) (*AudioFile, error) {
	// Check if audio file already exists for this line and label
	existingAudioFiles, err := GetLineAudioFiles(ctx, storyID, lineNumber)
	if err != nil {
		return nil, err
	}
	for _, audioFile := range existingAudioFiles {
		if audioFile.Label == label {
			return nil, ErrAudioFileExists
		}
	}

	result, err := queries.CreateAudioFile(ctx, db.CreateAudioFileParams{
		StoryID:    pgtype.Int4{Int32: int32(storyID), Valid: true},
		LineNumber: pgtype.Int4{Int32: int32(lineNumber), Valid: true},
		FilePath:   filePath,
		FileBucket: fileBucket,
		Label:      label,
	})
	if err != nil {
		return nil, err
	}

	return &AudioFile{
		ID:         int(result.AudioFileID),
		StoryID:    int(result.StoryID.Int32),
		LineNumber: int(result.LineNumber.Int32),
		FilePath:   result.FilePath,
		FileBucket: result.FileBucket,
		Label:      result.Label,
	}, nil
}

// GetAudioFile retrieves an audio file by ID
func GetAudioFile(ctx context.Context, audioFileID int) (*AudioFile, error) {
	result, err := queries.GetAudioFile(ctx, int32(audioFileID))
	if err == sql.ErrNoRows || err == pgx.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	return &AudioFile{
		ID:         int(result.AudioFileID),
		StoryID:    int(result.StoryID.Int32),
		LineNumber: int(result.LineNumber.Int32),
		FilePath:   result.FilePath,
		FileBucket: result.FileBucket,
		Label:      result.Label,
	}, nil
}

// GetLineAudioFiles retrieves all audio files for a specific line
func GetLineAudioFiles(ctx context.Context, storyID, lineNumber int) ([]AudioFile, error) {
	results, err := queries.GetLineAudioFiles(ctx, db.GetLineAudioFilesParams{
		StoryID:    pgtype.Int4{Int32: int32(storyID), Valid: true},
		LineNumber: pgtype.Int4{Int32: int32(lineNumber), Valid: true},
	})
	if err != nil {
		return nil, err
	}

	audioFiles := make([]AudioFile, 0, len(results))
	for _, result := range results {
		audioFiles = append(audioFiles, AudioFile{
			ID:         int(result.AudioFileID),
			StoryID:    int(result.StoryID.Int32),
			LineNumber: int(result.LineNumber.Int32),
			FilePath:   result.FilePath,
			FileBucket: result.FileBucket,
			Label:      result.Label,
		})
	}

	return audioFiles, nil
}

// GetStoryAudioFilesByLabel retrieves all audio files for a story with a specific label
func GetStoryAudioFilesByLabel(ctx context.Context, storyID int, label string) ([]AudioFile, error) {
	results, err := queries.GetStoryAudioFilesByLabel(ctx, db.GetStoryAudioFilesByLabelParams{
		StoryID: pgtype.Int4{Int32: int32(storyID), Valid: true},
		Label:   label,
	})
	if err != nil {
		return nil, err
	}

	audioFiles := make([]AudioFile, 0, len(results))
	for _, result := range results {
		audioFiles = append(audioFiles, AudioFile{
			ID:         int(result.AudioFileID),
			StoryID:    int(result.StoryID.Int32),
			LineNumber: int(result.LineNumber.Int32),
			FilePath:   result.FilePath,
			FileBucket: result.FileBucket,
			Label:      result.Label,
		})
	}

	return audioFiles, nil
}

// GetAllStoryAudioFiles retrieves all audio files for a story
func GetAllStoryAudioFiles(ctx context.Context, storyID int) ([]AudioFile, error) {
	results, err := queries.GetAllStoryAudioFiles(ctx, pgtype.Int4{Int32: int32(storyID), Valid: true})
	if err != nil {
		return nil, err
	}

	audioFiles := make([]AudioFile, 0, len(results))
	for _, result := range results {
		audioFiles = append(audioFiles, AudioFile{
			ID:         int(result.AudioFileID),
			StoryID:    int(result.StoryID.Int32),
			LineNumber: int(result.LineNumber.Int32),
			FilePath:   result.FilePath,
			FileBucket: result.FileBucket,
			Label:      result.Label,
		})
	}

	return audioFiles, nil
}

// UpdateAudioFile updates an existing audio file
func UpdateAudioFile(ctx context.Context, audioFileID int, storyID int, filePath, fileBucket, label string) (*AudioFile, error) {
	result, err := queries.UpdateAudioFile(ctx, db.UpdateAudioFileParams{
		AudioFileID: int32(audioFileID),
		StoryID:     pgtype.Int4{Int32: int32(storyID), Valid: true},
		FilePath:    filePath,
		FileBucket:  fileBucket,
		Label:       label,
	})
	if err == sql.ErrNoRows || err == pgx.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	return &AudioFile{
		ID:         int(result.AudioFileID),
		StoryID:    int(result.StoryID.Int32),
		LineNumber: int(result.LineNumber.Int32),
		FilePath:   result.FilePath,
		FileBucket: result.FileBucket,
		Label:      result.Label,
	}, nil
}

// deleteAudioFilesFromStorage deletes audio files from Supabase storage
func deleteAudioFilesFromStorage(audioFiles []AudioFile) error {
	if storageClient == nil {
		return errors.New("storage client not initialized")
	}

	for _, audioFile := range audioFiles {
		err := storageRetry(func() error {
			_, removeErr := storageClient.RemoveFile(audioFile.FileBucket, []string{audioFile.FilePath})
			return removeErr
		})
		if err != nil {
			return fmt.Errorf("failed to delete file from storage: %w", err)
		}
	}
	return nil
}

// DeleteAudioFile deletes an audio file
func DeleteAudioFile(ctx context.Context, audioFileID int) error {
	// Get audio file details before deletion
	audioFile, err := GetAudioFile(ctx, audioFileID)
	if err != nil {
		return err
	}

	// Delete from Supabase storage first
	if err := deleteAudioFilesFromStorage([]AudioFile{*audioFile}); err != nil {
		return err
	}

	// Delete from database
	err = queries.DeleteAudioFile(ctx, int32(audioFileID))
	if err == sql.ErrNoRows || err == pgx.ErrNoRows {
		return ErrNotFound
	}
	return err
}

// DeleteLineAudioFiles deletes all audio files for a specific line
func DeleteLineAudioFiles(ctx context.Context, storyID, lineNumber int) error {
	// Get all audio files for the line before deletion
	audioFiles, err := GetLineAudioFiles(ctx, storyID, lineNumber)
	if err != nil {
		return err
	}

	// Delete from Supabase storage first
	if err := deleteAudioFilesFromStorage(audioFiles); err != nil {
		return err
	}

	// Delete from database
	return queries.DeleteLineAudioFiles(ctx, db.DeleteLineAudioFilesParams{
		StoryID:    pgtype.Int4{Int32: int32(storyID), Valid: true},
		LineNumber: pgtype.Int4{Int32: int32(lineNumber), Valid: true},
	})
}

// DeleteStoryAudioFiles deletes all audio files for a story
func DeleteStoryAudioFiles(ctx context.Context, storyID int) error {
	// Get all audio files for the story before deletion
	audioFiles, err := GetAllStoryAudioFiles(ctx, storyID)
	if err != nil {
		return err
	}

	// Delete from Supabase storage first
	if err := deleteAudioFilesFromStorage(audioFiles); err != nil {
		return err
	}

	// Delete from database
	return queries.DeleteStoryAudioFiles(ctx, pgtype.Int4{Int32: int32(storyID), Valid: true})
}

// DeleteStoryAudioFilesByLabel deletes all audio files for a story with a specific label
func DeleteStoryAudioFilesByLabel(ctx context.Context, storyID int, label string) error {
	// Get all audio files for the story with the label before deletion
	audioFiles, err := GetStoryAudioFilesByLabel(ctx, storyID, label)
	if err != nil {
		return err
	}

	// Delete from Supabase storage first
	if err := deleteAudioFilesFromStorage(audioFiles); err != nil {
		return err
	}

	// Delete from database
	return queries.DeleteStoryAudioFilesByLabel(ctx, db.DeleteStoryAudioFilesByLabelParams{
		StoryID: pgtype.Int4{Int32: int32(storyID), Valid: true},
		Label:   label,
	})
}

// GetAudioFilesByLabel returns all audio files with a specific label across all stories
func GetAudioFilesByLabel(ctx context.Context, label string) ([]AudioFile, error) {
	results, err := queries.GetAudioFilesByLabel(ctx, label)
	if err != nil {
		return nil, err
	}

	audioFiles := make([]AudioFile, 0, len(results))
	for _, result := range results {
		audioFiles = append(audioFiles, AudioFile{
			ID:         int(result.AudioFileID),
			StoryID:    int(result.StoryID.Int32),
			LineNumber: int(result.LineNumber.Int32),
			FilePath:   result.FilePath,
			FileBucket: result.FileBucket,
			Label:      result.Label,
		})
	}

	return audioFiles, nil
}

// GetSignedAudioURL generates a signed URL for a specific audio file
func GetSignedAudioURL(ctx context.Context, audioFileID int, userID string, expiresInSeconds int) (string, error) {
	if storageClient == nil {
		return "", errors.New("storage client not initialized")
	}

	// Get audio file record
	audioFile, err := GetAudioFile(ctx, audioFileID)
	if err != nil {
		return "", err
	}

	// Check user can access this story's course
	story, err := queries.GetStory(ctx, int32(audioFile.StoryID))
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
		result, signErr := storageClient.CreateSignedUrl(audioFile.FileBucket, audioFile.FilePath, expiresInSeconds)
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

// GetSignedAudioURLsForStory generates signed URLs for all audio files in a story with optional label filter
func GetSignedAudioURLsForStory(ctx context.Context, storyID int, userID string, label string, expiresInSeconds int) (map[int]string, error) {
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

	// Get audio files
	var audioFiles []AudioFile
	if label != "" {
		audioFiles, err = GetStoryAudioFilesByLabel(ctx, storyID, label)
	} else {
		audioFiles, err = GetAllStoryAudioFiles(ctx, storyID)
	}
	if err != nil {
		return nil, err
	}

	// Generate signed URLs with retry
	signedURLs := make(map[int]string)
	for _, audioFile := range audioFiles {
		var signedURL string
		err := storageRetry(func() error {
			result, signErr := storageClient.CreateSignedUrl(audioFile.FileBucket, audioFile.FilePath, expiresInSeconds)
			if signErr == nil {
				signedURL = result.SignedURL
			}
			return signErr
		})
		if err != nil {
			return nil, err
		}
		signedURLs[audioFile.LineNumber] = signedURL
	}

	return signedURLs, nil
}

// GetSignedAudioURLsForLine generates signed URLs for all audio files on a specific line
func GetSignedAudioURLsForLine(ctx context.Context, storyID, lineNumber int, userID string, expiresInSeconds int) (map[int]string, error) {
	if storageClient == nil {
		return nil, errors.New("storage client not initialized")
	}

	// Check user can access this story's course
	story, err := queries.GetStory(ctx, int32(storyID))
	if err != nil {
		return nil, err
	}

	if story.CourseID.Valid {
		canAccess := CanUserAccessCourse(ctx, userID, story.CourseID.Int32)
		if !canAccess {
			return nil, errors.New("access denied")
		}
	}

	// Get audio files for the line
	audioFiles, err := GetLineAudioFiles(ctx, storyID, lineNumber)
	if err != nil {
		return nil, err
	}

	// Generate signed URLs with retry
	signedURLs := make(map[int]string)
	for _, audioFile := range audioFiles {
		var signedURL string
		err := storageRetry(func() error {
			result, signErr := storageClient.CreateSignedUrl(audioFile.FileBucket, audioFile.FilePath, expiresInSeconds)
			if signErr == nil {
				signedURL = result.SignedURL
			}
			return signErr
		})
		if err != nil {
			return nil, err
		}
		signedURLs[audioFile.ID] = signedURL
	}

	return signedURLs, nil
}

// StoryExists checks if a story exists in the database
func StoryExists(ctx context.Context, storyID int32) (bool, error) {
	return queries.StoryExists(ctx, storyID)
}

// LineExists checks if a specific line exists in a story
func LineExists(ctx context.Context, storyID, lineNumber int) (bool, error) {
	return queries.LineExists(ctx, db.LineExistsParams{
		StoryID:    int32(storyID),
		LineNumber: int32(lineNumber),
	})
}

// GenerateSignedUploadURL creates a signed URL for uploading files to Supabase storage
func GenerateSignedUploadURL(ctx context.Context, bucket, filePath string) (string, error) {
	if storageClient == nil {
		return "", errors.New("storage client not initialized")
	}

	// Generate signed upload URL with retry
	var uploadURL string
	err := storageRetry(func() error {
		result, signErr := storageClient.CreateSignedUploadUrl(bucket, filePath)
		if signErr == nil {
			uploadURL = storageBaseURL + result.Url
		}
		return signErr
	})
	if err != nil {
		return "", err
	}

	return uploadURL, nil
}
