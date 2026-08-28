package models

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"glossias/src/pkg/generated/db"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

// ProduceSegmentsPerStory is the number of Produce segments each story must
// have.
const ProduceSegmentsPerStory = 2

// ProduceSegment is one Hebrew passage, drawn from the story, that the student
// renders into English during the Produce phase, together with the reference
// English translation revealed afterwards.
type ProduceSegment struct {
	ID               int    `json:"id"`
	StoryID          int    `json:"storyId"`
	SegmentOrder     int    `json:"segmentOrder"`
	ReferenceEnglish string `json:"referenceEnglish"`
	HebrewText       string `json:"hebrewText"`
	GrammarPointID   *int   `json:"grammarPointId,omitempty"`
	GrammarPointName string `json:"grammarPointName,omitempty"`
	// LineStart and LineEnd are the 1-based story lines the Hebrew was
	// drawn from (inclusive; equal for a single line), so the student page
	// can mark the segment's slot in the text and the editor can re-seed the
	// segment from the same lines later. Nil when the author has not placed
	// it.
	LineStart *int `json:"lineStart,omitempty"`
	LineEnd   *int `json:"lineEnd,omitempty"`
}

// ProduceAttemptStart is when a student began writing a segment, with the
// elapsed time computed by the database.
type ProduceAttemptStart struct {
	SegmentID      int       `json:"segmentId"`
	StartedAt      time.Time `json:"startedAt"`
	ElapsedSeconds int       `json:"elapsedSeconds"`
}

func optionalInt(v pgtype.Int4) *int {
	if !v.Valid {
		return nil
	}
	n := int(v.Int32)
	return &n
}

func toPgInt4(v *int) pgtype.Int4 {
	if v == nil {
		return pgtype.Int4{}
	}
	return pgtype.Int4{Int32: int32(*v), Valid: true}
}

// ProduceSubmission is a student's attempt at one segment. AiScore is nil while
// the attempt is ungraded — including when grading failed, so that a grading
// outage never blocks the student.
type ProduceSubmission struct {
	ID           int        `json:"id"`
	UserID       string     `json:"userId"`
	StoryID      int        `json:"storyId"`
	SegmentID    int        `json:"segmentId"`
	SegmentOrder int        `json:"segmentOrder,omitempty"`
	StudentText  string     `json:"studentText"`
	AiScore      *int       `json:"aiScore,omitempty"`
	AiFeedback   string     `json:"aiFeedback,omitempty"`
	GradedAt     *time.Time `json:"gradedAt,omitempty"`
	CreatedAt    *time.Time `json:"createdAt,omitempty"`
}

// ProduceSummary aggregates a user's latest submission per segment for scoring.
// The counts are per segment, not per raw submission: SegmentsSubmitted reaching
// ProduceSegmentsPerStory is what marks the phase complete.
type ProduceSummary struct {
	SegmentsSubmitted int     `json:"segmentsSubmitted"`
	SegmentsGraded    int     `json:"segmentsGraded"`
	AverageScore      float64 `json:"averageScore"`
}

// GetStoryProduceSegments returns a story's segments in presentation order.
func GetStoryProduceSegments(ctx context.Context, storyID int) ([]ProduceSegment, error) {
	if queries == nil {
		return nil, errors.New("database not initialized")
	}

	rows, err := queries.GetStoryProduceSegments(ctx, int32(storyID))
	if err != nil {
		return nil, err
	}

	segments := make([]ProduceSegment, 0, len(rows))
	for _, row := range rows {
		segment := ProduceSegment{
			ID:               int(row.ID),
			StoryID:          int(row.StoryID),
			SegmentOrder:     int(row.SegmentOrder),
			ReferenceEnglish: row.ReferenceEnglish,
			HebrewText:       row.HebrewText,
			GrammarPointName: row.GrammarPointName.String,
			GrammarPointID:   optionalInt(row.GrammarPointID),
			LineStart:        optionalInt(row.LineStart),
			LineEnd:          optionalInt(row.LineEnd),
		}
		segments = append(segments, segment)
	}

	return segments, nil
}

// GetProduceSegment retrieves a single segment by ID.
func GetProduceSegment(ctx context.Context, id int) (*ProduceSegment, error) {
	if queries == nil {
		return nil, errors.New("database not initialized")
	}

	row, err := queries.GetProduceSegment(ctx, int32(id))
	if errors.Is(err, sql.ErrNoRows) || errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	segment := &ProduceSegment{
		ID:               int(row.ID),
		StoryID:          int(row.StoryID),
		SegmentOrder:     int(row.SegmentOrder),
		ReferenceEnglish: row.ReferenceEnglish,
		HebrewText:       row.HebrewText,
		GrammarPointName: row.GrammarPointName.String,
		GrammarPointID:   optionalInt(row.GrammarPointID),
		LineStart:        optionalInt(row.LineStart),
		LineEnd:          optionalInt(row.LineEnd),
	}

	return segment, nil
}

// UpsertProduceSegment creates or replaces the segment at a story's given
// order slot.
func UpsertProduceSegment(ctx context.Context, segment ProduceSegment) (*ProduceSegment, error) {
	if queries == nil {
		return nil, errors.New("database not initialized")
	}

	row, err := queries.UpsertProduceSegment(ctx, db.UpsertProduceSegmentParams{
		StoryID:          int32(segment.StoryID),
		SegmentOrder:     int32(segment.SegmentOrder),
		ReferenceEnglish: segment.ReferenceEnglish,
		HebrewText:       segment.HebrewText,
		GrammarPointID:   toPgInt4(segment.GrammarPointID),
		LineStart:        toPgInt4(segment.LineStart),
		LineEnd:          toPgInt4(segment.LineEnd),
	})
	if err != nil {
		return nil, err
	}

	saved := &ProduceSegment{
		ID:               int(row.ID),
		StoryID:          int(row.StoryID),
		SegmentOrder:     int(row.SegmentOrder),
		ReferenceEnglish: row.ReferenceEnglish,
		HebrewText:       row.HebrewText,
		GrammarPointID:   optionalInt(row.GrammarPointID),
		LineStart:        optionalInt(row.LineStart),
		LineEnd:          optionalInt(row.LineEnd),
	}

	InvalidateStoryContentReadiness(saved.StoryID)
	return saved, nil
}

// DeleteProduceSegment removes a single segment.
func DeleteProduceSegment(ctx context.Context, storyID, id int) error {
	if queries == nil {
		return errors.New("database not initialized")
	}
	if err := queries.DeleteProduceSegment(ctx, int32(id)); err != nil {
		return err
	}
	InvalidateStoryContentReadiness(storyID)
	return nil
}

// DeleteStoryProduceSegments removes every segment for a story.
func DeleteStoryProduceSegments(ctx context.Context, storyID int) error {
	if queries == nil {
		return errors.New("database not initialized")
	}
	if err := queries.DeleteStoryProduceSegments(ctx, int32(storyID)); err != nil {
		return err
	}
	InvalidateStoryContentReadiness(storyID)
	return nil
}

// CountStoryProduceSegments returns how many segments a story has.
func CountStoryProduceSegments(ctx context.Context, storyID int) (int, error) {
	if queries == nil {
		return 0, errors.New("database not initialized")
	}

	count, err := queries.CountStoryProduceSegments(ctx, int32(storyID))
	if err != nil {
		return 0, err
	}
	return int(count), nil
}

// GetStoryProduceExplanation returns the contrastive grammar explanation shown
// after both segments, or ErrNotFound when none has been authored.
func GetStoryProduceExplanation(ctx context.Context, storyID int) (string, error) {
	if queries == nil {
		return "", errors.New("database not initialized")
	}

	row, err := queries.GetStoryProduceExplanation(ctx, int32(storyID))
	if errors.Is(err, sql.ErrNoRows) || errors.Is(err, pgx.ErrNoRows) {
		return "", ErrNotFound
	}
	if err != nil {
		return "", err
	}

	return row.ExplanationText, nil
}

// UpsertStoryProduceExplanation creates or replaces a story's explanation.
func UpsertStoryProduceExplanation(ctx context.Context, storyID int, explanation string) error {
	if queries == nil {
		return errors.New("database not initialized")
	}

	_, err := queries.UpsertStoryProduceExplanation(ctx, db.UpsertStoryProduceExplanationParams{
		StoryID:         int32(storyID),
		ExplanationText: explanation,
	})
	if err != nil {
		return err
	}
	InvalidateStoryContentReadiness(storyID)
	return nil
}

// DeleteStoryProduceExplanation removes a story's explanation.
func DeleteStoryProduceExplanation(ctx context.Context, storyID int) error {
	if queries == nil {
		return errors.New("database not initialized")
	}
	if err := queries.DeleteStoryProduceExplanation(ctx, int32(storyID)); err != nil {
		return err
	}
	InvalidateStoryContentReadiness(storyID)
	return nil
}

// CountStoryLines returns how many lines a story has, for validating a
// segment's line number.
func CountStoryLines(ctx context.Context, storyID int) (int, error) {
	if queries == nil {
		return 0, errors.New("database not initialized")
	}
	lines, err := queries.GetStoryLines(ctx, int32(storyID))
	if err != nil {
		return 0, err
	}
	return len(lines), nil
}

// StartProduceAttempt records that the user began writing a segment and
// returns the start — the original one if the segment was already started, so
// a reload cannot reset the countdown.
func StartProduceAttempt(ctx context.Context, userID string, storyID, segmentID int) (ProduceAttemptStart, error) {
	if queries == nil {
		return ProduceAttemptStart{}, errors.New("database not initialized")
	}

	row, err := queries.StartProduceAttempt(ctx, db.StartProduceAttemptParams{
		UserID:    userID,
		StoryID:   int32(storyID),
		SegmentID: int32(segmentID),
	})
	if err != nil {
		return ProduceAttemptStart{}, err
	}
	return ProduceAttemptStart{
		SegmentID:      int(row.SegmentID),
		StartedAt:      row.StartedAt.Time,
		ElapsedSeconds: int(row.ElapsedSeconds),
	}, nil
}

// GetUserStoryProduceAttemptStarts returns every segment the user has started
// in a story, with the time elapsed since.
func GetUserStoryProduceAttemptStarts(ctx context.Context, userID string, storyID int) ([]ProduceAttemptStart, error) {
	if queries == nil {
		return nil, errors.New("database not initialized")
	}

	rows, err := queries.GetUserStoryProduceAttemptStarts(ctx, db.GetUserStoryProduceAttemptStartsParams{
		UserID:  userID,
		StoryID: int32(storyID),
	})
	if err != nil {
		return nil, err
	}

	starts := make([]ProduceAttemptStart, 0, len(rows))
	for _, row := range rows {
		starts = append(starts, ProduceAttemptStart{
			SegmentID:      int(row.SegmentID),
			StartedAt:      row.StartedAt.Time,
			ElapsedSeconds: int(row.ElapsedSeconds),
		})
	}
	return starts, nil
}

// CreateProduceSubmission stores an ungraded student attempt and returns it so
// the caller can grade it afterwards.
func CreateProduceSubmission(ctx context.Context, userID string, storyID, segmentID int, studentText string) (*ProduceSubmission, error) {
	if queries == nil {
		return nil, errors.New("database not initialized")
	}

	row, err := queries.CreateProduceSubmission(ctx, db.CreateProduceSubmissionParams{
		UserID:      userID,
		StoryID:     int32(storyID),
		SegmentID:   int32(segmentID),
		StudentText: studentText,
	})
	if err != nil {
		return nil, err
	}

	submission := &ProduceSubmission{
		ID:          int(row.ID),
		UserID:      row.UserID,
		StoryID:     int(row.StoryID),
		SegmentID:   int(row.SegmentID),
		StudentText: row.StudentText,
		AiFeedback:  row.AiFeedback.String,
	}
	if row.AiScore.Valid {
		score := int(row.AiScore.Int32)
		submission.AiScore = &score
	}
	if row.GradedAt.Valid {
		gradedAt := row.GradedAt.Time
		submission.GradedAt = &gradedAt
	}
	if row.CreatedAt.Valid {
		createdAt := row.CreatedAt.Time
		submission.CreatedAt = &createdAt
	}

	return submission, nil
}

// GradeProduceSubmission attaches an AI score and feedback to a submission.
func GradeProduceSubmission(ctx context.Context, submissionID, score int, feedback string) error {
	if queries == nil {
		return errors.New("database not initialized")
	}

	return queries.GradeProduceSubmission(ctx, db.GradeProduceSubmissionParams{
		ID:         int32(submissionID),
		AiScore:    pgtype.Int4{Int32: int32(score), Valid: true},
		AiFeedback: optionalText(feedback),
	})
}

// GetUserStoryProduceSubmissions returns the user's latest submission per
// segment for a story.
func GetUserStoryProduceSubmissions(ctx context.Context, userID string, storyID int) ([]ProduceSubmission, error) {
	if queries == nil {
		return nil, errors.New("database not initialized")
	}

	rows, err := queries.GetUserStoryProduceSubmissions(ctx, db.GetUserStoryProduceSubmissionsParams{
		UserID:  userID,
		StoryID: int32(storyID),
	})
	if err != nil {
		return nil, err
	}

	submissions := make([]ProduceSubmission, 0, len(rows))
	for _, row := range rows {
		submission := ProduceSubmission{
			ID:           int(row.ID),
			UserID:       row.UserID,
			StoryID:      int(row.StoryID),
			SegmentID:    int(row.SegmentID),
			SegmentOrder: int(row.SegmentOrder),
			StudentText:  row.StudentText,
			AiFeedback:   row.AiFeedback.String,
		}
		if row.AiScore.Valid {
			score := int(row.AiScore.Int32)
			submission.AiScore = &score
		}
		if row.GradedAt.Valid {
			gradedAt := row.GradedAt.Time
			submission.GradedAt = &gradedAt
		}
		if row.CreatedAt.Valid {
			createdAt := row.CreatedAt.Time
			submission.CreatedAt = &createdAt
		}
		submissions = append(submissions, submission)
	}

	return submissions, nil
}

// GetUserStoryProduceSummary aggregates the user's latest submission per
// segment. AverageScore covers only graded submissions.
func GetUserStoryProduceSummary(ctx context.Context, userID string, storyID int) (ProduceSummary, error) {
	if queries == nil {
		return ProduceSummary{}, errors.New("database not initialized")
	}

	row, err := queries.GetUserStoryProduceSummary(ctx, db.GetUserStoryProduceSummaryParams{
		UserID:  userID,
		StoryID: int32(storyID),
	})
	if err != nil {
		return ProduceSummary{}, err
	}

	return ProduceSummary{
		SegmentsSubmitted: int(row.SegmentsSubmitted),
		SegmentsGraded:    int(row.SegmentsGraded),
		AverageScore:      row.AverageScore,
	}, nil
}

// ProduceGradingLogEntry is one grading run to record: the submission and
// segment it was for, and either the verdict or the error, with the trace of
// what went to the model.
type ProduceGradingLogEntry struct {
	Submission ProduceSubmission
	Segment    ProduceSegment
	Grade      ProduceGrade
	Trace      ProduceGradeTrace
	// PromptID is the produce_grading_prompts version used; 0 (stored NULL)
	// means the built-in default, or no model call.
	PromptID int
	Err      error
}

// LogProduceGrading appends a row to produce_grading_log. On error Score is
// left NULL and the message stored; a blank-attempt grade (no model call)
// has no model, prompts or response.
func LogProduceGrading(ctx context.Context, e ProduceGradingLogEntry) error {
	if queries == nil {
		return errors.New("database not initialized")
	}

	params := db.InsertProduceGradingLogParams{
		SubmissionID:         int32(e.Submission.ID),
		UserID:               e.Submission.UserID,
		StoryID:              int32(e.Submission.StoryID),
		SegmentID:            int32(e.Segment.ID),
		HebrewText:           e.Segment.HebrewText,
		ReferenceEnglish:     e.Segment.ReferenceEnglish,
		StudentText:          e.Submission.StudentText,
		GrammarPointName:     optionalText(e.Segment.GrammarPointName),
		Model:                optionalText(e.Trace.Model),
		PromptID:             optionalInt4(e.PromptID),
		UserPrompt:           optionalText(e.Trace.UserPrompt),
		RawResponse:          optionalText(e.Trace.RawResponse),
		StopReason:           optionalText(e.Trace.StopReason),
		InputTokens:          optionalInt4(e.Trace.InputTokens),
		OutputTokens:         optionalInt4(e.Trace.OutputTokens),
		CacheReadInputTokens: optionalInt4(e.Trace.CacheReadInputTokens),
		LatencyMs:            optionalInt4(int(e.Trace.Latency.Milliseconds())),
	}
	if e.Err != nil {
		params.Error = optionalText(e.Err.Error())
	} else {
		params.Score = pgtype.Int4{Int32: int32(e.Grade.Score), Valid: true}
		params.Feedback = optionalText(e.Grade.Feedback)
	}
	return queries.InsertProduceGradingLog(ctx, params)
}

// optionalInt4 stores zero as NULL — the trace leaves counts at zero when the
// call never completed.
func optionalInt4(n int) pgtype.Int4 {
	return pgtype.Int4{Int32: int32(n), Valid: n != 0}
}
