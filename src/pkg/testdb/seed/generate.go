package seed

import (
	"encoding/json"
	"fmt"
	"math/rand/v2"
	"strings"
	"time"

	"glossias/src/pkg/models"
)

// Table is the rows for one database table, in COPY order.
type Table struct {
	Name    string
	Columns []string
	Rows    [][]any
	// Serial is the SERIAL column whose sequence Load advances past the
	// generated ids, or "" when the table has none.
	Serial string
}

// Story describes one generated story for test lookups.
type Story struct {
	ID        int
	Courses   []int // course_stories links; Courses[0] is stories.course_id
	Trial     bool  // linked to a trial course
	Lines     int
	Footnotes int
}

// Dataset is a generated dataset plus handles tests use to pick fixtures.
type Dataset struct {
	Spec   Spec
	Tables []*Table

	SuperAdmin   string
	CourseAdmins map[int]string // course id -> admin user id
	PaidCourses  []int
	TrialCourses []int
	Students     map[Kind][]string
	Stories      []Story
	// Progress lists, per student, the stories they have exercise progress on.
	Progress map[string][]int
	// Archived lists, per student, the stories with a completed attempt.
	Archived map[string][]int
}

// StoriesInCourse returns the ids of stories linked to courseID.
func (d *Dataset) StoriesInCourse(courseID int) []int {
	var ids []int
	for _, s := range d.Stories {
		for _, c := range s.Courses {
			if c == courseID {
				ids = append(ids, s.ID)
				break
			}
		}
	}
	return ids
}

// MultiCourseStory returns a story linked to more than one course, or false.
func (d *Dataset) MultiCourseStory() (Story, bool) {
	for _, s := range d.Stories {
		if len(s.Courses) > 1 {
			return s, true
		}
	}
	return Story{}, false
}

// TrialStory returns a story in a trial course, or false.
func (d *Dataset) TrialStory() (Story, bool) {
	for _, s := range d.Stories {
		if s.Trial {
			return s, true
		}
	}
	return Story{}, false
}

// Table returns the named table, or nil.
func (d *Dataset) Table(name string) *Table {
	for _, t := range d.Tables {
		if t.Name == name {
			return t
		}
	}
	return nil
}

// words is the vocabulary story lines are built from. ASCII only, so text
// positions are the same in bytes, runes and UTF-16 units.
var words = []string{
	"shalom", "bayit", "sefer", "yeled", "yalda", "ima", "aba", "mayim", "lechem", "shemesh",
	"yareach", "kochav", "eretz", "shamayim", "derech", "ir", "melech", "malka", "lev", "yad",
	"regel", "rosh", "einayim", "kol", "davar", "yom", "layla", "boker", "erev", "shana",
	"chodesh", "shavua", "ish", "isha", "am", "goy", "har", "yam", "nahar", "etz",
	"perach", "kelev", "chatul", "sus", "tzipor", "dag", "lechem", "yayin", "shulchan", "kise",
}

type gen struct {
	spec Spec
	r    *rand.Rand
	ds   *Dataset
	t    map[string]*Table

	nextVocab, nextGrammarPoint, nextGrammarItem, nextFootnote int
	nextAudio, nextTargetVocab, nextRecall, nextProduce        int
}

func (g *gen) table(name, serial string, cols ...string) *Table {
	t := &Table{Name: name, Columns: cols, Serial: serial}
	g.t[name] = t
	g.ds.Tables = append(g.ds.Tables, t)
	return t
}

func (g *gen) add(name string, vals ...any) {
	t := g.t[name]
	if len(vals) != len(t.Columns) {
		panic(fmt.Sprintf("seed: %s got %d values for %d columns", name, len(vals), len(t.Columns)))
	}
	t.Rows = append(t.Rows, vals)
}

func (g *gen) between(r [2]int) int {
	if r[1] <= r[0] {
		return r[0]
	}
	return r[0] + g.r.IntN(r[1]-r[0]+1)
}

// Generate builds the dataset described by spec. It is deterministic: the
// same spec (including Now) always produces identical tables.
func Generate(spec Spec) *Dataset {
	if spec.Now.IsZero() {
		spec.Now = time.Date(2026, 9, 1, 12, 0, 0, 0, time.UTC)
	}
	spec.Now = spec.Now.UTC().Truncate(time.Second)
	g := &gen{
		spec: spec,
		r:    rand.New(rand.NewPCG(spec.Seed, 0x9e3779b97f4a7c15)),
		ds: &Dataset{
			Spec:         spec,
			CourseAdmins: map[int]string{},
			Students:     map[Kind][]string{},
			Progress:     map[string][]int{},
			Archived:     map[string][]int{},
		},
		t: map[string]*Table{},
	}

	// Parents before children: Load copies tables in this order.
	g.table("users", "", "user_id", "email", "name", "is_super_admin", "created_at")
	g.table("courses", "course_id", "course_id", "course_number", "name", "description", "is_trial")
	g.table("course_admins", "", "course_id", "user_id")
	g.table("course_users", "", "course_id", "user_id", "enrolled_at", "status")
	g.table("access_entitlements", "", "user_id", "course_id", "source", "starts_at", "expires_at", "amount_cents")
	g.table("stories", "story_id", "story_id", "week_number", "day_letter", "video_url", "last_revision", "author_id", "author_name", "course_id")
	g.table("course_stories", "", "course_id", "story_id")
	g.table("story_titles", "", "story_id", "language_code", "title")
	g.table("story_descriptions", "", "story_id", "language_code", "description_text")
	g.table("grammar_points", "grammar_point_id", "grammar_point_id", "story_id", "name", "description")
	g.table("story_lines", "", "story_id", "line_number", "text")
	g.table("line_translations", "", "story_id", "line_number", "language_code", "translation_text")
	g.table("line_audio_files", "audio_file_id", "audio_file_id", "story_id", "line_number", "file_path", "file_bucket", "label")
	g.table("vocabulary_items", "id", "id", "story_id", "line_number", "word", "lexical_form", "position_start", "position_end")
	g.table("grammar_items", "id", "id", "story_id", "line_number", "grammar_point_id", "text", "position_start", "position_end")
	g.table("footnotes", "id", "id", "story_id", "line_number", "footnote_text")
	g.table("footnote_references", "", "footnote_id", "reference")
	g.table("target_vocabulary", "id", "id", "story_id", "lexical_form", "audio_path", "audio_bucket", "correct_image_path", "image_bucket")
	g.table("recall_sentences", "id", "id", "story_id", "sequence_order", "hebrew_text", "target_vocab_id", "image_path", "image_bucket", "audio_path", "audio_bucket")
	g.table("produce_segments", "id", "id", "story_id", "segment_order", "hebrew_text", "reference_english", "grammar_point_id", "line_start", "line_end")
	g.table("story_produce_explanations", "", "story_id", "explanation_text")
	g.table("vocab_correct_answers", "score_id", "user_id", "story_id", "line_number", "vocab_item_id", "attempted_at")
	g.table("vocab_incorrect_answers", "id", "user_id", "story_id", "line_number", "vocab_item_id", "incorrect_answer", "attempted_at")
	g.table("grammar_correct_answers", "score_id", "user_id", "story_id", "line_number", "grammar_point_id", "attempted_at")
	g.table("identify_correct_answers", "id", "user_id", "story_id", "line_number", "target_vocab_id", "attempted_at")
	g.table("recall_correct_answers", "id", "user_id", "story_id", "recall_sentence_id", "attempted_at")
	g.table("translation_requests", "request_id", "user_id", "story_id", "requested_lines", "created_at", "completed_at")
	g.table("produce_submissions", "id", "user_id", "story_id", "segment_id", "student_text", "created_at")
	g.table("user_time_tracking", "tracking_id", "user_id", "route", "story_id", "started_at", "ended_at", "total_time_seconds", "phase")
	// Identity columns (GENERATED ALWAYS) are left to the database: on a
	// fresh database they are assigned 1..n in COPY order, which Load checks.
	g.table("story_attempts", "", "user_id", "story_id", "attempt_number", "started_at", "completed_at")
	g.table("story_attempt_score_snapshots", "", "attempt_id", "snapshot", "snapshot_at")

	g.courses()
	g.people()
	for i := range spec.Stories {
		g.story(i + 1)
	}
	g.progress()
	return g.ds
}

func (g *gen) courses() {
	id := 0
	for i := range g.spec.PaidCourses {
		id++
		g.add("courses", int32(id), fmt.Sprintf("HEB-%d01", i+1), fmt.Sprintf("Biblical Hebrew %d", i+1), "Paid course", false)
		g.ds.PaidCourses = append(g.ds.PaidCourses, id)
	}
	for i := range g.spec.TrialCourses {
		id++
		g.add("courses", int32(id), fmt.Sprintf("TRIAL-%d", i+1), fmt.Sprintf("Hebrew Trial %d", i+1), "Free trial", true)
		g.ds.TrialCourses = append(g.ds.TrialCourses, id)
	}
}

func (g *gen) user(id, name string, super bool) {
	email := strings.ReplaceAll(id, "_", ".") + "@example.test"
	g.add("users", id, email, name, super, g.spec.Now.Add(-200*24*time.Hour))
}

func (g *gen) people() {
	now := g.spec.Now
	g.ds.SuperAdmin = "seed_super_admin"
	g.user(g.ds.SuperAdmin, "Seed Super Admin", true)

	allCourses := append(append([]int{}, g.ds.PaidCourses...), g.ds.TrialCourses...)
	for _, c := range allCourses {
		id := fmt.Sprintf("seed_course_admin_%d", c)
		g.user(id, fmt.Sprintf("Seed Course Admin %d", c), false)
		g.add("course_admins", int32(c), id)
		g.ds.CourseAdmins[c] = id
	}

	enroll := func(user string, course int) {
		g.add("course_users", int32(course), user, now.Add(-60*24*time.Hour), "active")
	}
	entitle := func(user string, course int, expires time.Time) {
		g.add("access_entitlements", user, int32(course), "purchase", expires.Add(-180*24*time.Hour), expires, int32(4900))
	}
	for _, kind := range Kinds {
		for i := range g.spec.StudentsPerKind {
			id := fmt.Sprintf("seed_student_%s_%02d", kind, i+1)
			g.user(id, fmt.Sprintf("Seed Student %s %d", kind, i+1), false)
			g.ds.Students[kind] = append(g.ds.Students[kind], id)
			switch kind {
			case KindPaid:
				for _, c := range g.ds.PaidCourses {
					enroll(id, c)
					entitle(id, c, now.Add(90*24*time.Hour))
				}
			case KindExpired:
				if len(g.ds.PaidCourses) > 0 {
					enroll(id, g.ds.PaidCourses[0])
					entitle(id, g.ds.PaidCourses[0], now.Add(-10*24*time.Hour))
				}
			case KindUnpaid:
				if len(g.ds.PaidCourses) > 0 {
					enroll(id, g.ds.PaidCourses[0])
				}
			case KindTrial:
				for _, c := range g.ds.TrialCourses {
					enroll(id, c)
				}
			case KindUnenrolled:
			}
		}
	}
}

func (g *gen) story(id int) {
	spec := g.spec
	now := spec.Now
	s := Story{ID: id}
	if spec.TrialEvery > 0 && id%spec.TrialEvery == 0 && len(g.ds.TrialCourses) > 0 {
		s.Courses = []int{g.ds.TrialCourses[(id/spec.TrialEvery-1)%len(g.ds.TrialCourses)]}
		s.Trial = true
	} else if len(g.ds.PaidCourses) > 0 {
		owner := g.ds.PaidCourses[(id-1)%len(g.ds.PaidCourses)]
		s.Courses = []int{owner}
		if len(g.ds.PaidCourses) > 1 && g.r.Float64() < spec.MultiCourseFraction {
			for _, c := range g.ds.PaidCourses {
				if c != owner {
					s.Courses = append(s.Courses, c)
					break
				}
			}
		}
	}

	var owner any
	if len(s.Courses) > 0 {
		owner = int32(s.Courses[0])
	}
	author := g.ds.SuperAdmin
	g.add("stories", int32(id), int32((id-1)/5+1), string(rune('a'+(id-1)%5)),
		fmt.Sprintf("https://video.example.test/story-%d", id), now.Add(-time.Duration(id)*time.Hour),
		author, "Seed Super Admin", owner)
	for _, c := range s.Courses {
		g.add("course_stories", int32(c), int32(id))
	}
	g.add("story_titles", int32(id), "en", fmt.Sprintf("Story %d", id))
	g.add("story_descriptions", int32(id), "en", fmt.Sprintf("Seeded story number %d.", id))

	// Grammar points.
	var points []int
	for p := range g.between(spec.GrammarPoints) {
		g.nextGrammarPoint++
		points = append(points, g.nextGrammarPoint)
		g.add("grammar_points", int32(g.nextGrammarPoint), int32(id), fmt.Sprintf("Grammar point %d.%d", id, p+1), "Seeded grammar point")
	}

	// Lines and per-line content.
	s.Lines = g.between([2]int{spec.LinesMin, spec.LinesMax})
	var lexicalForms []string
	seenForm := map[string]bool{}
	for ln := 1; ln <= s.Lines; ln++ {
		n := 5 + g.r.IntN(6)
		lineWords := make([]string, n)
		for i := range lineWords {
			lineWords[i] = words[g.r.IntN(len(words))]
		}
		text := strings.Join(lineWords, " ")
		g.add("story_lines", int32(id), int32(ln), text)
		g.add("line_translations", int32(id), int32(ln), "en", fmt.Sprintf("Translation of story %d line %d.", id, ln))
		g.nextAudio++
		g.add("line_audio_files", int32(g.nextAudio), int32(id), int32(ln),
			fmt.Sprintf("stories/%d/line_%d.mp3", id, ln), "audio-files", "complete")

		// Word offsets within the line.
		starts := make([]int, n)
		pos := 0
		for i, w := range lineWords {
			starts[i] = pos
			pos += len(w) + 1
		}
		used := map[int]bool{}
		for range min(g.between(spec.VocabPerLine), n) {
			i := g.r.IntN(n)
			if used[i] {
				continue
			}
			used[i] = true
			w := lineWords[i]
			g.nextVocab++
			g.add("vocabulary_items", int32(g.nextVocab), int32(id), int32(ln), w, w, int32(starts[i]), int32(starts[i]+len(w)))
			if !seenForm[w] {
				seenForm[w] = true
				lexicalForms = append(lexicalForms, w)
			}
		}
		if len(points) > 0 && g.r.Float64() < spec.GrammarItemChance {
			i := g.r.IntN(n)
			g.nextGrammarItem++
			g.add("grammar_items", int32(g.nextGrammarItem), int32(id), int32(ln), int32(points[g.r.IntN(len(points))]),
				lineWords[i], int32(starts[i]), int32(starts[i]+len(lineWords[i])))
		}
		if spec.FootnoteEvery > 0 && ln%spec.FootnoteEvery == 0 {
			g.nextFootnote++
			s.Footnotes++
			g.add("footnotes", int32(g.nextFootnote), int32(id), int32(ln), fmt.Sprintf("Note on story %d line %d.", id, ln))
			for r := range 1 + g.r.IntN(3) {
				g.add("footnote_references", int32(g.nextFootnote), fmt.Sprintf("Ref %d", r+1))
			}
		}
	}

	// Target vocabulary drawn from lexical forms that occur in the story.
	var targets []int
	for i := range min(spec.TargetVocab, len(lexicalForms)) {
		g.nextTargetVocab++
		targets = append(targets, g.nextTargetVocab)
		form := lexicalForms[i]
		g.add("target_vocabulary", int32(g.nextTargetVocab), int32(id), form,
			fmt.Sprintf("target/%d/%s.mp3", id, form), "audio-files",
			fmt.Sprintf("target/%d/%s.png", id, form), "images")
	}
	for i := range min(spec.RecallSentences, 5, len(targets)) {
		g.nextRecall++
		g.add("recall_sentences", int32(g.nextRecall), int32(id), int32(i+1),
			fmt.Sprintf("Recall sentence %d for story %d", i+1, id), int32(targets[i]),
			fmt.Sprintf("recall/%d/%d.png", id, i+1), "images",
			fmt.Sprintf("recall/%d/%d.mp3", id, i+1), "audio-files")
	}
	for i := range min(spec.ProduceSegments, 2) {
		g.nextProduce++
		var gp any
		if len(points) > 0 {
			gp = int32(points[i%len(points)])
		}
		line := int32(min(i+1, s.Lines))
		g.add("produce_segments", int32(g.nextProduce), int32(id), int32(i+1),
			fmt.Sprintf("Hebrew produce segment %d", i+1), fmt.Sprintf("English reference %d", i+1), gp, line, line)
	}
	if spec.ProduceSegments > 0 {
		g.add("story_produce_explanations", int32(id), "Translate each segment into English.")
	}
	g.ds.Stories = append(g.ds.Stories, s)
}

// progress gives paid and unpaid students exercise answers on some of their
// stories, and some of those an archived attempt.
func (g *gen) progress() {
	now := g.spec.Now
	vocabByStory := map[int][][]any{}
	for _, row := range g.t["vocabulary_items"].Rows {
		vocabByStory[int(row[1].(int32))] = append(vocabByStory[int(row[1].(int32))], row)
	}
	grammarByStory := map[int][][]any{}
	for _, row := range g.t["grammar_items"].Rows {
		grammarByStory[int(row[1].(int32))] = append(grammarByStory[int(row[1].(int32))], row)
	}
	targetsByStory := map[int][]int32{}
	for _, row := range g.t["target_vocabulary"].Rows {
		targetsByStory[int(row[1].(int32))] = append(targetsByStory[int(row[1].(int32))], row[0].(int32))
	}
	recallByStory := map[int][]int32{}
	for _, row := range g.t["recall_sentences"].Rows {
		recallByStory[int(row[1].(int32))] = append(recallByStory[int(row[1].(int32))], row[0].(int32))
	}
	segmentsByStory := map[int][]int32{}
	for _, row := range g.t["produce_segments"].Rows {
		segmentsByStory[int(row[1].(int32))] = append(segmentsByStory[int(row[1].(int32))], row[0].(int32))
	}

	attemptID := 0
	students := append(append([]string{}, g.ds.Students[KindPaid]...), g.ds.Students[KindUnpaid]...)
	for _, user := range students {
		for _, s := range g.ds.Stories {
			if s.Trial || g.r.Float64() >= g.spec.ProgressFraction {
				continue
			}
			sid := int32(s.ID)
			at := now.Add(-time.Duration(1+g.r.IntN(30*24)) * time.Hour)
			g.ds.Progress[user] = append(g.ds.Progress[user], s.ID)

			vocab := vocabByStory[s.ID]
			for i, v := range vocab[:len(vocab)/2] {
				if i%5 == 4 {
					g.add("vocab_incorrect_answers", user, sid, v[2], v[0], "wrong", at)
				}
				g.add("vocab_correct_answers", user, sid, v[2], v[0], at)
			}
			for _, gi := range grammarByStory[s.ID][:len(grammarByStory[s.ID])/2] {
				g.add("grammar_correct_answers", user, sid, gi[2], gi[3], at)
			}
			for i, tv := range targetsByStory[s.ID] {
				g.add("identify_correct_answers", user, sid, int32(1+i%s.Lines), tv, at)
			}
			for _, rs := range recallByStory[s.ID][:len(recallByStory[s.ID])/2] {
				g.add("recall_correct_answers", user, sid, rs, at)
			}
			g.add("translation_requests", user, sid, []int32{1, 2}, at, at)
			if segs := segmentsByStory[s.ID]; len(segs) > 0 {
				g.add("produce_submissions", user, sid, segs[0], "My translation attempt.", at)
			}
			for _, phase := range []string{"video", "vocab", "grammar", "identify"} {
				g.add("user_time_tracking", user, fmt.Sprintf("/stories/%d/%s", s.ID, phase), sid,
					at, at.Add(5*time.Minute), int32(300), phase)
			}

			if g.r.Float64() < g.spec.ArchivedFraction {
				attemptID++
				done := at.Add(-24 * time.Hour)
				g.add("story_attempts", user, sid, int32(1), done.Add(-2*time.Hour), done)
				snap, err := json.Marshal(models.AttemptScoreSnapshot{
					StoryTitle:          fmt.Sprintf("Story %d", s.ID),
					TotalTimeSeconds:    1800,
					OverallAccuracy:     0.8,
					VocabAccuracy:       0.85,
					VocabCorrectCount:   len(vocab),
					GrammarAccuracy:     0.75,
					GrammarCorrectCount: len(grammarByStory[s.ID]),
				})
				if err != nil {
					panic(err)
				}
				g.add("story_attempt_score_snapshots", int64(attemptID), string(snap), done)
				g.ds.Archived[user] = append(g.ds.Archived[user], s.ID)
			}
		}
	}
}
