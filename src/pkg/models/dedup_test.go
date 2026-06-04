package models

import (
	"context"
	"errors"
	"glossias/src/pkg/database"
	"testing"
)

func TestDedupVocabularyInsert_Exists(t *testing.T) {
	mockDB := database.NewMockDBTX()
	// CheckVocabularyExists returns bool
	mockDB.StubQuery("CheckVocabularyExists", [][]interface{}{{true}}, nil)

	SetDB(mockDB)
	defer func() {
		SetDB(struct{}{})
	}()

	vocab := VocabularyItem{
		Word:        "word",
		LexicalForm: "form",
		Position:    [2]int{0, 4},
	}
	err := dedupVocabularyInsert(context.Background(), 1, 1, vocab)

	if !errors.Is(err, errExists) {
		t.Errorf("expected errExists, got %v", err)
	}
}

func TestDedupVocabularyInsert_New(t *testing.T) {
	mockDB := database.NewMockDBTX()
	// CheckVocabularyExists returns false
	mockDB.StubQuery("CheckVocabularyExists", [][]interface{}{{false}}, nil)
	// CreateVocabularyItem returns id (int32)
	mockDB.StubQuery("CreateVocabularyItem", [][]interface{}{{int32(1)}}, nil)

	SetDB(mockDB)
	defer func() {
		SetDB(struct{}{})
	}()

	vocab := VocabularyItem{
		Word:        "word",
		LexicalForm: "form",
		Position:    [2]int{0, 4},
	}
	err := dedupVocabularyInsert(context.Background(), 1, 1, vocab)

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

func TestDedupGrammarInsert_Exists(t *testing.T) {
	mockDB := database.NewMockDBTX()
	// CheckGrammarExists returns bool
	mockDB.StubQuery("CheckGrammarExists", [][]interface{}{{true}}, nil)

	SetDB(mockDB)
	defer func() {
		SetDB(struct{}{})
	}()

	grammar := GrammarItem{
		Text:     "grammar",
		Position: [2]int{0, 7},
	}
	err := dedupGrammarInsert(context.Background(), 1, 1, grammar)

	if !errors.Is(err, errExists) {
		t.Errorf("expected errExists, got %v", err)
	}
}

func TestDedupGrammarInsert_New(t *testing.T) {
	mockDB := database.NewMockDBTX()
	// CheckGrammarExists returns false
	mockDB.StubQuery("CheckGrammarExists", [][]interface{}{{false}}, nil)
	// CreateGrammarItem returns id (int32)
	mockDB.StubQuery("CreateGrammarItem", [][]interface{}{{int32(1)}}, nil)

	SetDB(mockDB)
	defer func() {
		SetDB(struct{}{})
	}()

	grammar := GrammarItem{
		Text:     "grammar",
		Position: [2]int{0, 7},
	}
	err := dedupGrammarInsert(context.Background(), 1, 1, grammar)

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

func TestDedupFootnoteInsert_Exists(t *testing.T) {
	mockDB := database.NewMockDBTX()
	// CheckFootnoteExists returns existing id (int32 > 0)
	mockDB.StubQuery("CheckFootnoteExists", [][]interface{}{{int32(42)}}, nil)

	SetDB(mockDB)
	defer func() {
		SetDB(struct{}{})
	}()

	fn := Footnote{
		Text: "footnote text",
	}
	err := dedupFootnoteInsert(context.Background(), 1, 1, fn)

	if !errors.Is(err, errExists) {
		t.Errorf("expected errExists, got %v", err)
	}
}

func TestDedupFootnoteInsert_New(t *testing.T) {
	mockDB := database.NewMockDBTX()
	// CheckFootnoteExists returns 0 (not found)
	mockDB.StubQuery("CheckFootnoteExists", [][]interface{}{{int32(0)}}, nil)
	// CreateFootnote returns new id (int32)
	mockDB.StubQuery("CreateFootnote", [][]interface{}{{int32(99)}}, nil)
	// CreateFootnoteReference returns empty (nil error)
	mockDB.StubExec("CreateFootnoteReference", nil)

	SetDB(mockDB)
	defer func() {
		SetDB(struct{}{})
	}()

	fn := Footnote{
		Text:       "footnote text",
		References: []string{"ref1"},
	}
	err := dedupFootnoteInsert(context.Background(), 1, 1, fn)

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}
