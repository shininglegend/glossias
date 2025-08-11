package models

import (
	"database/sql"

	_ "github.com/lib/pq"
)

func SaveNewStory(story *Story) error {
	return withTransaction(func(tx *sql.Tx) error {
		// Insert main story - Changed to RETURNING for PostgreSQL
		var storyID int
		err := tx.QueryRow(`
            INSERT INTO stories (week_number, day_letter, grammar_point, author_id, author_name)
            VALUES ($1, $2, $3, $4, $5)
            RETURNING story_id`,
			story.Metadata.WeekNumber, story.Metadata.DayLetter, story.Metadata.GrammarPoint,
			story.Metadata.Author.ID, story.Metadata.Author.Name).Scan(&storyID)
		if err != nil {
			return err
		}

		story.Metadata.StoryID = storyID
		return saveStoryComponents(tx, story)
	})
}

func SaveStoryData(storyID int, story *Story) error {
	exists, err := storyExists(storyID)
	if err != nil {
		return err
	}
	if !exists {
		return ErrNotFound
	}

	return withTransaction(func(tx *sql.Tx) error {
		// Changed placeholders and timestamp function
		_, err := tx.Exec(`
            UPDATE stories
            SET week_number = $1, day_letter = $2, grammar_point = $3,
                author_id = $4, author_name = $5, last_revision = CURRENT_TIMESTAMP
            WHERE story_id = $6`,
			story.Metadata.WeekNumber, story.Metadata.DayLetter, story.Metadata.GrammarPoint,
			story.Metadata.Author.ID, story.Metadata.Author.Name, storyID)
		if err != nil {
			return err
		}

		if err := deleteStoryComponents(tx, storyID); err != nil {
			return err
		}

		return saveStoryComponents(tx, story)
	})
}

func saveStoryComponents(tx *sql.Tx, story *Story) error {
	// Save titles - Changed to $n placeholders
	for lang, title := range story.Metadata.Title {
		if _, err := tx.Exec(`
            INSERT INTO story_titles (story_id, language_code, title)
            VALUES ($1, $2, $3)`,
			story.Metadata.StoryID, lang, title); err != nil {
			return err
		}
	}

	// Save description
	if story.Metadata.Description.Text != "" || story.Metadata.Description.Language != "" {
		if _, err := tx.Exec(`
            INSERT INTO story_descriptions (story_id, language_code, description_text)
            VALUES ($1, $2, $3)`,
			story.Metadata.StoryID, story.Metadata.Description.Language,
			story.Metadata.Description.Text); err != nil {
			return err
		}
	}

	return saveLines(tx, story.Metadata.StoryID, story.Content.Lines)
}

func saveLines(tx *sql.Tx, storyID int, lines []StoryLine) error {
	for _, line := range lines {
		if err := saveLine(tx, storyID, &line); err != nil {
			return err
		}
	}
	return nil
}

func saveLine(tx *sql.Tx, storyID int, line *StoryLine) error {
	// Save line
	_, err := tx.Exec(`
        INSERT INTO story_lines (story_id, line_number, text, audio_file)
        VALUES ($1, $2, $3, $4)`,
		storyID, line.LineNumber, line.Text, line.AudioFile)
	if err != nil {
		return err
	}

	// Save vocabulary
	for _, v := range line.Vocabulary {
		if err := dedupVocabularyInsert(tx, storyID, line.LineNumber, v); err != nil {
			return err
		}
	}

	// Save grammar items
	for _, g := range line.Grammar {
		if err := dedupGrammarInsert(tx, storyID, line.LineNumber, g); err != nil {
			return err
		}
	}

	// Save footnotes
	for _, f := range line.Footnotes {
		if err := dedupFootnoteInsert(tx, storyID, line.LineNumber, f); err != nil {
			return err
		}
	}

	return nil
}

func deleteStoryComponents(tx *sql.Tx, storyID int) error {
	tables := []string{"footnotes", "vocabulary_items", "grammar_items",
		"story_lines", "story_titles", "story_descriptions"}

	for _, table := range tables {
		if _, err := tx.Exec(`DELETE FROM `+table+` WHERE story_id = $1`, storyID); err != nil {
			return err
		}
	}
	return nil
}

func storyExists(id int) (bool, error) {
	var exists bool
	err := store.DB().QueryRow("SELECT EXISTS(SELECT 1 FROM stories WHERE story_id = $1)", id).Scan(&exists)
	return exists, err
}
