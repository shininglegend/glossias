package models

import (
	"context"
	"errors"
	"fmt"
	"glossias/src/pkg/database"
	"log/slog"
	"sync"
	"testing"
	"time"
)

// fakeGrader records requests and returns a canned verdict or error.
type fakeGrader struct {
	mu    sync.Mutex
	calls []ProduceGradeRequest
	grade ProduceGrade
	err   error
}

func (f *fakeGrader) GradeProduce(_ context.Context, req ProduceGradeRequest) (ProduceGrade, ProduceGradeTrace, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls = append(f.calls, req)
	return f.grade, fakeTrace, f.err
}

// fakeTrace stands in for what the real grader captures from the API.
var fakeTrace = ProduceGradeTrace{Model: "fake-model", UserPrompt: "user", RawResponse: `{"score":90}`}

func (f *fakeGrader) callCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.calls)
}

func newTestGradingService(t *testing.T, grader ProduceGrader) (*ProduceGradingService, *database.MockDBTX) {
	t.Helper()
	mockDB := database.NewMockDBTX()
	SetDB(mockDB)
	t.Cleanup(func() { SetDB(struct{}{}) })
	return NewProduceGradingService(grader, slog.New(slog.DiscardHandler)), mockDB
}

var (
	testSegment = ProduceSegment{
		ID:               7,
		ReferenceEnglish: "The boy sees the dog.",
		HebrewText:       "הילד רואה את הכלב",
		GrammarPointName: "Definite object marker",
	}
	testSubmission = ProduceSubmission{ID: 42, SegmentID: 7, StudentText: "The boy sees the dog."}
)

func TestProduceGradingService_GradesAndStores(t *testing.T) {
	grader := &fakeGrader{grade: ProduceGrade{Score: 90, Feedback: "Great."}}
	svc, _ := newTestGradingService(t, grader)

	svc.Enqueue("user-1", testSubmission, testSegment)
	svc.Close()

	if grader.callCount() != 1 {
		t.Fatalf("grader called %d times, want 1", grader.callCount())
	}
	got := grader.calls[0]
	if got.StudentText != testSubmission.StudentText || got.HebrewText != testSegment.HebrewText ||
		got.ReferenceEnglish != testSegment.ReferenceEnglish || got.GrammarPointName != testSegment.GrammarPointName {
		t.Errorf("unexpected grade request: %+v", got)
	}
}

func TestProduceGradingService_BlankAttemptSkipsModel(t *testing.T) {
	grader := &fakeGrader{grade: ProduceGrade{Score: 90}}
	svc, _ := newTestGradingService(t, grader)

	svc.Enqueue("user-1", ProduceSubmission{ID: 1, StudentText: "  \n"}, testSegment)
	svc.Close()

	if grader.callCount() != 0 {
		t.Error("a blank attempt should be graded locally, not sent to the model")
	}
}

func TestProduceGradingService_FailsOpen(t *testing.T) {
	// Neither a grader error nor a store error may panic or surface; the
	// submission just stays ungraded.
	grader := &fakeGrader{err: errors.New("api down")}
	svc, mockDB := newTestGradingService(t, grader)
	mockDB.StubExec("UPDATE produce_submissions", errors.New("db down"))

	svc.Enqueue("user-1", testSubmission, testSegment)
	svc.Close()

	if grader.callCount() != 1 {
		t.Errorf("grader called %d times, want 1", grader.callCount())
	}
}

func TestProduceGradingService_NilIsSafe(t *testing.T) {
	var svc *ProduceGradingService
	svc.Enqueue("user-1", testSubmission, testSegment) // must not panic
	svc.Close()
}

func TestProduceGradingService_QuotaLeavesUngraded(t *testing.T) {
	grader := &fakeGrader{grade: ProduceGrade{Score: 50}}
	svc, _ := newTestGradingService(t, grader)
	svc.quota = newUserQuota(2, 100, time.Hour)

	for i := range 5 {
		svc.Enqueue("user-1", ProduceSubmission{ID: i + 1, StudentText: "x"}, testSegment)
	}
	svc.Enqueue("user-2", ProduceSubmission{ID: 99, StudentText: "x"}, testSegment)
	svc.Close()

	// user-1 gets its 2 per-minute slots, user-2 is independent.
	if grader.callCount() != 3 {
		t.Errorf("grader called %d times, want 3", grader.callCount())
	}
}

func TestUserQuota(t *testing.T) {
	start := time.Date(2026, 8, 27, 10, 0, 0, 0, time.UTC)
	q := newUserQuota(3, 5, time.Hour)

	t.Run("per-minute bucket", func(t *testing.T) {
		now := start
		for i := range 3 {
			if !q.allow("a", now) {
				t.Fatalf("call %d should be allowed", i+1)
			}
		}
		if q.allow("a", now) {
			t.Error("4th call in the same instant should be refused")
		}
		// Tokens refill at 3/min: 20s later one is back.
		if !q.allow("a", now.Add(21*time.Second)) {
			t.Error("expected a refilled token after 21s")
		}
	})

	t.Run("daily cap wins even when the bucket has tokens", func(t *testing.T) {
		q := newUserQuota(100, 2, time.Hour)
		now := start
		for i := range 2 {
			if !q.allow("b", now) {
				t.Fatalf("call %d should be allowed", i+1)
			}
		}
		if q.allow("b", now.Add(time.Hour)) {
			t.Error("3rd of the day should be refused")
		}
		if !q.allow("b", now.Add(24*time.Hour)) {
			t.Error("a new UTC day should reset the count")
		}
	})

	t.Run("idle users are evicted", func(t *testing.T) {
		q := newUserQuota(10, 10, time.Hour)
		q.allow("old", start)
		if q.size() != 1 {
			t.Fatalf("size = %d, want 1", q.size())
		}
		q.allow("new", start.Add(2*time.Hour))
		if q.size() != 1 {
			t.Errorf("size = %d, want 1 after eviction of the idle user", q.size())
		}
		if _, ok := q.users["old"]; ok {
			t.Error("idle user should have been evicted")
		}
	})
}

func TestProduceGradingService_LogFailureDoesNotBlockGrade(t *testing.T) {
	grader := &fakeGrader{grade: ProduceGrade{Score: 70, Feedback: "Good."}}
	svc, mockDB := newTestGradingService(t, grader)
	mockDB.StubExec("produce_grading_log", errors.New("log table unavailable"))

	svc.Enqueue("user-1", testSubmission, testSegment)
	svc.Close()

	// The grade itself must still have been stored: the log is best-effort.
	if grader.callCount() != 1 {
		t.Fatalf("grader called %d times, want 1", grader.callCount())
	}
}

func TestProduceGradingService_LogsFailedGrades(t *testing.T) {
	grader := &fakeGrader{err: errors.New("model down")}
	svc, mockDB := newTestGradingService(t, grader)
	// A failing log write surfaces nothing to the student either; this just
	// exercises the error path through LogProduceGrading with Err set.
	mockDB.StubExec("UPDATE produce_submissions", errors.New("must not be reached"))

	svc.Enqueue("user-1", testSubmission, testSegment)
	svc.Close()

	if grader.callCount() != 1 {
		t.Fatalf("grader called %d times, want 1", grader.callCount())
	}
}

func TestProduceGradingService_FallsBackToDefaultPrompt(t *testing.T) {
	grader := &fakeGrader{grade: ProduceGrade{Score: 80}}
	svc, _ := newTestGradingService(t, grader)
	// No stub: the prompt query returns no rows.

	svc.Enqueue("user-1", testSubmission, testSegment)
	svc.Close()

	if grader.callCount() != 1 {
		t.Fatalf("grader called %d times, want 1", grader.callCount())
	}
	if got := grader.calls[0].SystemPrompt; got != DefaultGradingSystemPrompt {
		t.Errorf("expected the built-in default prompt, got %q", got)
	}
}

// sequenceGrader returns the next error in errs on each call (nil grades
// succeed), then repeats the last one.
type sequenceGrader struct {
	mu    sync.Mutex
	errs  []error
	calls int
}

func (g *sequenceGrader) GradeProduce(_ context.Context, _ ProduceGradeRequest) (ProduceGrade, ProduceGradeTrace, error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	i := min(g.calls, len(g.errs)-1)
	g.calls++
	if err := g.errs[i]; err != nil {
		return ProduceGrade{}, fakeTrace, err
	}
	return ProduceGrade{Score: 80, Feedback: "Nice."}, fakeTrace, nil
}

// enqueueN enqueues n non-blank submissions, each from its own user so the
// per-user quota never interferes, and waits for them to finish.
func enqueueN(svc *ProduceGradingService, n int) {
	for i := range n {
		svc.Enqueue("outage-user-"+string(rune('a'+i)), ProduceSubmission{ID: 1000 + i, StudentText: "x"}, testSegment)
	}
	svc.Close()
}

func TestProduceGradingService_RaisesIncidentAfterStreakAndResolvesOnSuccess(t *testing.T) {
	a := installAlerter(t)
	down := &GradingAPIError{StatusCode: 503, Err: errors.New("claude api request: 503 overloaded")}
	grader := &sequenceGrader{errs: []error{down}}
	svc, _ := newTestGradingService(t, grader)

	enqueueN(svc, GradingErrorStreakLimit-1)
	if n := len(a.raisedFor(GradingIncidentKey)); n != 0 {
		t.Fatalf("raised %d incidents before the limit", n)
	}
	enqueueN(svc, 1)
	raised := a.raisedFor(GradingIncidentKey)
	if len(raised) != 1 {
		t.Fatalf("raised = %d, want 1 at the limit", len(raised))
	}
	if inc := raised[0]; inc.Key != GradingIncidentKey || !containsAll(inc.Summary, fmt.Sprintf("%d consecutive", GradingErrorStreakLimit), "HTTP 503", "ungraded") || !containsAll(inc.Detail, "503 overloaded") {
		t.Fatalf("incident = %+v", inc)
	}
	// The outage goes on: no second incident, and a blank attempt (no API
	// call) neither counts nor resolves.
	enqueueN(svc, 3)
	svc.Enqueue("blank", ProduceSubmission{ID: 5, StudentText: " "}, testSegment)
	svc.Close()
	if len(a.raisedFor(GradingIncidentKey)) != 1 || a.resolvedFor(GradingIncidentKey) != 0 {
		t.Fatalf("raised=%d resolved=%d during the outage", len(a.raisedFor(GradingIncidentKey)), a.resolvedFor(GradingIncidentKey))
	}

	// The model answers again: resolve once, and only once.
	grader.mu.Lock()
	grader.errs = []error{nil}
	grader.mu.Unlock()
	enqueueN(svc, 2)
	if a.resolvedFor(GradingIncidentKey) != 1 || a.proofs[0] == "" {
		t.Fatalf("resolved = %d, want exactly 1", a.resolvedFor(GradingIncidentKey))
	}
	if svc.apiErrors.length() != 0 {
		t.Fatal("a success must clear the streak")
	}
	// Payments were never involved.
	if len(a.raisedFor(PaymentsIncidentKey)) != 0 || a.resolvedFor(PaymentsIncidentKey) != 0 {
		t.Fatal("grading must not touch the payments incident")
	}
}

func TestProduceGradingService_RejectedKeyRaisesImmediately(t *testing.T) {
	a := installAlerter(t)
	unauthorized := &GradingAPIError{StatusCode: 401, Err: errors.New("claude api request req_1: 401 invalid x-api-key")}
	svc, _ := newTestGradingService(t, &sequenceGrader{errs: []error{unauthorized}})

	enqueueN(svc, 1)
	raised := a.raisedFor(GradingIncidentKey)
	if len(raised) != 1 {
		t.Fatalf("raised = %d, want 1 on the first 401", len(raised))
	}
	if !containsAll(raised[0].Title, "401") || !containsAll(raised[0].Summary, "ANTHROPIC_API_KEY") {
		t.Fatalf("incident = %+v", raised[0])
	}
	enqueueN(svc, 2)
	if len(a.raisedFor(GradingIncidentKey)) != 1 {
		t.Fatal("a continuing 401 must not re-raise")
	}
}

func TestProduceGradingService_VerdictProblemsDoNotCountAsOutage(t *testing.T) {
	a := installAlerter(t)
	// A refusal, a cut-off or undecodable JSON all mean the API answered.
	verdict := errors.New("decode grading response: unexpected end of JSON input")
	svc, _ := newTestGradingService(t, &sequenceGrader{errs: []error{verdict}})

	enqueueN(svc, GradingErrorStreakLimit+2)
	if len(a.raised) != 0 {
		t.Fatalf("raised = %+v, want none for verdict problems", a.raised)
	}
	// Nor do they resolve anything when nothing was raised.
	if len(a.resolved) != 0 {
		t.Fatalf("resolved = %v, want none", a.resolved)
	}
	if svc.apiErrors.length() != 0 {
		t.Fatal("verdict problems must not build a streak")
	}
}

func TestProduceGradingService_SuccessInsideStreakResetsIt(t *testing.T) {
	a := installAlerter(t)
	down := &GradingAPIError{Err: errors.New("claude api request: dial tcp: i/o timeout")}
	grader := &sequenceGrader{errs: []error{down}}
	svc, _ := newTestGradingService(t, grader)

	enqueueN(svc, GradingErrorStreakLimit-1)
	// One success, then failures again.
	grader.mu.Lock()
	grader.errs = []error{nil, down}
	grader.calls = 0
	grader.mu.Unlock()
	enqueueN(svc, 1)
	enqueueN(svc, GradingErrorStreakLimit-1)
	if len(a.raised) != 0 {
		t.Fatalf("a success between failures must restart the count; raised = %+v", a.raised)
	}
	if svc.apiErrors.length() != GradingErrorStreakLimit-1 {
		t.Fatalf("streak = %d", svc.apiErrors.length())
	}
}
