package models

import (
	"database/sql"
	"fmt"
)

// DedupConfig controls which operations use deduplication
type DedupConfig struct {
	EnableVocabulary bool
	EnableGrammar    bool
	EnableFootnotes  bool
}

var dedupConfig = DedupConfig{
	EnableVocabulary: true,
	EnableGrammar:    true,
	EnableFootnotes:  true,
}

var errExists = fmt.Errorf("item already exists")

// SetDedupConfig allows toggling deduplication features
func SetDedupConfig(config DedupConfig) {
	dedupConfig = config
}

// dedupVocabularyInsert checks for existing vocabulary and returns existing ID or inserts new
func dedupVocabularyInsert(tx *sql.Tx, storyID, lineNumber int, vocab VocabularyItem) error {
	if !dedupConfig.EnableVocabulary {
		return insertVocabulary(tx, storyID, lineNumber, vocab)
	}

	var exists bool
	err := tx.QueryRow(`
		SELECT EXISTS(
			SELECT 1 FROM vocabulary_items
			WHERE story_id = $1 AND line_number = $2 AND word = $3 AND lexical_form = $4
			AND position_start = $5 AND position_end = $6
		)`, storyID, lineNumber, vocab.Word, vocab.LexicalForm, vocab.Position[0], vocab.Position[1]).Scan(&exists)

	if err != nil {
		return err
	}

	if exists {
		return errExists
	}
	return insertVocabulary(tx, storyID, lineNumber, vocab)
}

// dedupGrammarInsert checks for existing grammar and returns existing ID or inserts new
func dedupGrammarInsert(tx *sql.Tx, storyID, lineNumber int, grammar GrammarItem) error {
	if !dedupConfig.EnableGrammar {
		return insertGrammar(tx, storyID, lineNumber, grammar)
	}

	var exists bool
	err := tx.QueryRow(`
		SELECT EXISTS(
			SELECT 1 FROM grammar_items
			WHERE story_id = $1 AND line_number = $2 AND text = $3
			AND position_start = $4 AND position_end = $5
		)`, storyID, lineNumber, grammar.Text, grammar.Position[0], grammar.Position[1]).Scan(&exists)

	if err != nil {
		return err
	}

	if exists {
		return errExists
	}
	return insertGrammar(tx, storyID, lineNumber, grammar)
}

// dedupFootnoteInsert checks for existing footnote and returns existing ID or inserts new
func dedupFootnoteInsert(tx *sql.Tx, storyID, lineNumber int, footnote Footnote) error {
	if !dedupConfig.EnableFootnotes {
		return insertFootnote(tx, storyID, lineNumber, footnote)
	}

	var existingID int
	err := tx.QueryRow(`
		SELECT f.id FROM footnotes f
		JOIN footnote_references fr ON f.id = fr.footnote_id
		WHERE f.story_id = $1 AND f.line_number = $2 AND f.footnote_text = $3
		GROUP BY f.id, f.footnote_text
		HAVING array_agg(fr.reference ORDER BY fr.reference) = $4`,
		storyID, lineNumber, footnote.Text, fmt.Sprintf("{%s}", joinReferences(footnote.References))).Scan(&existingID)

	if err == sql.ErrNoRows {
		return insertFootnote(tx, storyID, lineNumber, footnote)
	}
	if err != nil {
		return err
	}
	return errExists
}

// Original insert functions (extracted from existing code)
func insertVocabulary(tx *sql.Tx, storyID, lineNumber int, vocab VocabularyItem) error {
	_, err := tx.Exec(`
		INSERT INTO vocabulary_items (
			story_id, line_number, word, lexical_form,
			position_start, position_end
		) VALUES ($1, $2, $3, $4, $5, $6)`,
		storyID, lineNumber, vocab.Word, vocab.LexicalForm,
		vocab.Position[0], vocab.Position[1])
	return err
}

func insertGrammar(tx *sql.Tx, storyID, lineNumber int, grammar GrammarItem) error {
	_, err := tx.Exec(`
		INSERT INTO grammar_items (
			story_id, line_number, text,
			position_start, position_end
		) VALUES ($1, $2, $3, $4, $5)`,
		storyID, lineNumber, grammar.Text,
		grammar.Position[0], grammar.Position[1])
	return err
}

func insertFootnote(tx *sql.Tx, storyID, lineNumber int, footnote Footnote) error {
	var footnoteID int
	err := tx.QueryRow(`
		INSERT INTO footnotes (story_id, line_number, footnote_text)
		VALUES ($1, $2, $3)
		RETURNING id`,
		storyID, lineNumber, footnote.Text).Scan(&footnoteID)
	if err != nil {
		return err
	}

	for _, ref := range footnote.References {
		if _, err := tx.Exec(`
			INSERT INTO footnote_references (footnote_id, reference)
			VALUES ($1, $2)`,
			footnoteID, ref); err != nil {
			return err
		}
	}
	return nil
}

func joinReferences(refs []string) string {
	if len(refs) == 0 {
		return ""
	}
	result := refs[0]
	for i := 1; i < len(refs); i++ {
		result += "," + refs[i]
	}
	return result
}
