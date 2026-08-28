package models

import (
	"context"
	"errors"
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

func (f *fakeGrader) GradeProduce(_ context.Context, req ProduceGradeRequest) (ProduceGrade, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls = append(f.calls, req)
	return f.grade, f.err
}

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
