package models

import (
	"context"
	"database/sql"
	"errors"

	"glossias/src/pkg/generated/db"
)

// LineTranslation represents a translation of a story line
type LineTranslation struct {
	StoryID         int32  `json:"storyId"`
	LineNumber      int32  `json:"lineNumber"`
	LanguageCode    string `json:"languageCode"`
	TranslationText string `json:"translationText"`
}

// GetLineTranslation retrieves a single translation for a specific line and language
func GetLineTranslation(ctx context.Context, storyID, lineNumber int, languageCode string) (string, error) {
	if queries == nil {
		return "", errors.New("database not initialized")
	}

	translation, err := queries.GetLineTranslation(ctx, db.GetLineTranslationParams{
		StoryID:      int32(storyID),
		LineNumber:   int32(lineNumber),
		LanguageCode: languageCode,
	})

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", ErrNotFound
		}
		return "", err
	}

	return translation, nil
}

// UpsertLineTranslation creates or updates a translation for a specific line
func UpsertLineTranslation(ctx context.Context, storyID, lineNumber int, languageCode, translationText string) error {
	if queries == nil {
		return errors.New("database not initialized")
	}

	err := queries.UpsertLineTranslation(ctx, db.UpsertLineTranslationParams{
		StoryID:         int32(storyID),
		LineNumber:      int32(lineNumber),
		LanguageCode:    languageCode,
		TranslationText: translationText,
	})
	return err
}

// GetAllTranslationsForStory retrieves all translations for a story
func GetAllTranslationsForStory(ctx context.Context, storyID int) ([]LineTranslation, error) {
	if queries == nil {
		return nil, errors.New("database not initialized")
	}

	rows, err := queries.GetAllTranslationsForStory(ctx, int32(storyID))
	if err != nil {
		return nil, err
	}

	translations := make([]LineTranslation, len(rows))
	for i, row := range rows {
		translations[i] = LineTranslation{
			StoryID:         row.StoryID,
			LineNumber:      row.LineNumber,
			LanguageCode:    row.LanguageCode,
			TranslationText: row.TranslationText,
		}
	}

	return translations, nil
}

// GetTranslationsByLanguage retrieves all translations for a story in a specific language
func GetTranslationsByLanguage(ctx context.Context, storyID int, languageCode string) ([]LineTranslation, error) {
	if queries == nil {
		return nil, errors.New("database not initialized")
	}

	rows, err := queries.GetTranslationsByLanguage(ctx, db.GetTranslationsByLanguageParams{
		StoryID:      int32(storyID),
		LanguageCode: languageCode,
	})
	if err != nil {
		return nil, err
	}

	translations := make([]LineTranslation, len(rows))
	for i, row := range rows {
		translations[i] = LineTranslation{
			StoryID:         int32(storyID),
			LineNumber:      row.LineNumber,
			LanguageCode:    languageCode,
			TranslationText: row.TranslationText,
		}
	}

	return translations, nil
}

// DeleteLineTranslation removes a specific translation
func DeleteLineTranslation(ctx context.Context, storyID, lineNumber int, languageCode string) error {
	if queries == nil {
		return errors.New("database not initialized")
	}

	return queries.DeleteLineTranslation(ctx, db.DeleteLineTranslationParams{
		StoryID:      int32(storyID),
		LineNumber:   int32(lineNumber),
		LanguageCode: languageCode,
	})
}

// DeleteStoryTranslations removes all translations for a story
func DeleteStoryTranslations(ctx context.Context, storyID int) error {
	if queries == nil {
		return errors.New("database not initialized")
	}

	return queries.DeleteAllTranslationsForStory(ctx, int32(storyID))
}
