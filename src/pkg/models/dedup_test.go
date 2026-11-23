package models

import (
	"context"
	"errors"
	"slices"
	"testing"

	"glossias/src/pkg/database"
)

func intPtr(n int) *int { return &n }

// batchFor builds a batch holding one line's annotations.
func batchFor(line StoryLine) *annotationBatch {
	b := newAnnotationBatch(1)
	b.addAnnotations(line.LineNumber, line)
	return b
}

func withDedup(t *testing.T, cfg DedupConfig) {
	t.Helper()
	prev := dedupConfig
	SetDedupConfig(cfg)
	t.Cleanup(func() { SetDedupConfig(prev) })
}

func useMock(t *testing.T) *database.MockDBTX {
	t.Helper()
	mockDB := database.NewMockDBTX()
	SetDB(mockDB)
	t.Cleanup(func() { SetDB(struct{}{}) })
	return mockDB
}

var allDedup = DedupConfig{EnableVocabulary: true, EnableGrammar: true, EnableFootnotes: true}

func TestAnnotationBatchExistingInDB(t *testing.T) {
	cases := []struct {
		name  string
		query string
		line  StoryLine
	}{
		{"vocabulary", "FindExistingVocabulary", StoryLine{LineNumber: 1, Vocabulary: []VocabularyItem{{Word: "w", LexicalForm: "f", Position: [2]int{0, 1}}}}},
		{"grammar", "FindExistingGrammar", StoryLine{LineNumber: 1, Grammar: []GrammarItem{{Text: "g", Position: [2]int{0, 1}}}}},
		{"footnote", "FindExistingFootnotes", StoryLine{LineNumber: 1, Footnotes: []Footnote{{Text: "f", References: []string{"r"}}}}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			withDedup(t, allDedup)
			mockDB := useMock(t)
			mockDB.StubQuery(tc.query, [][]any{{int32(0)}}, nil)

			if err := batchFor(tc.line).write(context.Background()); !errors.Is(err, errExists) {
				t.Fatalf("err = %v, want errExists", err)
			}
			if n := len(mockDB.Calls("InsertFootnotesWithRefs")); n != 0 {
				t.Errorf("inserted footnotes after a duplicate (%d calls)", n)
			}
		})
	}
}

func TestAnnotationBatchNewItemsCheckOncePerType(t *testing.T) {
	withDedup(t, allDedup)
	mockDB := useMock(t)

	line := StoryLine{
		LineNumber: 3,
		Vocabulary: []VocabularyItem{{Word: "a", LexicalForm: "a", Position: [2]int{0, 1}}, {Word: "b", LexicalForm: "b", Position: [2]int{2, 3}}},
		Grammar:    []GrammarItem{{Text: "g", Position: [2]int{0, 1}}, {Text: "h", Position: [2]int{2, 3}}},
		Footnotes:  []Footnote{{Text: "one", References: []string{"r1", "r2"}}, {Text: "two"}},
	}
	if err := batchFor(line).write(context.Background()); err != nil {
		t.Fatal(err)
	}
	for _, q := range []string{"FindExistingVocabulary", "FindExistingGrammar", "FindExistingFootnotes", "InsertFootnotesWithRefs"} {
		if n := len(mockDB.Calls(q)); n != 1 {
			t.Errorf("%s calls = %d, want 1", q, n)
		}
	}
}

func TestAnnotationBatchPayloadDuplicates(t *testing.T) {
	cases := []struct {
		name string
		line StoryLine
	}{
		{"vocabulary", StoryLine{LineNumber: 1, Vocabulary: []VocabularyItem{
			{Word: "w", LexicalForm: "f", Position: [2]int{0, 1}},
			{Word: "w", LexicalForm: "f", Position: [2]int{0, 1}},
		}}},
		// grammar_point_id is not part of the match.
		{"grammar", StoryLine{LineNumber: 1, Grammar: []GrammarItem{
			{Text: "g", Position: [2]int{0, 1}, GrammarPointID: intPtr(1)},
			{Text: "g", Position: [2]int{0, 1}, GrammarPointID: intPtr(2)},
		}}},
		// references are not part of the match.
		{"footnote", StoryLine{LineNumber: 1, Footnotes: []Footnote{
			{Text: "f", References: []string{"a"}},
			{Text: "f", References: []string{"b"}},
		}}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			withDedup(t, allDedup)
			mockDB := useMock(t)
			if err := batchFor(tc.line).write(context.Background()); !errors.Is(err, errExists) {
				t.Fatalf("err = %v, want errExists", err)
			}
			if n := len(mockDB.Calls("FindExisting")); n != 0 {
				t.Errorf("queried the DB for a payload duplicate (%d calls)", n)
			}
		})
	}
}

func TestAnnotationBatchSameItemOnDifferentLinesIsNotDuplicate(t *testing.T) {
	withDedup(t, allDedup)
	useMock(t)
	b := newAnnotationBatch(1)
	for ln := range 2 {
		b.addAnnotations(ln+1, StoryLine{
			Vocabulary: []VocabularyItem{{Word: "w", LexicalForm: "f", Position: [2]int{0, 1}}},
			Grammar:    []GrammarItem{{Text: "g", Position: [2]int{0, 1}}},
			Footnotes:  []Footnote{{Text: "f"}},
		})
	}
	if err := b.write(context.Background()); err != nil {
		t.Fatalf("err = %v, want nil", err)
	}
}

func TestAnnotationBatchDedupFlagsOff(t *testing.T) {
	cases := []struct {
		name string
		cfg  DedupConfig
		line StoryLine
	}{
		{"vocabulary", DedupConfig{EnableGrammar: true, EnableFootnotes: true}, StoryLine{LineNumber: 1, Vocabulary: []VocabularyItem{
			{Word: "w", LexicalForm: "f", Position: [2]int{0, 1}},
			{Word: "w", LexicalForm: "f", Position: [2]int{0, 1}},
		}}},
		{"grammar", DedupConfig{EnableVocabulary: true, EnableFootnotes: true}, StoryLine{LineNumber: 1, Grammar: []GrammarItem{
			{Text: "g", Position: [2]int{0, 1}},
			{Text: "g", Position: [2]int{0, 1}},
		}}},
		{"footnote", DedupConfig{EnableVocabulary: true, EnableGrammar: true}, StoryLine{LineNumber: 1, Footnotes: []Footnote{
			{Text: "f"},
			{Text: "f"},
		}}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			withDedup(t, tc.cfg)
			mockDB := useMock(t)
			// Even a stored match is ignored when the flag is off.
			mockDB.StubQuery("FindExisting", [][]any{{int32(0)}}, nil)
			if err := batchFor(tc.line).write(context.Background()); err != nil {
				t.Fatalf("err = %v, want nil with dedup off", err)
			}
			if n := len(mockDB.Calls("FindExisting")); n != 0 {
				t.Errorf("dedup query ran with the flag off (%d calls)", n)
			}
		})
	}
}

func TestAnnotationBatchFootnoteRefsKeyedByPosition(t *testing.T) {
	b := newAnnotationBatch(9)
	b.addAnnotations(1, StoryLine{Footnotes: []Footnote{{Text: "same", References: []string{"a", "b"}}}})
	b.addAnnotations(2, StoryLine{Footnotes: []Footnote{{Text: "none"}, {Text: "same", References: []string{"c"}}}})

	p := b.footnotesParams()
	if p.StoryID != 9 || !slices.Equal(p.LineNumbers, []int32{1, 2, 2}) || !slices.Equal(p.FootnoteTexts, []string{"same", "none", "same"}) {
		t.Errorf("footnotes = %+v", p)
	}
	if !slices.Equal(p.RefFootnoteOrds, []int32{1, 1, 3}) || !slices.Equal(p.ReferenceTexts, []string{"a", "b", "c"}) {
		t.Errorf("references = %v %v, want ords [1 1 3] for [a b c]", p.RefFootnoteOrds, p.ReferenceTexts)
	}
}

func TestAnnotationBatchEmptyRunsNoQueries(t *testing.T) {
	withDedup(t, allDedup)
	mockDB := useMock(t)
	if err := newAnnotationBatch(1).write(context.Background()); err != nil {
		t.Fatal(err)
	}
	if n := len(mockDB.Calls("")); n != 0 {
		t.Errorf("empty batch ran %d statements", n)
	}
}
