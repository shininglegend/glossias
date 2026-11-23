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

// annotationBatch gathers the annotations of one or more lines so a save
// checks and writes them with a constant number of queries instead of a few
// per item. Rows keep payload order (line by line, item by item), so serial
// ids are assigned in the same order the old per-item inserts produced.
type annotationBatch struct {
	storyID   int32
	vocab     []db.BulkCreateVocabularyItemsParams
	grammar   []db.BulkCreateGrammarItemsParams
	footnotes []batchFootnote
	audio     []db.BulkCreateAudioFilesParams
}

type batchFootnote struct {
	lineNumber int32
	text       string
	references []string
}

func newAnnotationBatch(storyID int) *annotationBatch {
	return &annotationBatch{storyID: int32(storyID)}
}

func (b *annotationBatch) key(lineNumber int) (pgtype.Int4, pgtype.Int4) {
	return pgtype.Int4{Int32: b.storyID, Valid: true}, pgtype.Int4{Int32: int32(lineNumber), Valid: true}
}

// addAnnotations queues a line's vocabulary, grammar and footnotes.
func (b *annotationBatch) addAnnotations(lineNumber int, line StoryLine) {
	storyID, ln := b.key(lineNumber)
	for _, v := range line.Vocabulary {
		b.vocab = append(b.vocab, db.BulkCreateVocabularyItemsParams{
			StoryID:       storyID,
			LineNumber:    ln,
			Word:          v.Word,
			LexicalForm:   v.LexicalForm,
			PositionStart: int32(v.Position[0]),
			PositionEnd:   int32(v.Position[1]),
		})
	}
	for _, g := range line.Grammar {
		grammarPointID := pgtype.Int4{}
		if g.GrammarPointID != nil {
			grammarPointID = pgtype.Int4{Int32: int32(*g.GrammarPointID), Valid: true}
		}
		b.grammar = append(b.grammar, db.BulkCreateGrammarItemsParams{
			StoryID:        storyID,
			LineNumber:     ln,
			GrammarPointID: grammarPointID,
			Text:           g.Text,
			PositionStart:  int32(g.Position[0]),
			PositionEnd:    int32(g.Position[1]),
		})
	}
	for _, f := range line.Footnotes {
		b.footnotes = append(b.footnotes, batchFootnote{lineNumber: int32(lineNumber), text: f.Text, references: f.References})
	}
}

// addAudio queues a line's audio files (full story saves only).
func (b *annotationBatch) addAudio(lineNumber int, files []AudioFile) {
	storyID, ln := b.key(lineNumber)
	for _, a := range files {
		b.audio = append(b.audio, db.BulkCreateAudioFilesParams{
			StoryID:    storyID,
			LineNumber: ln,
			FilePath:   a.FilePath,
			FileBucket: a.FileBucket,
			Label:      a.Label,
		})
	}
}

// write checks for duplicates, then inserts everything queued. It must run
// inside withTransaction (pass txCtx): any duplicate returns errExists and the
// caller's transaction rolls the whole save back.
func (b *annotationBatch) write(ctx context.Context) error {
	if err := b.checkDuplicates(ctx); err != nil {
		return err
	}
	if len(b.vocab) > 0 {
		if _, err := queries.BulkCreateVocabularyItems(ctx, b.vocab); err != nil {
			return err
		}
	}
	if len(b.grammar) > 0 {
		if _, err := queries.BulkCreateGrammarItems(ctx, b.grammar); err != nil {
			return err
		}
	}
	if len(b.footnotes) > 0 {
		if err := queries.InsertFootnotesWithRefs(ctx, b.footnotesParams()); err != nil {
			return err
		}
	}
	if len(b.audio) > 0 {
		if _, err := queries.BulkCreateAudioFiles(ctx, b.audio); err != nil {
			return err
		}
	}
	return nil
}

// checkDuplicates keeps the old per-item dedup semantics for every type whose
// dedup flag is on: an item that matches one already stored, or one earlier
// in the same payload (which the old code would have just inserted), is an
// error. The payload check runs first, in Go; the stored check is one query
// per type.
func (b *annotationBatch) checkDuplicates(ctx context.Context) error {
	if dedupConfig.EnableVocabulary && len(b.vocab) > 0 {
		// Every column of the row is part of the match.
		if hasDuplicate(b.vocab, func(v db.BulkCreateVocabularyItemsParams) db.BulkCreateVocabularyItemsParams { return v }) {
			return errExists
		}
		p := db.FindExistingVocabularyParams{StoryID: b.storyID}
		for _, v := range b.vocab {
			p.LineNumbers = append(p.LineNumbers, v.LineNumber.Int32)
			p.Words = append(p.Words, v.Word)
			p.LexicalForms = append(p.LexicalForms, v.LexicalForm)
			p.PositionStarts = append(p.PositionStarts, v.PositionStart)
			p.PositionEnds = append(p.PositionEnds, v.PositionEnd)
		}
		if err := failIfAny(queries.FindExistingVocabulary(ctx, p)); err != nil {
			return err
		}
	}

	if dedupConfig.EnableGrammar && len(b.grammar) > 0 {
		// grammar_point_id is not part of the match.
		if hasDuplicate(b.grammar, func(g db.BulkCreateGrammarItemsParams) db.BulkCreateGrammarItemsParams {
			g.GrammarPointID = pgtype.Int4{}
			return g
		}) {
			return errExists
		}
		p := db.FindExistingGrammarParams{StoryID: b.storyID}
		for _, g := range b.grammar {
			p.LineNumbers = append(p.LineNumbers, g.LineNumber.Int32)
			p.Texts = append(p.Texts, g.Text)
			p.PositionStarts = append(p.PositionStarts, g.PositionStart)
			p.PositionEnds = append(p.PositionEnds, g.PositionEnd)
		}
		if err := failIfAny(queries.FindExistingGrammar(ctx, p)); err != nil {
			return err
		}
	}

	if dedupConfig.EnableFootnotes && len(b.footnotes) > 0 {
		type footnoteKey struct {
			line int32
			text string
		}
		if hasDuplicate(b.footnotes, func(f batchFootnote) footnoteKey { return footnoteKey{f.lineNumber, f.text} }) {
			return errExists
		}
		p := db.FindExistingFootnotesParams{StoryID: b.storyID}
		for _, f := range b.footnotes {
			p.LineNumbers = append(p.LineNumbers, f.lineNumber)
			p.FootnoteTexts = append(p.FootnoteTexts, f.text)
		}
		if err := failIfAny(queries.FindExistingFootnotes(ctx, p)); err != nil {
			return err
		}
	}
	return nil
}

// footnotesParams flattens the queued footnotes and their references into the
// parallel arrays InsertFootnotesWithRefs takes; each reference carries the
// 1-based payload position of its footnote.
func (b *annotationBatch) footnotesParams() db.InsertFootnotesWithRefsParams {
	p := db.InsertFootnotesWithRefsParams{
		StoryID:         b.storyID,
		LineNumbers:     make([]int32, 0, len(b.footnotes)),
		FootnoteTexts:   make([]string, 0, len(b.footnotes)),
		RefFootnoteOrds: []int32{},
		ReferenceTexts:  []string{},
	}
	for i, f := range b.footnotes {
		p.LineNumbers = append(p.LineNumbers, f.lineNumber)
		p.FootnoteTexts = append(p.FootnoteTexts, f.text)
		for _, ref := range f.references {
			p.RefFootnoteOrds = append(p.RefFootnoteOrds, int32(i+1))
			p.ReferenceTexts = append(p.ReferenceTexts, ref)
		}
	}
	return p
}

// hasDuplicate reports whether two items share a key.
func hasDuplicate[T any, K comparable](items []T, key func(T) K) bool {
	seen := make(map[K]struct{}, len(items))
	for _, it := range items {
		k := key(it)
		if _, ok := seen[k]; ok {
			return true
		}
		seen[k] = struct{}{}
	}
	return false
}

// failIfAny turns a FindExisting* result into errExists when any payload
// index already exists.
func failIfAny(existing []int32, err error) error {
	if err != nil {
		return err
	}
	if len(existing) > 0 {
		return errExists
	}
	return nil
}
