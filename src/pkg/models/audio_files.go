package models

import (
	"context"
	"database/sql"
	"errors"

	"glossias/src/pkg/generated/db"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

// CreateAudioFile creates a new audio file record
func CreateAudioFile(ctx context.Context, storyID, lineNumber int, filePath, fileBucket, label string) (*AudioFile, error) {
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

// DeleteAudioFile deletes an audio file
func DeleteAudioFile(ctx context.Context, audioFileID int) error {
	err := queries.DeleteAudioFile(ctx, int32(audioFileID))
	if err == sql.ErrNoRows || err == pgx.ErrNoRows {
		return ErrNotFound
	}
	return err
}

// DeleteLineAudioFiles deletes all audio files for a specific line
func DeleteLineAudioFiles(ctx context.Context, storyID, lineNumber int) error {
	return queries.DeleteLineAudioFiles(ctx, db.DeleteLineAudioFilesParams{
		StoryID:    pgtype.Int4{Int32: int32(storyID), Valid: true},
		LineNumber: pgtype.Int4{Int32: int32(lineNumber), Valid: true},
	})
}

// DeleteStoryAudioFiles deletes all audio files for a story
func DeleteStoryAudioFiles(ctx context.Context, storyID int) error {
	return queries.DeleteStoryAudioFiles(ctx, pgtype.Int4{Int32: int32(storyID), Valid: true})
}

// DeleteStoryAudioFilesByLabel deletes all audio files for a story with a specific label
func DeleteStoryAudioFilesByLabel(ctx context.Context, storyID int, label string) error {
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

	// Generate signed URL from Supabase
	result, err := storageClient.CreateSignedUrl(audioFile.FileBucket, audioFile.FilePath, expiresInSeconds)
	if err != nil {
		return "", err
	}

	return result.SignedURL, nil
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

	// Generate signed URLs
	signedURLs := make(map[int]string)
	for _, audioFile := range audioFiles {
		result, err := storageClient.CreateSignedUrl(audioFile.FileBucket, audioFile.FilePath, expiresInSeconds)
		if err != nil {
			return nil, err
		}
		signedURLs[audioFile.ID] = result.SignedURL
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

	// Generate signed URLs
	signedURLs := make(map[int]string)
	for _, audioFile := range audioFiles {
		result, err := storageClient.CreateSignedUrl(audioFile.FileBucket, audioFile.FilePath, expiresInSeconds)
		if err != nil {
			return nil, err
		}
		signedURLs[audioFile.ID] = result.SignedURL
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

	// Generate signed upload URL
	result, err := storageClient.CreateSignedUploadUrl(bucket, filePath)
	if err != nil {
		return "", err
	}

	// Construct complete URL using base storage URL
	if storageBaseURL == "" {
		return "", errors.New("storage base URL not configured")
	}

	return storageBaseURL + "/storage/v1/s3" + result.Url, nil
}
