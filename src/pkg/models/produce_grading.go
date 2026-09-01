package models

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// ProduceGradingService grades submissions in the background after they are
// stored (SUMMER_2026.md T13). The student never waits on it and never sees a
// grading failure: on any error the submission simply stays ungraded
// (ai_score NULL) and the Score page treats it as pending.
//
// Because every grade is a paid API call, the service enforces a per-user
// quota on top of the global per-IP rate limiter. Over quota, the attempt is
// still stored — it just isn't graded — so a runaway client cannot run up the
// bill and a legitimate student is never blocked from finishing the phase.
type ProduceGradingService struct {
	grader ProduceGrader
	log    *slog.Logger
	quota  *userQuota

	// sem bounds concurrent grading calls so a burst of submissions cannot
	// open an unbounded number of API connections.
	sem chan struct{}
	// wg lets Close wait for in-flight grades (tests and graceful shutdown).
	wg sync.WaitGroup

	// timeout bounds one grading job, independent of any HTTP request.
	timeout time.Duration
	// now is swappable for tests.
	now func() time.Time
}

// Per-user grading quota: comfortably above legitimate use (two segments per
// story, one attempt each) while capping the cost one account can incur.
const (
	gradingPerMinute       = 10
	gradingPerDay          = 50
	gradingMaxConcurrent   = 4
	gradingJobTimeout      = 30 * time.Second
	gradingQuotaIdleExpiry = 48 * time.Hour
)

// NewProduceGradingService wires a grader into the background pipeline.
func NewProduceGradingService(grader ProduceGrader, logger *slog.Logger) *ProduceGradingService {
	return &ProduceGradingService{
		grader:  grader,
		log:     logger,
		quota:   newUserQuota(gradingPerMinute, gradingPerDay, gradingQuotaIdleExpiry),
		sem:     make(chan struct{}, gradingMaxConcurrent),
		timeout: gradingJobTimeout,
		now:     time.Now,
	}
}

// Enqueue schedules grading of a freshly stored submission. It returns
// immediately; the caller has already answered the student.
//
// Segment supplies the Hebrew, the reference English and the grammar point. A nil service
// (grading disabled) is safe to call.
func (s *ProduceGradingService) Enqueue(userID string, submission ProduceSubmission, segment ProduceSegment) {
	if s == nil {
		return
	}
	if !s.quota.allow(userID, s.now()) {
		s.log.Warn("Produce grading quota exceeded; leaving submission ungraded",
			"userID", userID, "submissionID", submission.ID)
		return
	}

	s.wg.Go(func() {
		s.sem <- struct{}{}
		defer func() { <-s.sem }()

		ctx, cancel := context.WithTimeout(context.Background(), s.timeout)
		defer cancel()
		if err := s.grade(ctx, submission, segment); err != nil {
			// Fail open: log and leave ai_score NULL.
			s.log.Error("Produce grading failed; submission left ungraded",
				"error", err, "userID", userID, "submissionID", submission.ID, "segmentID", segment.ID)
		}
	})
}

// Close waits for in-flight grading to finish.
func (s *ProduceGradingService) Close() {
	if s == nil {
		return
	}
	s.wg.Wait()
}

// grade produces and stores the verdict for one submission, and records the
// whole exchange in produce_grading_log whatever the outcome. Logging is
// best-effort: a failure to log never fails the grade.
func (s *ProduceGradingService) grade(ctx context.Context, submission ProduceSubmission, segment ProduceSegment) error {
	grade, trace, promptID, err := s.gradeAttempt(ctx, submission, segment)

	if logErr := LogProduceGrading(ctx, ProduceGradingLogEntry{
		Submission: submission,
		Segment:    segment,
		Grade:      grade,
		Trace:      trace,
		PromptID:   promptID,
		Err:        err,
	}); logErr != nil {
		s.log.Error("Failed to write produce grading log", "error", logErr, "submissionID", submission.ID)
	}

	if err != nil {
		return err
	}
	if err := GradeProduceSubmission(ctx, submission.ID, grade.Score, grade.Feedback); err != nil {
		return errors.Join(errors.New("store grade"), err)
	}
	return nil
}

// gradeAttempt decides the grade. A blank attempt (the timer ran out before
// the student wrote anything) is graded locally — there is nothing for the
// model to assess and no reason to pay for the call — and has an empty trace.
//
// The returned prompt ID is the produce_grading_prompts version the call ran
// with, or 0 for the built-in default (no version readable, or no call made).
func (s *ProduceGradingService) gradeAttempt(ctx context.Context, submission ProduceSubmission, segment ProduceSegment) (ProduceGrade, ProduceGradeTrace, int, error) {
	if strings.TrimSpace(submission.StudentText) == "" {
		return emptyAttemptGrade, ProduceGradeTrace{}, 0, nil
	}

	// The active prompt is read per run so an edit on the System page takes
	// effect immediately. Falling back to the default keeps grading alive if
	// the table is unreadable; the log then shows prompt_id NULL.
	prompt, err := GetActiveProduceGradingPrompt(ctx)
	if err != nil {
		if !errors.Is(err, ErrNotFound) {
			s.log.Warn("Could not read active grading prompt; using built-in default", "error", err)
		}
		prompt = defaultGradingPrompt()
	}

	req := ProduceGradeRequest{
		SystemPrompt:     prompt.Text,
		ReferenceEnglish: segment.ReferenceEnglish,
		HebrewText:       segment.HebrewText,
		StudentText:      submission.StudentText,
		GrammarPointName: segment.GrammarPointName,
	}
	if segment.GrammarPointID != nil {
		// The description gives the model the author's framing of the point;
		// it is context, not a requirement, so a lookup failure is not fatal.
		if gp, err := GetGrammarPoint(ctx, *segment.GrammarPointID); err == nil && gp != nil {
			req.GrammarPointDescription = gp.Description
			if req.GrammarPointName == "" {
				req.GrammarPointName = gp.Name
			}
		}
	}

	grade, trace, err := s.grader.GradeProduce(ctx, req)
	return grade, trace, prompt.ID, err
}

// emptyAttemptGrade is stored for blank submissions without an API call.
var emptyAttemptGrade = ProduceGrade{
	Score:    0,
	Feedback: "Nothing was written before the time ran out — next time, get a few words down early and build from there.",
}

// userQuota is a per-user limiter: a token bucket for the per-minute rate plus
// a rolling daily count. Idle users are evicted so the map cannot grow without
// bound (the leak developer_review.md #2 describes in the IP limiter).
type userQuota struct {
	mu         sync.Mutex
	perMinute  int
	perDay     int
	idleExpiry time.Duration
	users      map[string]*userQuotaEntry
	lastSweep  time.Time
}

type userQuotaEntry struct {
	minute   *rate.Limiter
	day      time.Time // UTC date the daily count belongs to
	dayCount int
	lastSeen time.Time
}

func newUserQuota(perMinute, perDay int, idleExpiry time.Duration) *userQuota {
	return &userQuota{
		perMinute:  perMinute,
		perDay:     perDay,
		idleExpiry: idleExpiry,
		users:      make(map[string]*userQuotaEntry),
	}
}

// allow consumes one grading slot for the user if both limits permit.
func (q *userQuota) allow(userID string, now time.Time) bool {
	q.mu.Lock()
	defer q.mu.Unlock()

	q.sweepLocked(now)

	e, ok := q.users[userID]
	if !ok {
		e = &userQuotaEntry{
			minute: rate.NewLimiter(rate.Every(time.Minute/time.Duration(q.perMinute)), q.perMinute),
		}
		q.users[userID] = e
	}
	e.lastSeen = now

	today := now.UTC().Truncate(24 * time.Hour)
	if !e.day.Equal(today) {
		e.day = today
		e.dayCount = 0
	}
	if e.dayCount >= q.perDay {
		return false
	}
	if !e.minute.AllowN(now, 1) {
		return false
	}
	e.dayCount++
	return true
}

// sweepLocked drops users not seen for idleExpiry, at most once a minute.
func (q *userQuota) sweepLocked(now time.Time) {
	if now.Sub(q.lastSweep) < time.Minute {
		return
	}
	q.lastSweep = now
	for id, e := range q.users {
		if now.Sub(e.lastSeen) > q.idleExpiry {
			delete(q.users, id)
		}
	}
}

// size reports tracked users, for tests.
func (q *userQuota) size() int {
	q.mu.Lock()
	defer q.mu.Unlock()
	return len(q.users)
}
