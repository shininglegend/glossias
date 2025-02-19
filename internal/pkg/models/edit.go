// glossias/internal/pkg/models/edit.go
package models

import (
	"database/sql"
	"errors"
	"time"
)

var (
	ErrInvalidStoryID    = errors.New("invalid story ID")
	ErrInvalidLineNumber = errors.New("invalid line number")
)

// EditStoryText updates only the text content of story lines
func EditStoryText(storyID int, lines []StoryLine) error {
	return withTransaction(func(tx *sql.Tx) error {
		// Delete existing lines
		if _, err := tx.Exec(`DELETE FROM story_lines WHERE story_id = ?`, storyID); err != nil {
			return err
		}

		// Insert updated lines
		for _, line := range lines {
			if _, err := tx.Exec(`
                INSERT INTO story_lines (story_id, line_number, text)
                VALUES (?, ?, ?)`,
				storyID, line.LineNumber, line.Text); err != nil {
				return err
			}
		}

		// Update last revision timestamp
		if _, err := tx.Exec(`
            UPDATE stories
            SET last_revision = datetime('now')
            WHERE story_id = ?`, storyID); err != nil {
			return err
		}

		return nil
	})
}

// EditStoryMetadata updates the story's metadata fields
func EditStoryMetadata(storyID int, metadata StoryMetadata) error {
	return withTransaction(func(tx *sql.Tx) error {
		// Update main story table
		if _, err := tx.Exec(`
            UPDATE stories
            SET week_number = ?,
                day_letter = ?,
                grammar_point = ?,
                author_id = ?,
                author_name = ?,
                last_revision = ?
            WHERE story_id = ?`,
			metadata.WeekNumber,
			metadata.DayLetter,
			metadata.GrammarPoint,
			metadata.Author.ID,
			metadata.Author.Name,
			time.Now().UTC(),
			storyID); err != nil {
			return err
		}

		// Update titles
		if _, err := tx.Exec(`DELETE FROM story_titles WHERE story_id = ?`, storyID); err != nil {
			return err
		}
		for lang, title := range metadata.Title {
			if _, err := tx.Exec(`
                INSERT INTO story_titles (story_id, language_code, title)
                VALUES (?, ?, ?)`,
				storyID, lang, title); err != nil {
				return err
			}
		}

		// Update description
		if _, err := tx.Exec(`DELETE FROM story_descriptions WHERE story_id = ?`, storyID); err != nil {
			return err
		}
		if metadata.Description.Text != "" {
			if _, err := tx.Exec(`
                INSERT INTO story_descriptions (story_id, language_code, description_text)
                VALUES (?, ?, ?)`,
				storyID, metadata.Description.Language, metadata.Description.Text); err != nil {
				return err
			}
		}

		return nil
	})
}

// AddLineAnnotations updates grammar points, vocabulary, and footnotes for a specific line
// It adds new annotations only - existing annotations are not removed
func AddLineAnnotations(storyID int, lineNumber int, line StoryLine) error {
	return withTransaction(func(tx *sql.Tx) error {
		// Verify line exists
		var exists int
		err := tx.QueryRow(`
            SELECT EXISTS(
                SELECT 1 FROM story_lines
                WHERE story_id = ? AND line_number = ?
            )`, storyID, lineNumber).Scan(&exists)
		if err != nil {
			return err
		}
		if exists == 0 {
			return ErrInvalidLineNumber
		}

		// // Delete existing annotations
		// for _, query := range []string{
		// 	`DELETE FROM vocabulary_items WHERE story_id = ? AND line_number = ?`,
		// 	`DELETE FROM grammar_items WHERE story_id = ? AND line_number = ?`,
		// 	`DELETE FROM footnotes WHERE story_id = ? AND line_number = ?`,
		// } {
		// 	if _, err := tx.Exec(query, storyID, lineNumber); err != nil {
		// 		return err
		// 	}
		// }

		// Insert vocabulary items
		for _, v := range line.Vocabulary {
			if _, err := tx.Exec(`
                INSERT INTO vocabulary_items (
                    story_id, line_number, word, lexical_form,
                    position_start, position_end
                ) VALUES (?, ?, ?, ?, ?, ?)`,
				storyID, lineNumber, v.Word, v.LexicalForm,
				v.Position[0], v.Position[1]); err != nil {
				return err
			}
		}

		// Insert grammar items
		for _, g := range line.Grammar {
			if _, err := tx.Exec(`
                INSERT INTO grammar_items (
                    story_id, line_number, text,
                    position_start, position_end
                ) VALUES (?, ?, ?, ?, ?)`,
				storyID, lineNumber, g.Text,
				g.Position[0], g.Position[1]); err != nil {
				return err
			}
		}

		// Insert footnotes and their references
		for _, f := range line.Footnotes {
			result, err := tx.Exec(`
                INSERT INTO footnotes (story_id, line_number, footnote_text)
                VALUES (?, ?, ?)`,
				storyID, lineNumber, f.Text)
			if err != nil {
				return err
			}

			footnoteID, err := result.LastInsertId()
			if err != nil {
				return err
			}

			// Insert references for this footnote
			for _, ref := range f.References {
				if _, err := tx.Exec(`
                    INSERT INTO footnote_references (footnote_id, reference)
                    VALUES (?, ?)`,
					footnoteID, ref); err != nil {
					return err
				}
			}
		}

		// Update last revision timestamp
		if _, err := tx.Exec(`
            UPDATE stories
            SET last_revision = datetime('now')
            WHERE story_id = ?`, storyID); err != nil {
			return err
		}

		return nil
	})
}

// ClearStoryAnnotations removes all annotations from a story while preserving the text and metadata
func ClearStoryAnnotations(storyID int) error {
	// Verify story exists first
	exists, err := storyExists(storyID)
	if err != nil {
		return err
	}
	if !exists {
		return ErrNotFound
	}

	return withTransaction(func(tx *sql.Tx) error {
		// Delete all annotations in the correct order to avoid foreign key constraints
		for _, query := range []string{
			`DELETE FROM footnote_references WHERE footnote_id IN
                (SELECT id FROM footnotes WHERE story_id = ?)`,
			`DELETE FROM footnotes WHERE story_id = ?`,
			`DELETE FROM vocabulary_items WHERE story_id = ?`,
			`DELETE FROM grammar_items WHERE story_id = ?`,
		} {
			if _, err := tx.Exec(query, storyID); err != nil {
				return err
			}
		}

		// Update last revision timestamp
		_, err := tx.Exec(`
            UPDATE stories
            SET last_revision = datetime('now')
            WHERE story_id = ?`,
			storyID)
		return err
	})
}
