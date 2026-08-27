package models

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgconn"
)

// ErrDuplicate reports a unique-constraint violation, so HTTP handlers can
// answer 409 without knowing Postgres error codes.
var ErrDuplicate = errors.New("record already exists")

// asDuplicate maps a Postgres unique-violation into ErrDuplicate, keeping the
// original error available to errors.Is/As. Other errors pass through unchanged.
func asDuplicate(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return fmt.Errorf("%w: %v", ErrDuplicate, err)
	}
	return err
}

// Assets for the Summer 2026 phases are addressed by the bucket path stored on
// the row that owns them — target_vocabulary.correct_image_path,
// target_vocabulary.audio_path, recall_sentences.image_path — rather than by a
// story_images / line_audio_files row ID. The helpers here sign and delete
// those paths directly. Authorization is the caller's job: admin handlers gate
// on CanUserEditStory, student handlers on CanUserAccessCourse.

// GetSignedURLForPath signs a single bucket path for reading.
func GetSignedURLForPath(ctx context.Context, bucket, path string, expiresInSeconds int) (string, error) {
	if storageClient == nil {
		return "", errors.New("storage client not initialized")
	}
	if bucket == "" || path == "" {
		return "", errors.New("bucket and path are required")
	}

	var signedURL string
	err := storageRetry(func() error {
		result, signErr := storageClient.CreateSignedUrl(bucket, path, expiresInSeconds)
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

// DeleteStorageObject removes a single file from a bucket. It is used when an
// asset is replaced or its owning row is deleted, so superseded uploads do not
// accumulate in storage.
func DeleteStorageObject(ctx context.Context, bucket, path string) error {
	if storageClient == nil {
		return errors.New("storage client not initialized")
	}
	if bucket == "" || path == "" {
		return nil
	}

	err := storageRetry(func() error {
		_, removeErr := storageClient.RemoveFile(bucket, []string{path})
		return removeErr
	})
	if err != nil {
		return fmt.Errorf("failed to delete %s/%s from storage: %w", bucket, path, err)
	}
	return nil
}

// SignTargetVocabularyURLs fills AudioURL and ImageURL on each word that has the
// corresponding asset. Words whose assets are not yet uploaded are left alone,
// so a partially authored story still renders.
func SignTargetVocabularyURLs(ctx context.Context, words []TargetVocabulary, expiresInSeconds int) error {
	for i := range words {
		if words[i].AudioPath != "" && words[i].AudioBucket != "" {
			url, err := GetSignedURLForPath(ctx, words[i].AudioBucket, words[i].AudioPath, expiresInSeconds)
			if err != nil {
				return err
			}
			words[i].AudioURL = url
		}
		if words[i].CorrectImagePath != "" && words[i].ImageBucket != "" {
			url, err := GetSignedURLForPath(ctx, words[i].ImageBucket, words[i].CorrectImagePath, expiresInSeconds)
			if err != nil {
				return err
			}
			words[i].ImageURL = url
		}
	}
	return nil
}

// SignRecallSentenceURLs fills ImageURL on each sentence that has an image.
func SignRecallSentenceURLs(ctx context.Context, sentences []RecallSentence, expiresInSeconds int) error {
	for i := range sentences {
		if sentences[i].ImagePath == "" || sentences[i].ImageBucket == "" {
			continue
		}
		url, err := GetSignedURLForPath(ctx, sentences[i].ImageBucket, sentences[i].ImagePath, expiresInSeconds)
		if err != nil {
			return err
		}
		sentences[i].ImageURL = url
	}
	return nil
}
