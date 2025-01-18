// story_data.go
package models

import (
	"database/sql"

	_ "github.com/mattn/go-sqlite3"
)

func GetStoryData(id int) (*Story, error) {
	story := NewStory()

	// Get main story data
	err := db.QueryRow(`
        SELECT s.week_number, s.day_letter, s.grammar_point,
               s.last_revision, s.author_id, s.author_name
        FROM stories s
        WHERE s.story_id = ?`, id).Scan(
		&story.Metadata.WeekNumber,
		&story.Metadata.DayLetter,
		&story.Metadata.GrammarPoint,
		&story.Metadata.LastRevision,
		&story.Metadata.Author.ID,
		&story.Metadata.Author.Name,
	)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	story.Metadata.StoryID = id

	// Get titles
	rows, err := db.Query(`
        SELECT language_code, title
        FROM story_titles
        WHERE story_id = ?`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var lang, title string
		if err := rows.Scan(&lang, &title); err != nil {
			return nil, err
		}
		story.Metadata.Title[lang] = title
	}

	// Get description
	err = db.QueryRow(`
        SELECT language_code, description_text
        FROM story_descriptions
        WHERE story_id = ?
        LIMIT 1`, id).Scan(
		&story.Metadata.Description.Language,
		&story.Metadata.Description.Text,
	)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}

	// Get lines with their components
	lines, err := getStoryLines(id)
	if err != nil {
		return nil, err
	}
	story.Content.Lines = lines

	return story, nil
}

func getStoryLines(storyID int) ([]StoryLine, error) {
	rows, err := db.Query(`
        SELECT line_number, text, audio_file
        FROM story_lines
        WHERE story_id = ?
        ORDER BY line_number`, storyID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var lines []StoryLine
	for rows.Next() {
		line := StoryLine{
			Vocabulary: []VocabularyItem{}, // Init with empty arrays
			Grammar:    []GrammarItem{},
			Footnotes:  []Footnote{},
		}
		var audioFile sql.NullString
		if err := rows.Scan(&line.LineNumber, &line.Text, &audioFile); err != nil {
			return nil, err
		}
		if audioFile.Valid {
			s := audioFile.String
			line.AudioFile = &s
		}

		// Get vocabulary items
		if err := getVocabularyItems(storyID, line.LineNumber, &line); err != nil {
			return nil, err
		}

		// Get grammar items
		if err := getGrammarItems(storyID, line.LineNumber, &line); err != nil {
			return nil, err
		}

		// Get footnotes
		if err := getFootnotes(storyID, line.LineNumber, &line); err != nil {
			return nil, err
		}

		lines = append(lines, line)
	}
	return lines, nil
}

// Helper functions to get line components
func getVocabularyItems(storyID, lineNumber int, line *StoryLine) error {
	rows, err := db.Query(`
        SELECT word, lexical_form, position_start, position_end
        FROM vocabulary_items
        WHERE story_id = ? AND line_number = ?`,
		storyID, lineNumber)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var item VocabularyItem
		if err := rows.Scan(&item.Word, &item.LexicalForm, &item.Position[0], &item.Position[1]); err != nil {
			return err
		}
		line.Vocabulary = append(line.Vocabulary, item)
	}
	return nil
}

// story_data.go (continued)

func getGrammarItems(storyID, lineNumber int, line *StoryLine) error {
	rows, err := db.Query(`
        SELECT text, position_start, position_end
        FROM grammar_items
        WHERE story_id = ? AND line_number = ?`,
		storyID, lineNumber)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var item GrammarItem
		if err := rows.Scan(&item.Text, &item.Position[0], &item.Position[1]); err != nil {
			return err
		}
		line.Grammar = append(line.Grammar, item)
	}
	return nil
}

func getFootnotes(storyID, lineNumber int, line *StoryLine) error {
	// Get footnotes
	rows, err := db.Query(`
        SELECT f.id, f.footnote_text
        FROM footnotes f
        WHERE f.story_id = ? AND f.line_number = ?`,
		storyID, lineNumber)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var footnote Footnote
		if err := rows.Scan(&footnote.ID, &footnote.Text); err != nil {
			return err
		}

		// Get references for each footnote
		refs, err := getFootnoteReferences(footnote.ID)
		if err != nil {
			return err
		}
		footnote.References = refs

		line.Footnotes = append(line.Footnotes, footnote)
	}
	return nil
}

func getFootnoteReferences(footnoteID int) ([]string, error) {
	rows, err := db.Query(`
        SELECT reference
        FROM footnote_references
        WHERE footnote_id = ?`,
		footnoteID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var references []string
	for rows.Next() {
		var ref string
		if err := rows.Scan(&ref); err != nil {
			return nil, err
		}
		references = append(references, ref)
	}
	return references, nil
}

// GetLineAnnotations retrieves all annotations for a specific line
func GetLineAnnotations(storyID int, lineNumber int) (*StoryLine, error) {
	line := &StoryLine{
		LineNumber: lineNumber,
		Vocabulary: []VocabularyItem{}, // init as empty arrays
		Grammar:    []GrammarItem{},
		Footnotes:  []Footnote{},
	}

	// Get vocabulary items
	if err := getVocabularyItems(storyID, lineNumber, line); err != nil {
		return nil, err
	}

	// Get grammar items
	if err := getGrammarItems(storyID, lineNumber, line); err != nil {
		return nil, err
	}

	// Get footnotes
	if err := getFootnotes(storyID, lineNumber, line); err != nil {
		return nil, err
	}

	return line, nil
}

// Helper function to execute transaction with error handling
func withTransaction(fn func(*sql.Tx) error) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}

	defer func() {
		if p := recover(); p != nil {
			// Something went wrong, rollback
			_ = tx.Rollback()
			panic(p) // re-throw panic after rollback
		}
	}()

	if err := fn(tx); err != nil {
		// Something went wrong, rollback
		_ = tx.Rollback()
		return err
	}

	// All good, commit
	return tx.Commit()
}

func GetAllStories(language string) ([]Story, error) {
	rows, err := db.Query(`
        SELECT DISTINCT s.story_id, s.week_number, s.day_letter, st.title
        FROM stories s
        JOIN story_titles st ON s.story_id = st.story_id
        ORDER BY s.week_number, s.day_letter`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var stories []Story
	for rows.Next() {
		var story Story
		var title string
		// [+] Added week_number and day_letter to scan
		if err := rows.Scan(
			&story.Metadata.StoryID,
			&story.Metadata.WeekNumber,
			&story.Metadata.DayLetter,
			&title,
		); err != nil {
			return nil, err
		}
		story.Metadata.Title = map[string]string{"en": title}
		stories = append(stories, story)
	}
	return stories, nil
}
