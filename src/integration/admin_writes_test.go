//go:build integration

package integration

// Admin write paths against real Postgres: the rows each write produces (the
// expectations are built from the payload, i.e. what the old per-row code
// inserted), dedup rollbacks, and query counts that must not grow with the
// payload. Every test mutates, so each runs on a Fresh clone.

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"slices"
	"testing"

	"glossias/src/pkg/database"
	"glossias/src/pkg/models"
	"glossias/src/pkg/testdb"
	"glossias/src/pkg/testutil"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// adminWriteEnv is a fresh clone plus the fixtures and a grammar point any
// grammar item may reference.
type adminWriteEnv struct {
	pool         *pgxpool.Pool
	f            *fixtures
	grammarPoint int
}

func newAdminWriteEnv(t *testing.T) *adminWriteEnv {
	t.Helper()
	fx := testdb.Fresh(t)
	awLoadSeedUsers(fx)
	env := &adminWriteEnv{pool: fx.Pool, f: newFixtures(t, fx.Data)}
	if err := fx.Pool.QueryRow(context.Background(), "SELECT MIN(grammar_point_id) FROM grammar_points").Scan(&env.grammarPoint); err != nil {
		t.Fatal(err)
	}
	return env
}

func awLoadSeedUsers(fx *testdb.Fixture) {
	for _, r := range fx.Data.Table("users").Rows {
		seedUsers[r[0].(string)] = [2]string{r[1].(string), r[2].(string)}
	}
}

// counted runs fn with a query counter and returns fn's error and the count.
func awCounted(fn func(ctx context.Context) error) (error, int) {
	ctx, _ := database.WithQueryCounter(context.Background())
	err := fn(ctx)
	return err, database.QueryCount(ctx)
}

// withDedupConfig sets the global dedup flags for one test.
func awWithDedupConfig(t *testing.T, cfg models.DedupConfig) {
	t.Helper()
	models.SetDedupConfig(cfg)
	t.Cleanup(func() {
		models.SetDedupConfig(models.DedupConfig{EnableVocabulary: true, EnableGrammar: true, EnableFootnotes: true})
	})
}

func awIsErrExists(err error) bool { return err != nil && err.Error() == "item already exists" }

// --- row snapshots ---------------------------------------------------------

type awLineRow struct {
	Line int32
	Text string
}

type awVocabRow struct {
	Line       int32
	Word, Form string
	Start, End int32
}

type awGrammarRow struct {
	Line       int32
	Point      int32 // 0 = NULL
	Text       string
	Start, End int32
}

type awFootnoteRow struct {
	Line int32
	Text string
	Refs []string // sorted
}

type awAudioRow struct {
	Line                int32
	Path, Bucket, Label string
}

type awStorySnapshot struct {
	Titles    map[string]string
	Lines     []awLineRow
	Vocab     []awVocabRow
	Grammar   []awGrammarRow
	Footnotes []awFootnoteRow
	Audio     []awAudioRow
}

func awCollect[T any](t *testing.T, pool *pgxpool.Pool, sql string, args ...any) []T {
	t.Helper()
	rows, err := pool.Query(context.Background(), sql, args...)
	if err != nil {
		t.Fatal(err)
	}
	out, err := pgx.CollectRows(rows, pgx.RowToStructByPos[T])
	if err != nil {
		t.Fatal(err)
	}
	if out == nil {
		out = []T{}
	}
	return out
}

// snapshot reads a story's content in insertion (id) order.
func awSnapshot(t *testing.T, pool *pgxpool.Pool, storyID int) awStorySnapshot {
	t.Helper()
	type titleRow struct{ Lang, Title string }
	s := awStorySnapshot{Titles: map[string]string{}}
	for _, r := range awCollect[titleRow](t, pool, "SELECT language_code, title FROM story_titles WHERE story_id = $1", storyID) {
		s.Titles[r.Lang] = r.Title
	}
	s.Lines = awCollect[awLineRow](t, pool, "SELECT line_number, text FROM story_lines WHERE story_id = $1 ORDER BY line_number", storyID)
	s.Vocab = awCollect[awVocabRow](t, pool, `SELECT line_number, word, lexical_form, position_start, position_end
		FROM vocabulary_items WHERE story_id = $1 ORDER BY id`, storyID)
	s.Grammar = awCollect[awGrammarRow](t, pool, `SELECT line_number, COALESCE(grammar_point_id, 0), text, position_start, position_end
		FROM grammar_items WHERE story_id = $1 ORDER BY id`, storyID)
	s.Footnotes = awCollect[awFootnoteRow](t, pool, `SELECT f.line_number, f.footnote_text,
		COALESCE(array_agg(r.reference ORDER BY r.reference) FILTER (WHERE r.reference IS NOT NULL), '{}')
		FROM footnotes f LEFT JOIN footnote_references r ON r.footnote_id = f.id
		WHERE f.story_id = $1 GROUP BY f.id ORDER BY f.id`, storyID)
	s.Audio = awCollect[awAudioRow](t, pool, `SELECT line_number, file_path, file_bucket, label
		FROM line_audio_files WHERE story_id = $1 ORDER BY audio_file_id`, storyID)
	return s
}

// expected builds the snapshot the old per-row code wrote for a payload:
// rows in payload order (line by line, item by item).
func awExpected(story *models.Story) awStorySnapshot {
	s := awStorySnapshot{Titles: map[string]string{}, Lines: []awLineRow{}, Vocab: []awVocabRow{}, Grammar: []awGrammarRow{}, Footnotes: []awFootnoteRow{}, Audio: []awAudioRow{}}
	for k, v := range story.Metadata.Title {
		s.Titles[k] = v
	}
	for _, l := range story.Content.Lines {
		ln := int32(l.LineNumber)
		s.Lines = append(s.Lines, awLineRow{ln, l.Text})
		for _, v := range l.Vocabulary {
			s.Vocab = append(s.Vocab, awVocabRow{ln, v.Word, v.LexicalForm, int32(v.Position[0]), int32(v.Position[1])})
		}
		for _, g := range l.Grammar {
			var gp int32
			if g.GrammarPointID != nil {
				gp = int32(*g.GrammarPointID)
			}
			s.Grammar = append(s.Grammar, awGrammarRow{ln, gp, g.Text, int32(g.Position[0]), int32(g.Position[1])})
		}
		for _, fn := range l.Footnotes {
			refs := slices.Clone(fn.References)
			if refs == nil {
				refs = []string{}
			}
			slices.Sort(refs)
			s.Footnotes = append(s.Footnotes, awFootnoteRow{ln, fn.Text, refs})
		}
		for _, a := range l.AudioFiles {
			s.Audio = append(s.Audio, awAudioRow{ln, a.FilePath, a.FileBucket, a.Label})
		}
	}
	return s
}

func awAssertSnapshot(t *testing.T, got, want awStorySnapshot) {
	t.Helper()
	gv, wv := reflect.ValueOf(got), reflect.ValueOf(want)
	for i := range gv.NumField() {
		name := gv.Type().Field(i).Name
		if !reflect.DeepEqual(gv.Field(i).Interface(), wv.Field(i).Interface()) {
			t.Errorf("%s:\n got %+v\nwant %+v", name, gv.Field(i).Interface(), wv.Field(i).Interface())
		}
	}
}

// --- payloads --------------------------------------------------------------

// newStoryPayload is a story with n lines, each carrying vocabulary, grammar
// (with and without a grammar point), a footnote with 0-2 references and an
// audio file; every item is distinct so dedup passes.
func (env *adminWriteEnv) newStoryPayload(n int) *models.Story {
	course := env.f.ownCourse
	gp := env.grammarPoint
	s := &models.Story{Metadata: models.StoryMetadata{
		WeekNumber: 9, DayLetter: "z",
		Title:       map[string]string{"en": "Write test", "he": "בדיקה"},
		Author:      models.Author{ID: env.f.admin, Name: "Admin"},
		Description: models.Description{Text: "A description"},
		Language:    "en",
		CourseID:    &course,
	}}
	for i := range n {
		ln := i + 1
		line := models.StoryLine{
			LineNumber: ln,
			Text:       fmt.Sprintf("line %d text", ln),
			Vocabulary: []models.VocabularyItem{
				{Word: fmt.Sprintf("w%d", ln), LexicalForm: fmt.Sprintf("lex%d", ln), Position: [2]int{0, 2}},
				{Word: fmt.Sprintf("v%d", ln), LexicalForm: fmt.Sprintf("lex%d", ln), Position: [2]int{3, 5}},
			},
			Grammar: []models.GrammarItem{
				{Text: fmt.Sprintf("g%d", ln), Position: [2]int{0, 2}},
				{Text: fmt.Sprintf("h%d", ln), Position: [2]int{3, 5}, GrammarPointID: &gp},
			},
			Footnotes: []models.Footnote{{Text: fmt.Sprintf("note %d", ln), References: []string{"Ref B", "Ref A"}[:ln%3]}},
			AudioFiles: []models.AudioFile{
				{FilePath: fmt.Sprintf("s/%d.mp3", ln), FileBucket: "audio", Label: "complete"},
			},
		}
		if ln%2 == 0 {
			line.Footnotes = append(line.Footnotes, models.Footnote{Text: fmt.Sprintf("second note %d", ln), References: []string{"Other"}})
		}
		s.Content.Lines = append(s.Content.Lines, line)
	}
	return s
}

func awCountRows(t *testing.T, pool *pgxpool.Pool, sql string, args ...any) int {
	t.Helper()
	var n int
	if err := pool.QueryRow(context.Background(), sql, args...).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

// --- save -------------------------------------------------------------------

func TestSaveNewStoryWritesPayloadRows(t *testing.T) {
	env := newAdminWriteEnv(t)
	story := env.newStoryPayload(4)
	if err, _ := awCounted(func(ctx context.Context) error { return models.SaveNewStory(ctx, story) }); err != nil {
		t.Fatal(err)
	}
	awAssertSnapshot(t, awSnapshot(t, env.pool, story.Metadata.StoryID), awExpected(story))

	var desc string
	if err := env.pool.QueryRow(context.Background(),
		"SELECT description_text FROM story_descriptions WHERE story_id = $1 AND language_code = 'en'", story.Metadata.StoryID).Scan(&desc); err != nil || desc != "A description" {
		t.Errorf("description = %q, %v", desc, err)
	}
}

// SaveStoryData works on a story without lines (with lines it hits the known
// story_lines_pkey bug, kept as is).
func TestSaveStoryDataOnEmptyStoryWritesPayloadRows(t *testing.T) {
	env := newAdminWriteEnv(t)
	empty := env.newStoryPayload(0)
	if err := models.SaveNewStory(context.Background(), empty); err != nil {
		t.Fatal(err)
	}
	story := env.newStoryPayload(3)
	story.Metadata.StoryID = empty.Metadata.StoryID
	story.Metadata.Title = map[string]string{"en": "Renamed", "es": "Nuevo"}
	if err := models.SaveStoryData(context.Background(), story.Metadata.StoryID, story); err != nil {
		t.Fatal(err)
	}
	want := awExpected(story)
	want.Titles["he"] = "בדיקה" // titles are upserted, never deleted, by a full save
	awAssertSnapshot(t, awSnapshot(t, env.pool, story.Metadata.StoryID), want)

	// A second full save now fails on the existing lines (known bug) and
	// changes nothing.
	before := awSnapshot(t, env.pool, story.Metadata.StoryID)
	if err := models.SaveStoryData(context.Background(), story.Metadata.StoryID, env.newStoryPayload(3)); err == nil {
		t.Error("saving over existing lines succeeded; the known story_lines_pkey bug is expected to remain")
	}
	awAssertSnapshot(t, awSnapshot(t, env.pool, story.Metadata.StoryID), before)
}

func TestSaveNewStoryQueryCountIndependentOfSize(t *testing.T) {
	env := newAdminWriteEnv(t)
	counts := map[int]int{}
	for _, n := range []int{5, 50} {
		story := env.newStoryPayload(n)
		story.Metadata.WeekNumber = n // distinct stories
		err, c := awCounted(func(ctx context.Context) error { return models.SaveNewStory(ctx, story) })
		if err != nil {
			t.Fatal(err)
		}
		awAssertSnapshot(t, awSnapshot(t, env.pool, story.Metadata.StoryID), awExpected(story))
		counts[n] = c
	}
	if counts[5] != counts[50] {
		t.Errorf("SaveNewStory queries: 5 lines = %d, 50 lines = %d; want equal", counts[5], counts[50])
	}
	t.Logf("SaveNewStory queries: %d (5 and 50 lines)", counts[5])

	// The same holds for a full save of an empty story.
	saveCounts := map[int]int{}
	for _, n := range []int{5, 50} {
		empty := env.newStoryPayload(0)
		if err := models.SaveNewStory(context.Background(), empty); err != nil {
			t.Fatal(err)
		}
		story := env.newStoryPayload(n)
		story.Metadata.StoryID = empty.Metadata.StoryID
		err, c := awCounted(func(ctx context.Context) error { return models.SaveStoryData(ctx, story.Metadata.StoryID, story) })
		if err != nil {
			t.Fatal(err)
		}
		saveCounts[n] = c
	}
	if saveCounts[5] != saveCounts[50] {
		t.Errorf("SaveStoryData queries: 5 lines = %d, 50 lines = %d; want equal", saveCounts[5], saveCounts[50])
	}
	t.Logf("SaveStoryData queries: %d (5 and 50 lines)", saveCounts[5])
}

// --- dedup ------------------------------------------------------------------

func TestSaveNewStoryPayloadDuplicateRollsBack(t *testing.T) {
	cases := map[string]func(l *models.StoryLine){
		"vocabulary": func(l *models.StoryLine) { l.Vocabulary = append(l.Vocabulary, l.Vocabulary[0]) },
		"grammar": func(l *models.StoryLine) {
			dup := l.Grammar[1]
			dup.GrammarPointID = nil // the point is not part of the match
			l.Grammar = append(l.Grammar, dup)
		},
		"footnote": func(l *models.StoryLine) {
			l.Footnotes = append(l.Footnotes, models.Footnote{Text: l.Footnotes[0].Text, References: []string{"different"}})
		},
	}
	for name, dup := range cases {
		t.Run(name, func(t *testing.T) {
			env := newAdminWriteEnv(t)
			stories := awCountRows(t, env.pool, "SELECT COUNT(*) FROM stories")
			story := env.newStoryPayload(5)
			dup(&story.Content.Lines[3]) // late in the payload, after other rows were queued
			if err := models.SaveNewStory(context.Background(), story); !awIsErrExists(err) {
				t.Fatalf("err = %v, want item already exists", err)
			}
			if n := awCountRows(t, env.pool, "SELECT COUNT(*) FROM stories"); n != stories {
				t.Errorf("stories %d -> %d: the failed save was not rolled back", stories, n)
			}
		})
	}
}

func TestSaveNewStoryDedupFlagOffKeepsDuplicates(t *testing.T) {
	cases := []struct {
		name string
		cfg  models.DedupConfig
		dup  func(l *models.StoryLine)
	}{
		{"vocabulary", models.DedupConfig{EnableGrammar: true, EnableFootnotes: true},
			func(l *models.StoryLine) { l.Vocabulary = append(l.Vocabulary, l.Vocabulary[0]) }},
		{"grammar", models.DedupConfig{EnableVocabulary: true, EnableFootnotes: true},
			func(l *models.StoryLine) { l.Grammar = append(l.Grammar, l.Grammar[0]) }},
		{"footnote", models.DedupConfig{EnableVocabulary: true, EnableGrammar: true},
			func(l *models.StoryLine) { l.Footnotes = append(l.Footnotes, l.Footnotes[0]) }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			env := newAdminWriteEnv(t)
			awWithDedupConfig(t, tc.cfg)
			story := env.newStoryPayload(3)
			tc.dup(&story.Content.Lines[1])
			if err := models.SaveNewStory(context.Background(), story); err != nil {
				t.Fatalf("err = %v, want the duplicate kept with dedup off", err)
			}
			awAssertSnapshot(t, awSnapshot(t, env.pool, story.Metadata.StoryID), awExpected(story))
		})
	}
}

// With footnote dedup off, two footnotes with the same text on one line each
// keep their own references (refs join by payload position, not text).
func TestFootnoteRefsAttachToTheirOwnFootnote(t *testing.T) {
	env := newAdminWriteEnv(t)
	awWithDedupConfig(t, models.DedupConfig{EnableVocabulary: true, EnableGrammar: true})
	story := env.newStoryPayload(2)
	story.Content.Lines[0].Footnotes = []models.Footnote{
		{Text: "same", References: []string{"first-a", "first-b"}},
		{Text: "same", References: []string{"second"}},
		{Text: "same"},
	}
	if err := models.SaveNewStory(context.Background(), story); err != nil {
		t.Fatal(err)
	}
	got := awSnapshot(t, env.pool, story.Metadata.StoryID).Footnotes
	want := awExpected(story).Footnotes
	if !reflect.DeepEqual(got, want) {
		t.Errorf("footnotes:\n got %+v\nwant %+v", got, want)
	}
}

// --- line annotations ---------------------------------------------------------

func (env *adminWriteEnv) annotatedLine(t *testing.T) (storyID, line int) {
	t.Helper()
	// A seeded line with vocabulary, grammar and a footnote.
	err := env.pool.QueryRow(context.Background(), `
		SELECT v.story_id, v.line_number FROM vocabulary_items v
		JOIN grammar_items g ON g.story_id = v.story_id AND g.line_number = v.line_number
		JOIN footnotes f ON f.story_id = v.story_id AND f.line_number = v.line_number
		WHERE v.story_id = $1 ORDER BY v.line_number LIMIT 1`, env.f.story.ID).Scan(&storyID, &line)
	if err != nil {
		t.Fatalf("no fully annotated line on story %d: %v", env.f.story.ID, err)
	}
	return storyID, line
}

func awNewAnnotations(n, line int, gp int) models.StoryLine {
	l := models.StoryLine{LineNumber: line}
	for i := range n {
		l.Vocabulary = append(l.Vocabulary, models.VocabularyItem{Word: fmt.Sprintf("nw%d", i), LexicalForm: "nl", Position: [2]int{900 + i, 901 + i}})
		l.Grammar = append(l.Grammar, models.GrammarItem{Text: fmt.Sprintf("ng%d", i), Position: [2]int{900 + i, 901 + i}, GrammarPointID: &gp})
		l.Footnotes = append(l.Footnotes, models.Footnote{Text: fmt.Sprintf("new note %d", i), References: []string{"r1", "r2"}[:i%3]})
	}
	return l
}

func TestAddLineAnnotationsAppendsPayloadRows(t *testing.T) {
	env := newAdminWriteEnv(t)
	storyID, line := env.annotatedLine(t)
	before := awSnapshot(t, env.pool, storyID)
	payload := awNewAnnotations(3, line, env.grammarPoint)
	if err := models.AddLineAnnotations(context.Background(), storyID, line, payload); err != nil {
		t.Fatal(err)
	}
	added := awExpected(&models.Story{Content: models.StoryContent{Lines: []models.StoryLine{payload}}})
	want := before
	want.Vocab = append(slices.Clone(before.Vocab), added.Vocab...)
	want.Grammar = append(slices.Clone(before.Grammar), added.Grammar...)
	want.Footnotes = append(slices.Clone(before.Footnotes), added.Footnotes...)
	awAssertSnapshot(t, awSnapshot(t, env.pool, storyID), want)
}

func TestAddLineAnnotationsQueryCountIndependentOfSize(t *testing.T) {
	env := newAdminWriteEnv(t)
	storyID, line := env.annotatedLine(t)
	counts := map[int]int{}
	for _, n := range []int{1, 25} {
		payload := awNewAnnotations(n, line, env.grammarPoint)
		for i := range payload.Vocabulary { // distinct per run
			payload.Vocabulary[i].Word += fmt.Sprintf("-%d", n)
			payload.Grammar[i].Text += fmt.Sprintf("-%d", n)
			payload.Footnotes[i].Text += fmt.Sprintf("-%d", n)
		}
		err, c := awCounted(func(ctx context.Context) error { return models.AddLineAnnotations(ctx, storyID, line, payload) })
		if err != nil {
			t.Fatal(err)
		}
		counts[n] = c
	}
	if counts[1] != counts[25] {
		t.Errorf("AddLineAnnotations queries: 1 item each = %d, 25 = %d; want equal", counts[1], counts[25])
	}
	t.Logf("AddLineAnnotations queries: %d", counts[1])
}

func TestAddLineAnnotationsExistingDuplicateRollsBack(t *testing.T) {
	cases := map[string]func(env *adminWriteEnv, storyID, line int, l *models.StoryLine){
		"vocabulary": func(env *adminWriteEnv, storyID, line int, l *models.StoryLine) {
			var v models.VocabularyItem
			if err := env.pool.QueryRow(context.Background(), `SELECT word, lexical_form, position_start, position_end
				FROM vocabulary_items WHERE story_id = $1 AND line_number = $2 LIMIT 1`, storyID, line).
				Scan(&v.Word, &v.LexicalForm, &v.Position[0], &v.Position[1]); err != nil {
				panic(err)
			}
			l.Vocabulary = append(l.Vocabulary, v)
		},
		"grammar": func(env *adminWriteEnv, storyID, line int, l *models.StoryLine) {
			var g models.GrammarItem
			if err := env.pool.QueryRow(context.Background(), `SELECT text, position_start, position_end
				FROM grammar_items WHERE story_id = $1 AND line_number = $2 LIMIT 1`, storyID, line).
				Scan(&g.Text, &g.Position[0], &g.Position[1]); err != nil {
				panic(err)
			}
			l.Grammar = append(l.Grammar, g)
		},
		"footnote": func(env *adminWriteEnv, storyID, line int, l *models.StoryLine) {
			var f models.Footnote
			if err := env.pool.QueryRow(context.Background(), `SELECT footnote_text
				FROM footnotes WHERE story_id = $1 AND line_number = $2 LIMIT 1`, storyID, line).Scan(&f.Text); err != nil {
				panic(err)
			}
			l.Footnotes = append(l.Footnotes, f)
		},
	}
	for name, addDup := range cases {
		t.Run(name, func(t *testing.T) {
			env := newAdminWriteEnv(t)
			storyID, line := env.annotatedLine(t)
			before := awSnapshot(t, env.pool, storyID)
			var rev any
			env.pool.QueryRow(context.Background(), "SELECT last_revision FROM stories WHERE story_id = $1", storyID).Scan(&rev)

			payload := awNewAnnotations(2, line, env.grammarPoint) // new rows of every type...
			addDup(env, storyID, line, &payload)                   // ...plus one already stored
			if err := models.AddLineAnnotations(context.Background(), storyID, line, payload); !awIsErrExists(err) {
				t.Fatalf("err = %v, want item already exists", err)
			}
			awAssertSnapshot(t, awSnapshot(t, env.pool, storyID), before)
			var after any
			env.pool.QueryRow(context.Background(), "SELECT last_revision FROM stories WHERE story_id = $1", storyID).Scan(&after)
			if !reflect.DeepEqual(rev, after) {
				t.Errorf("last_revision changed on a rolled-back save: %v -> %v", rev, after)
			}

			// With that type's dedup off, the same payload goes through.
			cfg := models.DedupConfig{EnableVocabulary: name != "vocabulary", EnableGrammar: name != "grammar", EnableFootnotes: name != "footnote"}
			awWithDedupConfig(t, cfg)
			if err := models.AddLineAnnotations(context.Background(), storyID, line, payload); err != nil {
				t.Fatalf("dedup off: err = %v", err)
			}
		})
	}
}

// --- edit text / metadata / footnote ----------------------------------------

func TestEditStoryTextReplacesLinesWithConstantQueries(t *testing.T) {
	env := newAdminWriteEnv(t)
	sid := env.f.story.ID
	counts := map[int]int{}
	for _, n := range []int{5, 50} {
		var lines []models.StoryLine
		want := []awLineRow{}
		for i := range n {
			lines = append(lines, models.StoryLine{LineNumber: i + 1, Text: fmt.Sprintf("edited %d/%d", i+1, n)})
			want = append(want, awLineRow{int32(i + 1), lines[i].Text})
		}
		err, c := awCounted(func(ctx context.Context) error { return models.EditStoryText(ctx, sid, lines) })
		if err != nil {
			t.Fatal(err)
		}
		counts[n] = c
		if got := awSnapshot(t, env.pool, sid).Lines; !reflect.DeepEqual(got, want) {
			t.Errorf("%d lines: got %+v", n, got)
		}
	}
	if counts[5] != counts[50] {
		t.Errorf("EditStoryText queries: 5 lines = %d, 50 = %d; want equal", counts[5], counts[50])
	}
	t.Logf("EditStoryText queries: %d", counts[5])

	if err := models.EditStoryText(context.Background(), 999999, nil); err == nil {
		t.Error("EditStoryText on a missing story succeeded; want an error as before")
	}
}

func TestEditStoryMetadataWritesTitlesAndGrammarPoints(t *testing.T) {
	env := newAdminWriteEnv(t)
	sid := env.f.story.ID
	course := env.f.ownCourse
	meta := models.StoryMetadata{
		WeekNumber: 3, DayLetter: "b",
		Title:  map[string]string{"en": "New title", "es": "Nuevo título"},
		Author: models.Author{ID: env.f.admin, Name: "Admin"},
		GrammarPoints: []models.GrammarPoint{
			{Name: "Point Z", Description: "first"},
			{Name: "Point A", Description: ""},
			{Name: "Point M", Description: "third"},
		},
		CourseID: &course,
	}
	if err := models.EditStoryMetadata(context.Background(), sid, meta); err != nil {
		t.Fatal(err)
	}
	if got := awSnapshot(t, env.pool, sid).Titles; !reflect.DeepEqual(got, meta.Title) {
		t.Errorf("titles = %v, want %v", got, meta.Title)
	}
	type gpRow struct {
		Name string
		Desc *string
	}
	got := awCollect[gpRow](t, env.pool, "SELECT name, description FROM grammar_points WHERE story_id = $1 ORDER BY grammar_point_id", sid)
	if len(got) != 3 || got[0].Name != "Point Z" || got[1].Name != "Point A" || got[2].Name != "Point M" {
		t.Fatalf("grammar points = %+v, want payload order", got)
	}
	if got[0].Desc == nil || *got[0].Desc != "first" || got[1].Desc != nil || got[2].Desc == nil || *got[2].Desc != "third" {
		t.Errorf("descriptions: %v %v %v, want first, NULL, third", got[0].Desc, got[1].Desc, got[2].Desc)
	}
}

func TestUpdateFootnoteAnnotationReplacesReferences(t *testing.T) {
	env := newAdminWriteEnv(t)
	sid := env.f.story.ID
	var id int
	if err := env.pool.QueryRow(context.Background(), "SELECT MIN(id) FROM footnotes WHERE story_id = $1", sid).Scan(&id); err != nil {
		t.Fatal(err)
	}
	if err := models.UpdateFootnoteAnnotation(context.Background(), sid, id, models.Footnote{Text: "rewritten", References: []string{"z", "y", "x"}}); err != nil {
		t.Fatal(err)
	}
	refs := awCollect[struct{ R string }](t, env.pool, "SELECT reference FROM footnote_references WHERE footnote_id = $1 ORDER BY reference", id)
	if len(refs) != 3 || refs[0].R != "x" || refs[1].R != "y" || refs[2].R != "z" {
		t.Errorf("references = %+v, want x y z", refs)
	}
}

// --- translations -----------------------------------------------------------

func awPutJSON(t *testing.T, path, user string, body any) (*httptest.ResponseRecorder, int) {
	t.Helper()
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(body); err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest("PUT", path, &buf)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(testUserHeader, user)
	return testutil.ServeCounted(router, req)
}

func awTranslations(t *testing.T, pool *pgxpool.Pool, storyID int, lang string) map[int32]string {
	t.Helper()
	out := map[int32]string{}
	for _, r := range awCollect[struct {
		Line int32
		Text string
	}](t, pool, "SELECT line_number, translation_text FROM line_translations WHERE story_id = $1 AND language_code = $2", storyID, lang) {
		out[r.Line] = r.Text
	}
	return out
}

func TestBulkTranslationsLastWinsAndAtomic(t *testing.T) {
	env := newAdminWriteEnv(t)
	sid := env.f.story.ID
	path := fmt.Sprintf("/api/admin/stories/%d/translations", sid)

	rr, _ := awPutJSON(t, path, env.f.admin, map[string]any{"languageCode": "fr", "translations": []map[string]any{
		{"lineNumber": 1, "translation": "un"},
		{"lineNumber": 2, "translation": "deux"},
		{"lineNumber": 1, "translation": "UN"},
	}})
	if rr.Code != http.StatusOK {
		t.Fatalf("status %d: %s", rr.Code, rr.Body.String())
	}
	if got := awTranslations(t, env.pool, sid, "fr"); !reflect.DeepEqual(got, map[int32]string{1: "UN", 2: "deux"}) {
		t.Errorf("fr = %v, want line 1 last-wins", got)
	}

	// An unknown line fails the whole batch: line 2 keeps its old text.
	before := awTranslations(t, env.pool, sid, "fr")
	rr, _ = awPutJSON(t, path, env.f.admin, map[string]any{"languageCode": "fr", "translations": []map[string]any{
		{"lineNumber": 2, "translation": "changed"},
		{"lineNumber": 99999, "translation": "nope"},
	}})
	if rr.Code != http.StatusInternalServerError {
		t.Errorf("unknown line: status %d, want 500", rr.Code)
	}
	if got := awTranslations(t, env.pool, sid, "fr"); !reflect.DeepEqual(got, before) {
		t.Errorf("fr after failed batch = %v, want unchanged %v", got, before)
	}

	// One statement whatever the batch size.
	counts := map[int]int{}
	for _, n := range []int{2, env.f.story.Lines} {
		var body []map[string]any
		for i := range n {
			body = append(body, map[string]any{"lineNumber": i + 1, "translation": fmt.Sprintf("t%d", i)})
		}
		rr, c := awPutJSON(t, path, env.f.admin, map[string]any{"languageCode": "de", "translations": body})
		if rr.Code != http.StatusOK {
			t.Fatalf("status %d", rr.Code)
		}
		counts[n] = c
	}
	if counts[2] != counts[env.f.story.Lines] {
		t.Errorf("bulk translation queries: %v, want equal", counts)
	}
}

// --- attempts ---------------------------------------------------------------

func TestDeleteArchivedAttemptKeepsNumbersContiguous(t *testing.T) {
	env := newAdminWriteEnv(t)
	ctx := context.Background()
	user, sid := env.f.paid, env.f.story.ID
	setup := func(n int) []int64 {
		if _, err := env.pool.Exec(ctx, "DELETE FROM story_attempts WHERE user_id = $1 AND story_id = $2", user, sid); err != nil {
			t.Fatal(err)
		}
		var ids []int64
		for i := range n {
			var id int64
			if err := env.pool.QueryRow(ctx, `INSERT INTO story_attempts (user_id, story_id, attempt_number, completed_at)
				VALUES ($1, $2, $3, now()) RETURNING attempt_id`, user, sid, i+1).Scan(&id); err != nil {
				t.Fatal(err)
			}
			ids = append(ids, id)
		}
		return ids
	}
	numbers := func() map[int64]int32 {
		out := map[int64]int32{}
		for _, r := range awCollect[struct {
			ID  int64
			Num int32
		}](t, env.pool, "SELECT attempt_id, attempt_number FROM story_attempts WHERE user_id = $1 AND story_id = $2", user, sid) {
			out[r.ID] = r.Num
		}
		return out
	}
	del := func(n int) int {
		req := httptest.NewRequest("DELETE", fmt.Sprintf("/api/admin/stories/%d/students/%s/attempts/%d", sid, user, n), nil)
		req.Header.Set(testUserHeader, env.f.admin)
		rr, c := testutil.ServeCounted(router, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("delete attempt %d: status %d: %s", n, rr.Code, rr.Body.String())
		}
		return c
	}

	ids := setup(5)
	middle := del(2)
	if got, want := numbers(), map[int64]int32{ids[0]: 1, ids[2]: 2, ids[3]: 3, ids[4]: 4}; !reflect.DeepEqual(got, want) {
		t.Errorf("after deleting 2 of 5: %v, want %v", got, want)
	}
	del(1)
	if got, want := numbers(), map[int64]int32{ids[2]: 1, ids[3]: 2, ids[4]: 3}; !reflect.DeepEqual(got, want) {
		t.Errorf("after deleting 1: %v, want %v", got, want)
	}
	del(3) // the last one: nothing to renumber
	if got, want := numbers(), map[int64]int32{ids[2]: 1, ids[3]: 2}; !reflect.DeepEqual(got, want) {
		t.Errorf("after deleting the last: %v, want %v", got, want)
	}

	ids = setup(12)
	if c := del(2); c != middle {
		t.Errorf("delete-attempt queries: %d with 5 attempts, %d with 12; want equal", middle, c)
	}
	want := map[int64]int32{ids[0]: 1}
	for i, id := range ids[2:] {
		want[id] = int32(i + 2)
	}
	if got := numbers(); !reflect.DeepEqual(got, want) {
		t.Errorf("after deleting 2 of 12: %v, want %v", got, want)
	}
}

// --- produce ----------------------------------------------------------------

func TestDeleteProduceSegmentByOrder(t *testing.T) {
	env := newAdminWriteEnv(t)
	sid := env.f.story.ID
	var order int
	if err := env.pool.QueryRow(context.Background(), "SELECT segment_order FROM produce_segments WHERE id = $1", env.f.produceSegID).Scan(&order); err != nil {
		t.Fatal(err)
	}
	path := fmt.Sprintf("/api/admin/stories/%d/produce/segments/%d", sid, order)
	for i, want := range []int{http.StatusNoContent, http.StatusNotFound} {
		req := httptest.NewRequest("DELETE", path, nil)
		req.Header.Set(testUserHeader, env.f.admin)
		rr, _ := testutil.ServeCounted(router, req)
		if rr.Code != want {
			t.Errorf("delete #%d: status %d, want %d", i+1, rr.Code, want)
		}
	}
	if n := awCountRows(t, env.pool, "SELECT COUNT(*) FROM produce_segments WHERE id = $1", env.f.produceSegID); n != 0 {
		t.Errorf("segment still there")
	}
}
