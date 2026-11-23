//go:build integration

package integration

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"glossias/src/pkg/testdb/seed"
)

// Case is one measured request.
type Case struct {
	// Name is the stable key in the baseline file: group/endpoint/who[/variant].
	Name   string
	Method string
	Path   string
	As     string // seeded user id; "" sends no user
	// Body is sent as JSON. Prepare, when set, builds the body instead; it
	// runs unmeasured right before the request (e.g. a GET to round-trip).
	Body    any
	Prepare func(t *testing.T) any
	// Paywall turns PAYWALL_ENABLED on for this case.
	Paywall bool
	// Mutates runs the case on a fresh clone and skips the warm measurement.
	Mutates bool
	// Want is the expected status; 0 accepts anything below 500.
	Want int
	// KnownBug documents a known server error; the case is still measured
	// and its status recorded, but a 5xx does not fail it.
	KnownBug string
}

// fixtures are the seeded rows the cases are built around.
type fixtures struct {
	ds *seed.Dataset

	paid, expired, unpaid, trial, unenrolled string
	super                                    string
	admin                                    string // course admin of story's owner course

	story     seed.Story // paid story the paid student has progress and an archived attempt on
	course1   seed.Story // a paid story in the first paid course (expired/unpaid students' course)
	trialSt   seed.Story
	ownCourse int

	recallIDs                []int
	targetID, targetLine     int
	targetOther              int
	vocabLineIdx             int
	vocabAnswer              string
	grammarPoint, grammarLn  int
	grammarPos, produceSegID int
}

func newFixtures(t *testing.T, ds *seed.Dataset) *fixtures {
	t.Helper()
	f := &fixtures{ds: ds, super: ds.SuperAdmin}
	f.expired = ds.Students[seed.KindExpired][0]
	f.unpaid = ds.Students[seed.KindUnpaid][0]
	f.trial = ds.Students[seed.KindTrial][0]
	f.unenrolled = ds.Students[seed.KindUnenrolled][0]
	for _, u := range ds.Students[seed.KindPaid] {
		if len(ds.Archived[u]) > 0 {
			f.paid = u
			f.story = storyByID(ds, ds.Archived[u][0])
			break
		}
	}
	if f.paid == "" {
		t.Fatal("seed has no paid student with an archived attempt")
	}
	f.ownCourse = f.story.Courses[0]
	f.admin = ds.CourseAdmins[f.ownCourse]
	for _, s := range ds.Stories {
		if !s.Trial && s.Courses[0] == ds.PaidCourses[0] {
			f.course1 = s
			break
		}
	}
	f.trialSt, _ = ds.TrialStory()

	sid := int32(f.story.ID)
	for _, r := range ds.Table("recall_sentences").Rows {
		if r[1].(int32) == sid {
			f.recallIDs = append(f.recallIDs, int(r[0].(int32)))
		}
	}
	forms := map[string]int{}
	for _, r := range ds.Table("vocabulary_items").Rows {
		if r[1].(int32) != sid {
			continue
		}
		if f.vocabAnswer == "" {
			f.vocabLineIdx = int(r[2].(int32)) - 1
			f.vocabAnswer = r[4].(string)
		}
		if _, ok := forms[r[4].(string)]; !ok {
			forms[r[4].(string)] = int(r[2].(int32))
		}
	}
	for _, r := range ds.Table("target_vocabulary").Rows {
		if r[1].(int32) != sid {
			continue
		}
		if f.targetID == 0 {
			f.targetID = int(r[0].(int32))
			f.targetLine = forms[r[2].(string)]
		} else if f.targetOther == 0 {
			f.targetOther = int(r[0].(int32))
		}
	}
	for _, r := range ds.Table("grammar_items").Rows {
		if r[1].(int32) == sid {
			f.grammarLn, f.grammarPoint, f.grammarPos = int(r[2].(int32)), int(r[3].(int32)), int(r[5].(int32))
			break
		}
	}
	for _, r := range ds.Table("produce_segments").Rows {
		if r[1].(int32) == sid {
			f.produceSegID = int(r[0].(int32))
			break
		}
	}
	return f
}

func storyByID(ds *seed.Dataset, id int) seed.Story {
	for _, s := range ds.Stories {
		if s.ID == id {
			return s
		}
	}
	panic(fmt.Sprintf("no story %d", id))
}

// cases lists every measured request. Names are the baseline keys: keep them
// stable, and add new cases rather than renaming.
func (f *fixtures) cases() []Case {
	s := fmt.Sprintf("/api/stories/%d", f.story.ID)
	adm := fmt.Sprintf("/api/admin/stories/%d", f.story.ID)
	var cs []Case
	add := func(c ...Case) { cs = append(cs, c...) }

	// Story list: every paywall situation, with the paywall off and on.
	for _, who := range []struct{ name, id string }{
		{"paid", f.paid}, {"expired", f.expired}, {"unpaid", f.unpaid},
		{"trial", f.trial}, {"unenrolled", f.unenrolled},
		{"course-admin", f.admin}, {"super-admin", f.super},
	} {
		add(
			Case{Name: "list/stories/" + who.name + "/paywall-off", Method: "GET", Path: "/api/stories", As: who.id, Want: 200},
			Case{Name: "list/stories/" + who.name + "/paywall-on", Method: "GET", Path: "/api/stories", As: who.id, Paywall: true, Want: 200},
		)
	}
	add(
		Case{Name: "list/by-course/paid", Method: "GET", Path: fmt.Sprintf("/api/stories/by-course/%d", f.ownCourse), As: f.paid, Want: 200},
		Case{Name: "list/by-course/trial", Method: "GET", Path: fmt.Sprintf("/api/stories/by-course/%d", f.trialSt.Courses[0]), As: f.trial, Want: 200},
		Case{Name: "user/me/paid", Method: "GET", Path: "/api/me", As: f.paid, Want: 200},
		Case{Name: "user/me/course-admin", Method: "GET", Path: "/api/me", As: f.admin, Want: 200},
		Case{Name: "user/pricing/unpaid", Method: "GET", Path: "/api/pricing", As: f.unpaid},
	)

	// Story pages, as the paid student with the paywall off and on.
	pages := []string{"metadata", "story-with-audio", "vocab", "grammar", "translate", "identify", "recall", "produce", "scores"}
	for _, p := range pages {
		add(
			Case{Name: "page/" + p + "/paid/paywall-off", Method: "GET", Path: s + "/" + p, As: f.paid},
			Case{Name: "page/" + p + "/paid/paywall-on", Method: "GET", Path: s + "/" + p, As: f.paid, Paywall: true},
		)
	}
	c1 := fmt.Sprintf("/api/stories/%d", f.course1.ID)
	add(
		Case{Name: "page/vocab/expired/paywall-on", Method: "GET", Path: c1 + "/vocab", As: f.expired, Paywall: true, Want: http.StatusPaymentRequired},
		Case{Name: "page/vocab/unpaid/paywall-on", Method: "GET", Path: c1 + "/vocab", As: f.unpaid, Paywall: true, Want: http.StatusPaymentRequired},
		Case{Name: "page/vocab/unenrolled", Method: "GET", Path: c1 + "/vocab", As: f.unenrolled, Want: http.StatusNotFound},
		Case{Name: "page/vocab/trial/paywall-on", Method: "GET", Path: fmt.Sprintf("/api/stories/%d/vocab", f.trialSt.ID), As: f.trial, Paywall: true, Want: 200},
		Case{Name: "page/identify/course-admin", Method: "GET", Path: s + "/identify", As: f.admin},
	)

	// Student writes.
	add(
		Case{Name: "write/check-vocab/paid", Method: "POST", Path: s + "/check-vocab", As: f.paid, Mutates: true,
			Body: map[string]any{"vocab_key": fmt.Sprintf("%d-0", f.vocabLineIdx), "answer": f.vocabAnswer}},
		Case{Name: "write/check-grammar/paid", Method: "POST", Path: s + "/check-grammar", As: f.paid, Mutates: true,
			Body: map[string]any{"grammar_point_id": f.grammarPoint, "line_number": f.grammarLn - 1, "position": f.grammarPos}},
		Case{Name: "write/check-identify/paid", Method: "POST", Path: s + "/check-identify", As: f.paid, Mutates: true,
			Body: map[string]any{"line_index": f.targetLine - 1, "target_vocab_id": f.targetID, "selected_target_vocab_id": f.targetOther}},
		Case{Name: "write/check-recall/paid", Method: "POST", Path: s + "/check-recall", As: f.paid, Mutates: true,
			Body: map[string]any{"ordered_sentence_ids": f.recallIDs}},
		Case{Name: "write/check-recall-pick/paid", Method: "POST", Path: s + "/check-recall-pick", As: f.paid, Mutates: true,
			Body: map[string]any{"sentence_id": f.recallIDs[0], "position": 1}},
		// Line indexes go in ?lines=[...] (0-based).
		Case{Name: "write/translate/paid", Method: "PUT", Path: s + "/translate?lines=%5B3%5D", As: f.paid, Mutates: true, Want: 200},
		Case{Name: "write/translate/unenrolled", Method: "PUT", Path: s + "/translate?lines=%5B3%5D", As: f.unenrolled, Mutates: true, Want: http.StatusNotFound},
		Case{Name: "write/next/paid", Method: "POST", Path: s + "/next", As: f.paid, Mutates: true,
			Body: map[string]any{"currentPage": "vocab"}},
		Case{Name: "write/produce-start/paid", Method: "POST", Path: s + "/produce/start", As: f.paid, Mutates: true,
			Body: map[string]any{"segment_id": f.produceSegID}},
		Case{Name: "write/produce-submit/paid", Method: "POST", Path: s + "/produce", As: f.paid, Mutates: true,
			Body: map[string]any{"segment_id": f.produceSegID, "student_text": "A seeded attempt."}},
		Case{Name: "write/time-tracking-start/paid", Method: "POST", Path: "/api/time-tracking/start", As: f.paid, Mutates: true,
			Body: map[string]any{"route": fmt.Sprintf("/stories/%d/vocab", f.story.ID), "story_id": f.story.ID}},
	)

	// Admin reads, as the owning course admin; the heavy ones as super admin too.
	cq := fmt.Sprintf("?course_id=%d", f.ownCourse)
	for _, p := range []string{"", "/metadata", "/courses", "/annotations", "/translations", "/content-readiness",
		"/target-vocabulary", "/produce", "/recall", "/students" + cq,
		"/students/" + f.paid + cq, "/students/" + f.paid + "/attempts" + cq} {
		key := strings.TrimPrefix(p, "/")
		key = strings.TrimSuffix(key, cq)
		key = strings.ReplaceAll(key, f.paid, "student")
		switch key {
		case "":
			key = "story"
		case "courses":
			key = "story-courses"
		}
		add(Case{Name: "admin/" + key + "/course-admin", Method: "GET", Path: adm + p, As: f.admin, Want: 200})
	}
	add(
		Case{Name: "admin/story/super-admin", Method: "GET", Path: adm, As: f.super, Want: 200},
		Case{Name: "admin/annotations/super-admin", Method: "GET", Path: adm + "/annotations", As: f.super, Want: 200},
		Case{Name: "admin/annotations-line/course-admin", Method: "GET", Path: adm + "/annotations?line=" + fmt.Sprint(f.story.Lines/2), As: f.admin},
		Case{Name: "admin/story/student-denied", Method: "GET", Path: adm, As: f.paid, Want: http.StatusForbidden},
		Case{Name: "admin/courses/super-admin", Method: "GET", Path: "/api/admin/courses", As: f.super, Want: 200},
		Case{Name: "admin/courses/course-admin", Method: "GET", Path: "/api/admin/courses", As: f.admin, Want: 200},
		Case{Name: "admin/course/course-admin", Method: "GET", Path: fmt.Sprintf("/api/admin/courses/%d", f.ownCourse), As: f.admin, Want: 200},
		Case{Name: "admin/course-admins/super-admin", Method: "GET", Path: fmt.Sprintf("/api/admin/courses/%d/admins", f.ownCourse), As: f.super, Want: 200},
		Case{Name: "admin/course-sections/course-admin", Method: "GET", Path: fmt.Sprintf("/api/admin/courses/%d/sections", f.ownCourse), As: f.admin},
		Case{Name: "admin/course-users/course-admin", Method: "GET", Path: fmt.Sprintf("/api/admin/course-users/%d", f.ownCourse), As: f.admin, Want: 200},
		Case{Name: "admin/course-users/super-admin", Method: "GET", Path: fmt.Sprintf("/api/admin/course-users/%d", f.ownCourse), As: f.super, Want: 200},
		Case{Name: "admin/system-payments/super-admin", Method: "GET", Path: "/api/admin/system/payments", As: f.super, Want: 200},
		Case{Name: "admin/system-grading-prompt/super-admin", Method: "GET", Path: "/api/admin/system/grading-prompt", As: f.super, Want: 200},
	)

	// Admin writes.
	add(
		Case{Name: "admin-write/save-story/course-admin", Method: "PUT", Path: adm, As: f.admin, Mutates: true,
			KnownBug: "SaveStoryData inserts story_lines without deleting or upserting, so any story with lines fails on story_lines_pkey",
			Prepare:  func(t *testing.T) any { return getJSON(t, adm, f.admin, "story") }},
		Case{Name: "admin-write/bulk-translations/course-admin", Method: "PUT", Path: adm + "/translations", As: f.admin, Mutates: true,
			Body: map[string]any{"languageCode": "es", "translations": lineTranslations(f.story.Lines)}},
		Case{Name: "admin-write/link-course/super-admin", Method: "POST", Path: adm + "/courses", As: f.super, Mutates: true,
			Body: map[string]any{"courseId": f.trialSt.Courses[0]}},
		Case{Name: "admin-write/reset-progress/course-admin", Method: "DELETE", Path: adm + "/students/" + f.paid + "/progress?phase=all", As: f.admin, Mutates: true},
		Case{Name: "admin-write/delete-attempt/course-admin", Method: "DELETE", Path: adm + "/students/" + f.paid + "/attempts/1", As: f.admin, Mutates: true},
		Case{Name: "admin-write/add-course-users/course-admin", Method: "POST", Path: fmt.Sprintf("/api/admin/course-users/%d", f.ownCourse), As: f.admin, Mutates: true,
			Body: map[string]any{"emails": f.unenrolledEmails()}, Want: 200},
		Case{Name: "admin-write/grant-access/super-admin", Method: "POST", Path: "/api/admin/users/" + f.unpaid + "/access", As: f.super, Mutates: true,
			Body: map[string]any{"course_id": f.ownCourse}},
	)
	return cs
}

// unenrolledEmails are the emails of every unenrolled student, for bulk
// enrollment.
func (f *fixtures) unenrolledEmails() []string {
	var out []string
	for _, id := range f.ds.Students[seed.KindUnenrolled] {
		out = append(out, seedUsers[id][0])
	}
	return out
}

func lineTranslations(n int) []map[string]any {
	out := make([]map[string]any, n)
	for i := range out {
		out[i] = map[string]any{"lineNumber": i + 1, "translation": fmt.Sprintf("Linea %d", i+1)}
	}
	return out
}

// getJSON fetches path as user (unmeasured) and returns the decoded body's
// envelope field (e.g. "story" of {"story": ..., "success": true}).
func getJSON(t *testing.T, path, user, field string) any {
	t.Helper()
	req := httptest.NewRequest("GET", path, nil)
	req.Header.Set(testUserHeader, user)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != 200 {
		t.Fatalf("prepare GET %s: %d %s", path, rr.Code, rr.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("prepare GET %s: %v", path, err)
	}
	d, ok := body[field]
	if !ok {
		t.Fatalf("prepare GET %s: no %q in response", path, field)
	}
	return d
}
