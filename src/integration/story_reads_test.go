//go:build integration

package integration

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"

	"glossias/src/pkg/database"
	"glossias/src/pkg/models"
	"glossias/src/pkg/testdb"
	"glossias/src/pkg/testdb/seed"
	"glossias/src/pkg/testutil"
)

// fillSeedUsers lets the fake auth middleware recognise every seeded user.
func fillSeedUsers(ds *seed.Dataset) {
	for _, r := range ds.Table("users").Rows {
		seedUsers[r[0].(string)] = [2]string{r[1].(string), r[2].(string)}
	}
}

// singleCourseStory is a paid story linked only to its owner course.
func singleCourseStory(t *testing.T, ds *seed.Dataset) seed.Story {
	t.Helper()
	for _, s := range ds.Stories {
		if !s.Trial && len(s.Courses) == 1 {
			return s
		}
	}
	t.Fatal("seed has no single-course paid story")
	return seed.Story{}
}

func paidStudent(ds *seed.Dataset) string { return ds.Students[seed.KindPaid][0] }

func findLine(lines []models.StoryLine, n int) *models.StoryLine {
	for i := range lines {
		if lines[i].LineNumber == n {
			return &lines[i]
		}
	}
	return nil
}

// Footnotes without references read back as no references (not [""]), and
// seeded references come back sorted.
func TestFootnotesWithoutReferences(t *testing.T) {
	fx := testdb.Fresh(t)
	ctx := context.Background()
	s := singleCourseStory(t, fx.Data)
	var fnID int
	if err := fx.Pool.QueryRow(ctx,
		"INSERT INTO footnotes (story_id, line_number, footnote_text) VALUES ($1, 1, 'Bare note') RETURNING id",
		s.ID).Scan(&fnID); err != nil {
		t.Fatal(err)
	}
	models.ClearAllCache()

	check := func(where string, fns []models.Footnote) {
		t.Helper()
		found := false
		for _, fn := range fns {
			if slices.Contains(fn.References, "") {
				t.Errorf("%s: footnote %d has an empty reference: %q", where, fn.ID, fn.References)
			}
			if !slices.IsSorted(fn.References) {
				t.Errorf("%s: footnote %d references not sorted: %q", where, fn.ID, fn.References)
			}
			if fn.ID == fnID {
				found = true
				if len(fn.References) != 0 {
					t.Errorf("%s: bare footnote references = %q, want []", where, fn.References)
				}
			} else if len(fn.References) == 0 {
				t.Errorf("%s: seeded footnote %d lost its references", where, fn.ID)
			}
		}
		if !found {
			t.Errorf("%s: bare footnote %d missing", where, fnID)
		}
	}

	line, err := models.GetLineAnnotations(ctx, s.ID, 1)
	if err != nil {
		t.Fatal(err)
	}
	check("line annotations", line.Footnotes)

	ann, err := models.GetStoryAnnotations(ctx, s.ID)
	if err != nil {
		t.Fatal(err)
	}
	var all []models.Footnote
	for _, l := range ann {
		all = append(all, l.Footnotes...)
	}
	check("story annotations", all)

	story, err := models.GetStoryData(ctx, s.ID, fx.Data.SuperAdmin)
	if err != nil {
		t.Fatal(err)
	}
	all = nil
	for _, l := range story.Content.Lines {
		all = append(all, l.Footnotes...)
	}
	check("story data", all)
}

// An audio row written through the model shows up in the cached story and
// line annotations at once.
func TestAudioWriteInvalidatesStory(t *testing.T) {
	fx := testdb.Fresh(t)
	ctx := context.Background()
	s := singleCourseStory(t, fx.Data)
	user := paidStudent(fx.Data)

	if _, err := models.GetStoryData(ctx, s.ID, user); err != nil {
		t.Fatal(err)
	}
	if _, err := models.GetLineAnnotations(ctx, s.ID, 1); err != nil {
		t.Fatal(err)
	}

	if _, err := models.CreateAudioFile(ctx, s.ID, 1, fmt.Sprintf("stories/%d/line_1_slow.mp3", s.ID), "audio-files", "slow"); err != nil {
		t.Fatal(err)
	}

	hasSlow := func(files []models.AudioFile) bool {
		return slices.ContainsFunc(files, func(a models.AudioFile) bool { return a.Label == "slow" })
	}
	story, err := models.GetStoryData(ctx, s.ID, user)
	if err != nil {
		t.Fatal(err)
	}
	if l := findLine(story.Content.Lines, 1); l == nil || !hasSlow(l.AudioFiles) {
		t.Errorf("GetStoryData line 1 audio after write: no 'slow' file")
	}
	line, err := models.GetLineAnnotations(ctx, s.ID, 1)
	if err != nil {
		t.Fatal(err)
	}
	if !hasSlow(line.AudioFiles) {
		t.Errorf("GetLineAnnotations line 1 audio after write: no 'slow' file")
	}
}

// InvalidateStoryMetadata also drops the per-line annotation entries.
func TestLineAnnotationsFreshAfterStoryInvalidation(t *testing.T) {
	fx := testdb.Fresh(t)
	ctx := context.Background()
	s := singleCourseStory(t, fx.Data)

	before, err := models.GetLineAnnotations(ctx, s.ID, 2)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := fx.Pool.Exec(ctx,
		"INSERT INTO footnotes (story_id, line_number, footnote_text) VALUES ($1, 2, 'Added later')", s.ID); err != nil {
		t.Fatal(err)
	}
	models.InvalidateStoryMetadata(s.ID)

	after, err := models.GetLineAnnotations(ctx, s.ID, 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(after.Footnotes) != len(before.Footnotes)+1 {
		t.Errorf("line 2 footnotes = %d after invalidation, want %d", len(after.Footnotes), len(before.Footnotes)+1)
	}
}

// After access is taken away, the next check denies even though "allowed"
// was cached a moment before.
func TestAccessRevocation(t *testing.T) {
	ctx := context.Background()

	// cachedAllow opens the story and proves the next check is served from
	// the cache (no queries at all: access and story both cached).
	cachedAllow := func(t *testing.T, storyID int, user string) {
		t.Helper()
		if _, err := models.GetStoryData(ctx, storyID, user); err != nil {
			t.Fatalf("before revocation: %v", err)
		}
		cctx, _ := database.WithQueryCounter(ctx)
		if _, err := models.GetStoryData(cctx, storyID, user); err != nil {
			t.Fatalf("before revocation (cached): %v", err)
		}
		if n := database.QueryCount(cctx); n != 0 {
			t.Fatalf("cached GetStoryData made %d queries, want 0", n)
		}
	}
	denied := func(t *testing.T, storyID int, user string) {
		t.Helper()
		if _, err := models.GetStoryData(ctx, storyID, user); !errors.Is(err, models.ErrNotFound) {
			t.Errorf("after revocation: err = %v, want ErrNotFound", err)
		}
	}

	t.Run("unlink course", func(t *testing.T) {
		fx := testdb.Fresh(t)
		s := singleCourseStory(t, fx.Data)
		user := fx.Data.Students[seed.KindUnenrolled][0]
		c, err := models.CreateCourse(ctx, "REVOKE-1", "Revocation test", "", false)
		if err != nil {
			t.Fatal(err)
		}
		// Plain SQL: AddUserToCourseByEmail rejects every valid status today.
		if _, err := fx.Pool.Exec(ctx, "INSERT INTO course_users (course_id, user_id, status) VALUES ($1, $2, 'active')", c.CourseID, user); err != nil {
			t.Fatal(err)
		}
		if err := models.LinkStoryCourse(ctx, int32(s.ID), c.CourseID); err != nil {
			t.Fatal(err)
		}
		cachedAllow(t, s.ID, user)
		if err := models.UnlinkStoryCourse(ctx, int32(s.ID), c.CourseID); err != nil {
			t.Fatal(err)
		}
		denied(t, s.ID, user)
	})

	t.Run("remove enrollment", func(t *testing.T) {
		fx := testdb.Fresh(t)
		s := singleCourseStory(t, fx.Data)
		user := paidStudent(fx.Data)
		cachedAllow(t, s.ID, user)
		if err := models.RemoveUserFromCourse(ctx, s.Courses[0], user); err != nil {
			t.Fatal(err)
		}
		denied(t, s.ID, user)
	})
}

// A recall submit writes every answer in one statement, whatever the number
// of sentences, and stores one row per sentence in the right table.
func TestRecallSubmitConstantCost(t *testing.T) {
	fx := testdb.Fresh(t)
	ctx := context.Background()
	user := paidStudent(fx.Data)

	// The story with the most recall sentences.
	bySentences := map[int][]int{}
	for _, r := range fx.Data.Table("recall_sentences").Rows {
		sid := int(r[1].(int32))
		bySentences[sid] = append(bySentences[sid], int(r[0].(int32)))
	}
	storyID, ids := 0, []int(nil)
	for _, s := range fx.Data.Stories {
		if len(bySentences[s.ID]) > len(ids) && !s.Trial {
			storyID, ids = s.ID, bySentences[s.ID]
		}
	}
	if len(ids) < 3 {
		t.Fatalf("seed has no story with 3+ recall sentences")
	}
	for _, tbl := range []string{"recall_correct_answers", "recall_incorrect_answers"} {
		if _, err := fx.Pool.Exec(ctx, "DELETE FROM "+tbl+" WHERE user_id = $1 AND story_id = $2", user, storyID); err != nil {
			t.Fatal(err)
		}
	}

	sentences, err := models.GetStoryRecallSentences(ctx, storyID)
	if err != nil {
		t.Fatal(err)
	}
	order := make(map[int]int, len(sentences))
	for _, s := range sentences {
		order[s.ID] = s.SequenceOrder
	}
	// Swap the first two: those two are wrong, the rest right.
	submitted := make([]int, len(sentences))
	for _, s := range sentences {
		submitted[s.SequenceOrder-1] = s.ID
	}
	submitted[0], submitted[1] = submitted[1], submitted[0]

	cctx, _ := database.WithQueryCounter(ctx)
	results, err := models.SaveRecallAttempt(cctx, user, storyID, submitted)
	if err != nil {
		t.Fatal(err)
	}
	// One read of the sentences, one insert.
	if n := database.QueryCount(cctx); n != 2 {
		t.Errorf("SaveRecallAttempt made %d queries for %d sentences, want 2", n, len(submitted))
	}

	var wantCorrect []int
	type wrong struct{ id, pos int }
	var wantWrong []wrong
	for i, id := range submitted {
		if want := order[id] == i+1; results[i] != want {
			t.Errorf("result[%d] = %v, want %v", i, results[i], want)
		}
		if order[id] == i+1 {
			wantCorrect = append(wantCorrect, id)
		} else {
			wantWrong = append(wantWrong, wrong{id, i + 1})
		}
	}

	rows, err := fx.Pool.Query(ctx, "SELECT recall_sentence_id FROM recall_correct_answers WHERE user_id = $1 AND story_id = $2 ORDER BY id", user, storyID)
	if err != nil {
		t.Fatal(err)
	}
	var gotCorrect []int
	for rows.Next() {
		var id int
		if err := rows.Scan(&id); err != nil {
			t.Fatal(err)
		}
		gotCorrect = append(gotCorrect, id)
	}
	rows.Close()
	if !slices.Equal(gotCorrect, wantCorrect) {
		t.Errorf("correct rows = %v, want %v", gotCorrect, wantCorrect)
	}

	rows, err = fx.Pool.Query(ctx, "SELECT recall_sentence_id, selected_position FROM recall_incorrect_answers WHERE user_id = $1 AND story_id = $2 ORDER BY id", user, storyID)
	if err != nil {
		t.Fatal(err)
	}
	var gotWrong []wrong
	for rows.Next() {
		var w wrong
		if err := rows.Scan(&w.id, &w.pos); err != nil {
			t.Fatal(err)
		}
		gotWrong = append(gotWrong, w)
	}
	rows.Close()
	if !slices.Equal(gotWrong, wantWrong) {
		t.Errorf("incorrect rows = %v, want %v", gotWrong, wantWrong)
	}
}

// PUT /api/stories/{id}/translate checks story access before writing.
func TestTranslatePutChecksAccess(t *testing.T) {
	ctx := context.Background()

	put := func(t *testing.T, storyID int, user string) (int, int) {
		t.Helper()
		req := httptest.NewRequest("PUT", fmt.Sprintf("/api/stories/%d/translate?lines=%%5B0%%5D", storyID), nil)
		req.Header.Set(testUserHeader, user)
		rr, n := testutil.ServeCounted(router, req)
		return rr.Code, n
	}
	rowsFor := func(t *testing.T, fx *testdb.Fixture, storyID int, user string) int {
		t.Helper()
		var n int
		if err := fx.Pool.QueryRow(ctx, "SELECT count(*) FROM translation_requests WHERE user_id = $1 AND story_id = $2", user, storyID).Scan(&n); err != nil {
			t.Fatal(err)
		}
		return n
	}

	t.Run("unenrolled gets 404 and nothing is written", func(t *testing.T) {
		fx := testdb.Fresh(t)
		fillSeedUsers(fx.Data)
		s := singleCourseStory(t, fx.Data)
		user := fx.Data.Students[seed.KindUnenrolled][0]
		if code, _ := put(t, s.ID, user); code != http.StatusNotFound {
			t.Errorf("status %d, want 404", code)
		}
		if n := rowsFor(t, fx, s.ID, user); n != 0 {
			t.Errorf("%d translation_requests rows written, want 0", n)
		}
	})

	t.Run("expired student with the paywall on gets 402", func(t *testing.T) {
		fx := testdb.Fresh(t)
		fillSeedUsers(fx.Data)
		t.Setenv("PAYWALL_ENABLED", "true")
		if models.PaymentsPaused() {
			t.Fatal("payments paused: the paywall would be lifted")
		}
		var s seed.Story
		for _, st := range fx.Data.Stories {
			if !st.Trial && st.Courses[0] == fx.Data.PaidCourses[0] {
				s = st
				break
			}
		}
		user := fx.Data.Students[seed.KindExpired][0]
		if code, _ := put(t, s.ID, user); code != http.StatusPaymentRequired {
			t.Errorf("status %d, want 402", code)
		}
		if n := rowsFor(t, fx, s.ID, user); n != 0 {
			t.Errorf("%d translation_requests rows written, want 0", n)
		}
	})

	t.Run("entitled student creates, merges and completes", func(t *testing.T) {
		fx := testdb.Fresh(t)
		fillSeedUsers(fx.Data)
		s := singleCourseStory(t, fx.Data)
		user := paidStudent(fx.Data)
		if _, err := fx.Pool.Exec(ctx, "DELETE FROM translation_requests WHERE user_id = $1 AND story_id = $2", user, s.ID); err != nil {
			t.Fatal(err)
		}
		if code, _ := put(t, s.ID, user); code != http.StatusOK {
			t.Errorf("first PUT: status %d, want 200", code)
		}
		req := httptest.NewRequest("PUT", fmt.Sprintf("/api/stories/%d/translate?lines=%%5B2,0,2%%5D&complete=true", s.ID), nil)
		req.Header.Set(testUserHeader, user)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Errorf("second PUT: status %d, want 200", rr.Code)
		}
		if n := rowsFor(t, fx, s.ID, user); n != 1 {
			t.Fatalf("%d translation_requests rows, want 1", n)
		}
		var lines []int32
		var completed bool
		if err := fx.Pool.QueryRow(ctx,
			"SELECT requested_lines, completed_at IS NOT NULL FROM translation_requests WHERE user_id = $1 AND story_id = $2",
			user, s.ID).Scan(&lines, &completed); err != nil {
			t.Fatal(err)
		}
		if !slices.Equal(lines, []int32{1, 3}) || !completed {
			t.Errorf("requested_lines = %v completed = %v, want [1 3] true", lines, completed)
		}
	})
}
