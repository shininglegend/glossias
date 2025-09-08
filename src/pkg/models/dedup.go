package models

import (
	"context"
	"fmt"

	"glossias/src/pkg/generated/db"

	"github.com/jackc/pgx/v5/pgtype"
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
func dedupVocabularyInsert(ctx context.Context, storyID, lineNumber int, vocab VocabularyItem) error {
	if !dedupConfig.EnableVocabulary {
		return insertVocabulary(ctx, storyID, lineNumber, vocab)
	}

	// Check if vocabulary item exists using SQLC
	exists, err := queries.CheckVocabularyExists(ctx, db.CheckVocabularyExistsParams{
		StoryID:       pgtype.Int4{Int32: int32(storyID), Valid: true},
		LineNumber:    pgtype.Int4{Int32: int32(lineNumber), Valid: true},
		Word:          vocab.Word,
		LexicalForm:   vocab.LexicalForm,
		PositionStart: int32(vocab.Position[0]),
		PositionEnd:   int32(vocab.Position[1]),
	})

	if err != nil {
		return err
	}

	if exists {
		return errExists
	}
	return insertVocabulary(ctx, storyID, lineNumber, vocab)
}

// dedupGrammarInsert checks for existing grammar and returns existing ID or inserts new
func dedupGrammarInsert(ctx context.Context, storyID, lineNumber int, grammar GrammarItem) error {
	if !dedupConfig.EnableGrammar {
		return insertGrammar(ctx, storyID, lineNumber, grammar)
	}

	// Check if grammar item exists using SQLC
	exists, err := queries.CheckGrammarExists(ctx, db.CheckGrammarExistsParams{
		StoryID:       pgtype.Int4{Int32: int32(storyID), Valid: true},
		LineNumber:    pgtype.Int4{Int32: int32(lineNumber), Valid: true},
		Text:          grammar.Text,
		PositionStart: int32(grammar.Position[0]),
		PositionEnd:   int32(grammar.Position[1]),
	})

	if err != nil {
		return err
	}

	if exists {
		return errExists
	}
	return insertGrammar(ctx, storyID, lineNumber, grammar)
}

// dedupFootnoteInsert checks for existing footnote and returns existing ID or inserts new
func dedupFootnoteInsert(ctx context.Context, storyID, lineNumber int, footnote Footnote) error {
	if !dedupConfig.EnableFootnotes {
		return insertFootnote(ctx, storyID, lineNumber, footnote)
	}

	existingID, err := queries.CheckFootnoteExists(ctx, db.CheckFootnoteExistsParams{
		StoryID:      pgtype.Int4{Int32: int32(storyID), Valid: true},
		LineNumber:   pgtype.Int4{Int32: int32(lineNumber), Valid: true},
		FootnoteText: footnote.Text,
	})

	if existingID == 0 {
		return insertFootnote(ctx, storyID, lineNumber, footnote)
	}
	if err != nil {
		return err
	}
	return errExists
}

// Original insert functions using SQLC
func insertVocabulary(ctx context.Context, storyID, lineNumber int, vocab VocabularyItem) error {
	_, err := queries.CreateVocabularyItem(ctx, db.CreateVocabularyItemParams{
		StoryID:       pgtype.Int4{Int32: int32(storyID), Valid: true},
		LineNumber:    pgtype.Int4{Int32: int32(lineNumber), Valid: true},
		Word:          vocab.Word,
		LexicalForm:   vocab.LexicalForm,
		PositionStart: int32(vocab.Position[0]),
		PositionEnd:   int32(vocab.Position[1]),
	})
	return err
}

func insertGrammar(ctx context.Context, storyID, lineNumber int, grammar GrammarItem) error {
	grammarPointID := pgtype.Int4{Valid: false}
	if grammar.GrammarPointID != nil {
		grammarPointID = pgtype.Int4{Int32: int32(*grammar.GrammarPointID), Valid: true}
	}
	_, err := queries.CreateGrammarItem(ctx, db.CreateGrammarItemParams{
		StoryID:        pgtype.Int4{Int32: int32(storyID), Valid: true},
		LineNumber:     pgtype.Int4{Int32: int32(lineNumber), Valid: true},
		GrammarPointID: grammarPointID,
		Text:           grammar.Text,
		PositionStart:  int32(grammar.Position[0]),
		PositionEnd:    int32(grammar.Position[1]),
	})
	return err
}

func insertFootnote(ctx context.Context, storyID, lineNumber int, footnote Footnote) error {
	result, err := queries.CreateFootnote(ctx, db.CreateFootnoteParams{
		StoryID:      pgtype.Int4{Int32: int32(storyID), Valid: true},
		LineNumber:   pgtype.Int4{Int32: int32(lineNumber), Valid: true},
		FootnoteText: footnote.Text,
	})
	if err != nil {
		return err
	}

	for _, ref := range footnote.References {
		err := queries.CreateFootnoteReference(ctx, db.CreateFootnoteReferenceParams{
			FootnoteID: result,
			Reference:  ref,
		})
		if err != nil {
			return err
		}
	}
	return nil
}
