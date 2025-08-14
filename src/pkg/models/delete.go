// glossias/src/pkg/models/delete.go
package models

import "database/sql"

// Delete removes a story and all its associated data from the database
func Delete(storyID int) error {
	// Verify story exists first
	exists, err := storyExists(storyID)
	if err != nil {
		return err
	}
	if !exists {
		return ErrNotFound
	}

	return withTransaction(func(tx *sql.Tx) error {
		// Delete in proper order to respect foreign key relationships
		// Though CASCADE would handle this, we're explicit for control
		if err := deleteFootnoteData(tx, storyID); err != nil {
			return err
		}

		if err := deleteAnnotations(tx, storyID); err != nil {
			return err
		}

		if err := deleteStoryContent(tx, storyID); err != nil {
			return err
		}

		if err := deleteMetadata(tx, storyID); err != nil {
			return err
		}

		// Finally delete the story itself
		if _, err := tx.Exec(`DELETE FROM stories WHERE story_id = $1`, storyID); err != nil {
			return err
		}

		return nil
	})
}

// deleteFootnoteData removes footnotes and their references
func deleteFootnoteData(tx *sql.Tx, storyID int) error {
	// Due to CASCADE, we only need to delete footnotes
	_, err := tx.Exec(`DELETE FROM footnotes WHERE story_id = $1`, storyID)
	return err
}

// deleteAnnotations removes vocabulary and grammar items
func deleteAnnotations(tx *sql.Tx, storyID int) error {
	queries := []string{
		`DELETE FROM vocabulary_items WHERE story_id = $1`,
		`DELETE FROM grammar_items WHERE story_id = $1`,
	}
	for _, query := range queries {
		if _, err := tx.Exec(query, storyID); err != nil {
			return err
		}
	}
	return nil
}

// deleteStoryContent removes the story lines
func deleteStoryContent(tx *sql.Tx, storyID int) error {
	_, err := tx.Exec(`DELETE FROM story_lines WHERE story_id = $1`, storyID)
	return err
}

// deleteMetadata removes titles and descriptions
func deleteMetadata(tx *sql.Tx, storyID int) error {
	queries := []string{
		`DELETE FROM story_titles WHERE story_id = $1`,
		`DELETE FROM story_descriptions WHERE story_id = $1`,
	}
	for _, query := range queries {
		if _, err := tx.Exec(query, storyID); err != nil {
			return err
		}
	}
	return nil
}
