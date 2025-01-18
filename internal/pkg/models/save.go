package models

import (
	"database/sql"

	_ "github.com/mattn/go-sqlite3"
)

func SaveNewStory(story *Story) error {
	return withTransaction(func(tx *sql.Tx) error {
		// Insert main story
		result, err := tx.Exec(`
            INSERT INTO stories (week_number, day_letter, grammar_point, author_id, author_name)
            VALUES (?, ?, ?, ?, ?)`,
			story.Metadata.WeekNumber, story.Metadata.DayLetter, story.Metadata.GrammarPoint,
			story.Metadata.Author.ID, story.Metadata.Author.Name)
		if err != nil {
			return err
		}

		storyID, err := result.LastInsertId()
		if err != nil {
			return err
		}
		story.Metadata.StoryID = int(storyID)

		return saveStoryComponents(tx, story)
	})
}

func SaveStoryData(storyID int, story *Story) error {
	// Verify existence
	exists, err := storyExists(storyID)
	if err != nil {
		return err
	}
	if !exists {
		return ErrNotFound
	}

	return withTransaction(func(tx *sql.Tx) error {
		_, err := tx.Exec(`
            UPDATE stories
            SET week_number = ?, day_letter = ?, grammar_point = ?,
                author_id = ?, author_name = ?, last_revision = datetime('now')
            WHERE story_id = ?`,
			story.Metadata.WeekNumber, story.Metadata.DayLetter, story.Metadata.GrammarPoint,
			story.Metadata.Author.ID, story.Metadata.Author.Name, storyID)
		if err != nil {
			return err
		}

		// Delete existing components
		if err := deleteStoryComponents(tx, storyID); err != nil {
			return err
		}

		return saveStoryComponents(tx, story)
	})
}

func saveStoryComponents(tx *sql.Tx, story *Story) error {
	// Save titles
	for lang, title := range story.Metadata.Title {
		if _, err := tx.Exec(`
            INSERT INTO story_titles (story_id, language_code, title)
            VALUES (?, ?, ?)`,
			story.Metadata.StoryID, lang, title); err != nil {
			return err
		}
	}

	// Save description
	if story.Metadata.Description.Text != "" || story.Metadata.Description.Language != "" {
		if _, err := tx.Exec(`
            INSERT INTO story_descriptions (story_id, language_code, description_text)
            VALUES (?, ?, ?)`,
			story.Metadata.StoryID, story.Metadata.Description.Language,
			story.Metadata.Description.Text); err != nil {
			return err
		}
	}

	// Save lines and their components
	for _, line := range story.Content.Lines {
		if err := saveLine(tx, story.Metadata.StoryID, &line); err != nil {
			return err
		}
	}

	return nil
}

func saveLine(tx *sql.Tx, storyID int, line *StoryLine) error {
	// Save line
	_, err := tx.Exec(`
        INSERT INTO story_lines (story_id, line_number, text, audio_file)
        VALUES (?, ?, ?, ?)`,
		storyID, line.LineNumber, line.Text, line.AudioFile)
	if err != nil {
		return err
	}

	// Save vocabulary
	for _, v := range line.Vocabulary {
		if _, err := tx.Exec(`
            INSERT INTO vocabulary_items (story_id, line_number, word, lexical_form, position_start, position_end)
            VALUES (?, ?, ?, ?, ?, ?)`,
			storyID, line.LineNumber, v.Word, v.LexicalForm, v.Position[0], v.Position[1]); err != nil {
			return err
		}
	}

	// Save grammar items
	for _, g := range line.Grammar {
		if _, err := tx.Exec(`
            INSERT INTO grammar_items (story_id, line_number, text, position_start, position_end)
            VALUES (?, ?, ?, ?, ?)`,
			storyID, line.LineNumber, g.Text, g.Position[0], g.Position[1]); err != nil {
			return err
		}
	}

	// Save footnotes
	for _, f := range line.Footnotes {
		result, err := tx.Exec(`
            INSERT INTO footnotes (story_id, line_number, footnote_text)
            VALUES (?, ?, ?)`,
			storyID, line.LineNumber, f.Text)
		if err != nil {
			return err
		}

		footnoteID, err := result.LastInsertId()
		if err != nil {
			return err
		}

		// Save footnote references
		for _, ref := range f.References {
			if _, err := tx.Exec(`
                INSERT INTO footnote_references (footnote_id, reference)
                VALUES (?, ?)`,
				footnoteID, ref); err != nil {
				return err
			}
		}
	}

	return nil
}

func deleteStoryComponents(tx *sql.Tx, storyID int) error {
	// TODO: Delete the footnote references, lol
	tables := []string{"footnotes", "vocabulary_items", //"footnote_references",
		"grammar_items", "story_lines", "story_titles", "story_descriptions"}

	for _, table := range tables {
		if _, err := tx.Exec(`DELETE FROM `+table+` WHERE story_id = ?`, storyID); err != nil {
			return err
		}
	}
	return nil
}

func storyExists(id int) (bool, error) {
	var exists bool
	err := db.QueryRow("SELECT EXISTS(SELECT 1 FROM stories WHERE story_id = ?)", id).Scan(&exists)
	return exists, err
}
