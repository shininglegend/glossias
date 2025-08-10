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
		if _, err := tx.Exec(`DELETE FROM story_lines WHERE story_id = $1`, storyID); err != nil {
			return err
		}

		// Insert updated lines
		for _, line := range lines {
			if _, err := tx.Exec(`
                INSERT INTO story_lines (story_id, line_number, text)
                VALUES ($1, $2, $3)`,
				storyID, line.LineNumber, line.Text); err != nil {
				return err
			}
		}

		// Update last revision timestamp
		if _, err := tx.Exec(`
            UPDATE stories
            SET last_revision = CURRENT_TIMESTAMP
            WHERE story_id = $1`, storyID); err != nil {
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
            SET week_number = $1,
                day_letter = $2,
                grammar_point = $3,
                author_id = $4,
                author_name = $5,
                last_revision = $6
            WHERE story_id = $7`,
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
		if _, err := tx.Exec(`DELETE FROM story_titles WHERE story_id = $1`, storyID); err != nil {
			return err
		}
		for lang, title := range metadata.Title {
			if _, err := tx.Exec(`
                INSERT INTO story_titles (story_id, language_code, title)
                VALUES ($1, $2, $3)`,
				storyID, lang, title); err != nil {
				return err
			}
		}

		// Update description
		if _, err := tx.Exec(`DELETE FROM story_descriptions WHERE story_id = $1`, storyID); err != nil {
			return err
		}

		if _, err := tx.Exec(`
                INSERT INTO story_descriptions (story_id, language_code, description_text)
                VALUES ($1, $2, $3)`,
			storyID, metadata.Description.Language, metadata.Description.Text); err != nil {
			return err
		}

		return nil
	})
}

// AddLineAnnotations updates grammar points, vocabulary, and footnotes for a specific line
func AddLineAnnotations(storyID int, lineNumber int, line StoryLine) error {
	return withTransaction(func(tx *sql.Tx) error {
		// Verify line exists
		var exists bool
		err := tx.QueryRow(`
            SELECT EXISTS(
                SELECT 1 FROM story_lines
                WHERE story_id = $1 AND line_number = $2
            )`, storyID, lineNumber).Scan(&exists)
		if err != nil {
			return err
		}
		if !exists {
			return ErrInvalidLineNumber
		}

		// Insert vocabulary items
		for _, v := range line.Vocabulary {
			if _, err := tx.Exec(`
                INSERT INTO vocabulary_items (
                    story_id, line_number, word, lexical_form,
                    position_start, position_end
                ) VALUES ($1, $2, $3, $4, $5, $6)`,
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
                ) VALUES ($1, $2, $3, $4, $5)`,
				storyID, lineNumber, g.Text,
				g.Position[0], g.Position[1]); err != nil {
				return err
			}
		}

		// Insert footnotes and their references
		for _, f := range line.Footnotes {
			var footnoteID int
			err := tx.QueryRow(`
                INSERT INTO footnotes (story_id, line_number, footnote_text)
                VALUES ($1, $2, $3)
                RETURNING id`,
				storyID, lineNumber, f.Text).Scan(&footnoteID)
			if err != nil {
				return err
			}

			// Insert references for this footnote
			for _, ref := range f.References {
				if _, err := tx.Exec(`
                    INSERT INTO footnote_references (footnote_id, reference)
                    VALUES ($1, $2)`,
					footnoteID, ref); err != nil {
					return err
				}
			}
		}

		// Update last revision timestamp
		if _, err := tx.Exec(`
            UPDATE stories
            SET last_revision = CURRENT_TIMESTAMP
            WHERE story_id = $1`, storyID); err != nil {
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
		queries := []string{
			`DELETE FROM footnote_references WHERE footnote_id IN
                (SELECT id FROM footnotes WHERE story_id = $1)`,
			`DELETE FROM footnotes WHERE story_id = $1`,
			`DELETE FROM vocabulary_items WHERE story_id = $1`,
			`DELETE FROM grammar_items WHERE story_id = $1`,
		}

		for _, query := range queries {
			if _, err := tx.Exec(query, storyID); err != nil {
				return err
			}
		}

		// Update last revision timestamp
		_, err := tx.Exec(`
            UPDATE stories
            SET last_revision = CURRENT_TIMESTAMP
            WHERE story_id = $1`,
			storyID)
		return err
	})
}

// ClearLineAnnotations removes all annotations from a specific line
func ClearLineAnnotations(storyID int, lineNumber int) error {
	return withTransaction(func(tx *sql.Tx) error {
		// Verify line exists
		var exists bool
		err := tx.QueryRow(`
			SELECT EXISTS(
				SELECT 1 FROM story_lines
				WHERE story_id = $1 AND line_number = $2
			)`, storyID, lineNumber).Scan(&exists)
		if err != nil {
			return err
		}
		if !exists {
			return ErrInvalidLineNumber
		}

		// Delete footnote references first, then footnotes
		_, err = tx.Exec(`
			DELETE FROM footnote_references
			WHERE footnote_id IN (
				SELECT id FROM footnotes
				WHERE story_id = $1 AND line_number = $2
			)`, storyID, lineNumber)
		if err != nil {
			return err
		}

		// Delete annotations for this line
		queries := []string{
			`DELETE FROM footnotes WHERE story_id = $1 AND line_number = $2`,
			`DELETE FROM vocabulary_items WHERE story_id = $1 AND line_number = $2`,
			`DELETE FROM grammar_items WHERE story_id = $1 AND line_number = $2`,
		}

		for _, query := range queries {
			if _, err := tx.Exec(query, storyID, lineNumber); err != nil {
				return err
			}
		}

		// Update last revision timestamp
		_, err = tx.Exec(`
			UPDATE stories
			SET last_revision = CURRENT_TIMESTAMP
			WHERE story_id = $1`, storyID)
		return err
	})
}

// UpdateVocabularyAnnotation updates a specific vocabulary annotation by position
func UpdateVocabularyAnnotation(storyID int, lineNumber int, position [2]int, vocab VocabularyItem) error {
	return withTransaction(func(tx *sql.Tx) error {
		// Update the vocabulary item
		result, err := tx.Exec(`
			UPDATE vocabulary_items
			SET word = $1, lexical_form = $2, position_start = $3, position_end = $4
			WHERE story_id = $5 AND line_number = $6 AND position_start = $7 AND position_end = $8`,
			vocab.Word, vocab.LexicalForm, vocab.Position[0], vocab.Position[1],
			storyID, lineNumber, position[0], position[1])
		if err != nil {
			return err
		}

		rowsAffected, err := result.RowsAffected()
		if err != nil {
			return err
		}
		if rowsAffected == 0 {
			return ErrInvalidLineNumber
		}

		// Update last revision timestamp
		_, err = tx.Exec(`
			UPDATE stories
			SET last_revision = CURRENT_TIMESTAMP
			WHERE story_id = $1`, storyID)
		return err
	})
}

// UpdateGrammarAnnotation updates a specific grammar annotation by position
func UpdateGrammarAnnotation(storyID int, lineNumber int, position [2]int, grammar GrammarItem) error {
	return withTransaction(func(tx *sql.Tx) error {
		// Update the grammar item
		result, err := tx.Exec(`
			UPDATE grammar_items
			SET text = $1, position_start = $2, position_end = $3
			WHERE story_id = $4 AND line_number = $5 AND position_start = $6 AND position_end = $7`,
			grammar.Text, grammar.Position[0], grammar.Position[1],
			storyID, lineNumber, position[0], position[1])
		if err != nil {
			return err
		}

		rowsAffected, err := result.RowsAffected()
		if err != nil {
			return err
		}
		if rowsAffected == 0 {
			return ErrInvalidLineNumber
		}

		// Update last revision timestamp
		_, err = tx.Exec(`
			UPDATE stories
			SET last_revision = CURRENT_TIMESTAMP
			WHERE story_id = $1`, storyID)
		return err
	})
}

// UpdateVocabularyByWord updates a vocabulary item's lexical form by matching the word
func UpdateVocabularyByWord(storyID int, lineNumber int, word string, newLexicalForm string) error {
	return withTransaction(func(tx *sql.Tx) error {
		// Update the vocabulary item by word
		result, err := tx.Exec(`
			UPDATE vocabulary_items
			SET lexical_form = $1
			WHERE story_id = $2 AND line_number = $3 AND word = $4`,
			newLexicalForm, storyID, lineNumber, word)
		if err != nil {
			return err
		}

		rowsAffected, err := result.RowsAffected()
		if err != nil {
			return err
		}
		if rowsAffected == 0 {
			return ErrInvalidLineNumber
		}

		// Update last revision timestamp
		_, err = tx.Exec(`
			UPDATE stories
			SET last_revision = CURRENT_TIMESTAMP
			WHERE story_id = $1`, storyID)
		return err
	})
}

// UpdateFootnoteAnnotation updates a specific footnote by ID
func UpdateFootnoteAnnotation(storyID int, footnoteID int, footnote Footnote) error {
	return withTransaction(func(tx *sql.Tx) error {
		// Update the footnote text
		result, err := tx.Exec(`
			UPDATE footnotes
			SET footnote_text = $1
			WHERE id = $2 AND story_id = $3`,
			footnote.Text, footnoteID, storyID)
		if err != nil {
			return err
		}

		rowsAffected, err := result.RowsAffected()
		if err != nil {
			return err
		}
		if rowsAffected == 0 {
			return ErrInvalidLineNumber
		}

		// Delete existing references
		_, err = tx.Exec(`DELETE FROM footnote_references WHERE footnote_id = $1`, footnoteID)
		if err != nil {
			return err
		}

		// Insert new references
		for _, ref := range footnote.References {
			_, err = tx.Exec(`
				INSERT INTO footnote_references (footnote_id, reference)
				VALUES ($1, $2)`, footnoteID, ref)
			if err != nil {
				return err
			}
		}

		// Update last revision timestamp
		_, err = tx.Exec(`
			UPDATE stories
			SET last_revision = CURRENT_TIMESTAMP
			WHERE story_id = $1`, storyID)
		return err
	})
}
