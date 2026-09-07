package models

import (
	"context"
	"encoding/json"
	"errors"
	"slices"
	"time"

	"glossias/src/pkg/generated/db"

	"github.com/jackc/pgx/v5"
)

var ErrStoryNotComplete = errors.New("story is not complete")

// AttemptProduceSegment is one Produce passage frozen with its AI notes.
type AttemptProduceSegment struct {
	SegmentOrder     int    `json:"segment_order"`
	HebrewText       string `json:"hebrew_text"`
	ReferenceEnglish string `json:"reference_english"`
	StudentText      string `json:"student_text"`
	AiScore          *int   `json:"ai_score,omitempty"`
	AiFeedback       string `json:"ai_feedback,omitempty"`
}

// AttemptScoreSnapshot is the frozen Score payload for one story attempt.
// Field names match the student Score API so the same JSON can be served.
type AttemptScoreSnapshot struct {
	StoryTitle       string  `json:"story_title"`
	TotalTimeSeconds int     `json:"total_time_seconds"`
	OverallAccuracy  float64 `json:"overall_accuracy"`

	IdentifyAccuracy       float64 `json:"identify_accuracy"`
	IdentifyCorrectCount   int     `json:"identify_correct_count"`
	IdentifyIncorrectCount int     `json:"identify_incorrect_count"`
	IdentifyTotal          int     `json:"identify_total"`

	ProduceScore             float64                 `json:"produce_score"`
	ProduceSegmentsSubmitted int                     `json:"produce_segments_submitted"`
	ProduceSegmentsGraded    int                     `json:"produce_segments_graded"`
	ProduceTotal             int                     `json:"produce_total"`
	ProduceSegments          []AttemptProduceSegment `json:"produce_segments"`

	RecallAccuracy       float64 `json:"recall_accuracy"`
	RecallCorrectCount   int     `json:"recall_correct_count"`
	RecallIncorrectCount int     `json:"recall_incorrect_count"`
	RecallAttempts       int     `json:"recall_attempts"`
	RecallTotal          int     `json:"recall_total"`

	VocabAccuracy         float64 `json:"vocab_accuracy"`
	VocabCorrectCount     int     `json:"vocab_correct_count"`
	VocabIncorrectCount   int     `json:"vocab_incorrect_count"`
	GrammarAccuracy       float64 `json:"grammar_accuracy"`
	GrammarCorrectCount   int     `json:"grammar_correct_count"`
	GrammarIncorrectCount int     `json:"grammar_incorrect_count"`

	VideoTimeSeconds       int `json:"video_time_seconds"`
	IdentifyTimeSeconds    int `json:"identify_time_seconds"`
	TranslationTimeSeconds int `json:"translation_time_seconds"`
	ProduceTimeSeconds     int `json:"produce_time_seconds"`
	RecallTimeSeconds      int `json:"recall_time_seconds"`
	VocabTimeSeconds       int `json:"vocab_time_seconds"`
	GrammarTimeSeconds     int `json:"grammar_time_seconds"`

	TranslationCompleted bool    `json:"translation_completed"`
	RequestedLines       []int32 `json:"requested_lines,omitempty"`
}

// StoryAttemptInfo is one redo cycle for admin attempt switching.
type StoryAttemptInfo struct {
	Number      int        `json:"number"`
	HasSnapshot bool       `json:"has_snapshot"`
	IsCurrent   bool       `json:"is_current"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
}

// StoryExercisesComplete reports whether live rows would unlock the Score page.
func StoryExercisesComplete(c *PageCompletion) bool {
	if c == nil {
		return false
	}
	if c.IdentifyTotal > 0 && !c.IdentifyComplete() {
		return false
	}
	if !c.TranslateComplete() {
		return false
	}
	if c.ProduceTotal > 0 && !c.ProduceComplete() {
		return false
	}
	if c.RecallTotal > 0 && !c.RecallComplete() {
		return false
	}
	return true
}

// GetOfficialAttemptScore returns the frozen first-attempt score, or nil
// when the student has never redone (live rows are still attempt 1).
func GetOfficialAttemptScore(ctx context.Context, userID string, storyID int) (*AttemptScoreSnapshot, error) {
	return GetAttemptScore(ctx, userID, storyID, 1)
}

// GetAttemptScore loads a frozen score for one attempt number.
func GetAttemptScore(ctx context.Context, userID string, storyID, attemptNumber int) (*AttemptScoreSnapshot, error) {
	if queries == nil {
		return nil, errors.New("database not initialized")
	}
	row, err := queries.GetUserStoryAttemptSnapshotByNumber(ctx, db.GetUserStoryAttemptSnapshotByNumberParams{
		UserID:        userID,
		StoryID:       int32(storyID),
		AttemptNumber: int32(attemptNumber),
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return unmarshalAttemptScore(row.Snapshot)
}

// BuildLiveAttemptScore computes the current Score payload from live rows.
func BuildLiveAttemptScore(ctx context.Context, userID string, storyID int, title string) (*AttemptScoreSnapshot, error) {
	summary, err := GetUserStoryScoreSummary(ctx, userID, storyID)
	if err != nil {
		return nil, err
	}
	timeData, err := GetUserStoryTimeTracking(ctx, userID, int32(storyID))
	if err != nil {
		return nil, err
	}
	totals, err := GetStoryPhaseTotals(ctx, storyID)
	if err != nil {
		return nil, err
	}
	scores := ComputePhaseScores(*summary, *totals)
	segments, err := liveProduceSegments(ctx, userID, storyID)
	if err != nil {
		return nil, err
	}
	translate, err := translateDetail(ctx, userID, int32(storyID))
	if err != nil {
		return nil, err
	}

	totalTime := timeData.VideoTimeSeconds + timeData.IdentifyTimeSeconds +
		timeData.TranslationTimeSeconds + timeData.ProduceTimeSeconds +
		timeData.RecallTimeSeconds + timeData.VocabTimeSeconds

	return &AttemptScoreSnapshot{
		StoryTitle:       title,
		TotalTimeSeconds: totalTime,
		OverallAccuracy:  scores.Overall,

		IdentifyAccuracy:       scores.IdentifyAccuracy,
		IdentifyCorrectCount:   summary.IdentifyCorrect,
		IdentifyIncorrectCount: summary.IdentifyIncorrect,
		IdentifyTotal:          totals.IdentifyTotal,

		ProduceScore:             scores.ProduceScore,
		ProduceSegmentsSubmitted: summary.ProduceSubmitted,
		ProduceSegmentsGraded:    summary.ProduceGraded,
		ProduceTotal:             totals.ProduceTotal,
		ProduceSegments:          segments,

		RecallAccuracy:       scores.RecallAccuracy,
		RecallCorrectCount:   summary.RecallCorrect,
		RecallIncorrectCount: summary.RecallIncorrect,
		RecallAttempts:       scores.RecallAttempts,
		RecallTotal:          totals.RecallTotal,

		VocabAccuracy:         scores.VocabAccuracy,
		VocabCorrectCount:     summary.VocabCorrect,
		VocabIncorrectCount:   summary.VocabIncorrect,
		GrammarAccuracy:       scores.GrammarAccuracy,
		GrammarCorrectCount:   summary.GrammarCorrect,
		GrammarIncorrectCount: summary.GrammarIncorrect,

		VideoTimeSeconds:       timeData.VideoTimeSeconds,
		IdentifyTimeSeconds:    timeData.IdentifyTimeSeconds,
		TranslationTimeSeconds: timeData.TranslationTimeSeconds,
		ProduceTimeSeconds:     timeData.ProduceTimeSeconds,
		RecallTimeSeconds:      timeData.RecallTimeSeconds,
		VocabTimeSeconds:       timeData.VocabTimeSeconds,
		GrammarTimeSeconds:     timeData.GrammarTimeSeconds,

		TranslationCompleted: translate.Completed,
		RequestedLines:       translate.RequestedLines,
	}, nil
}

// StartStudentStoryRedo freezes the current live score as the completed
// attempt, opens the next attempt, and wipes exercise answers so Identify
// starts clean. Video time is left intact.
func StartStudentStoryRedo(ctx context.Context, userID string, storyID int32, title string) (ResetResult, error) {
	completion, err := GetUserStoryPageCompletion(ctx, userID, int(storyID))
	if err != nil {
		return ResetResult{}, err
	}
	if !StoryExercisesComplete(completion) {
		return ResetResult{}, ErrStoryNotComplete
	}

	snap, err := BuildLiveAttemptScore(ctx, userID, int(storyID), title)
	if err != nil {
		return ResetResult{}, err
	}

	// Snapshot and wipe commit together: a failed wipe must not leave a
	// frozen attempt beside still-live answers (or vice versa).
	result := newExerciseResetResult()
	err = withTransaction(ctx, func(txCtx context.Context) error {
		if err := persistCompletedAttempt(txCtx, userID, storyID, snap); err != nil {
			return err
		}
		return resetExercises(txCtx, userID, storyID, result.Deleted)
	})
	if err != nil {
		return ResetResult{}, err
	}
	return result, nil
}

func persistCompletedAttempt(ctx context.Context, userID string, storyID int32, snap *AttemptScoreSnapshot) error {
	if queries == nil {
		return errors.New("database not initialized")
	}
	payload, err := json.Marshal(snap)
	if err != nil {
		return err
	}

	// The latest row is the open attempt; a student who has never redone has
	// no rows yet, so attempt 1 is created here.
	current, err := queries.GetLatestUserStoryAttempt(ctx, db.GetLatestUserStoryAttemptParams{
		UserID:  userID,
		StoryID: storyID,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		current, err = queries.CreateStoryAttempt(ctx, db.CreateStoryAttemptParams{
			UserID:        userID,
			StoryID:       storyID,
			AttemptNumber: 1,
		})
	}
	if err != nil {
		return err
	}

	if err := queries.UpsertAttemptScoreSnapshot(ctx, db.UpsertAttemptScoreSnapshotParams{
		AttemptID: current.AttemptID,
		Snapshot:  payload,
	}); err != nil {
		return err
	}
	if err := queries.MarkStoryAttemptComplete(ctx, current.AttemptID); err != nil {
		return err
	}

	_, err = queries.CreateStoryAttempt(ctx, db.CreateStoryAttemptParams{
		UserID:        userID,
		StoryID:       storyID,
		AttemptNumber: current.AttemptNumber + 1,
	})
	return err
}

// ListUserStoryAttempts returns every redo cycle, synthesizing attempt 1
// when the student has never redone.
func ListUserStoryAttempts(ctx context.Context, userID string, storyID int32) ([]StoryAttemptInfo, error) {
	if queries == nil {
		return nil, errors.New("database not initialized")
	}
	rows, err := queries.GetUserStoryAttemptsWithSnapshots(ctx, db.GetUserStoryAttemptsWithSnapshotsParams{
		UserID:  userID,
		StoryID: storyID,
	})
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return []StoryAttemptInfo{{Number: 1, IsCurrent: true}}, nil
	}

	current := int(rows[len(rows)-1].AttemptNumber)
	out := make([]StoryAttemptInfo, 0, len(rows))
	for _, row := range rows {
		info := StoryAttemptInfo{
			Number:      int(row.AttemptNumber),
			HasSnapshot: row.HasSnapshot,
			IsCurrent:   int(row.AttemptNumber) == current,
		}
		if row.CompletedAt.Valid {
			t := row.CompletedAt.Time
			info.CompletedAt = &t
		}
		out = append(out, info)
	}
	return out, nil
}

// ScoreProduceNotes is the latest Produce submission per segment for the
// student Score card (order, text, AI score and feedback).
func ScoreProduceNotes(ctx context.Context, userID string, storyID int) ([]AttemptProduceSegment, error) {
	subs, err := GetUserStoryProduceSubmissions(ctx, userID, storyID)
	if err != nil {
		return nil, err
	}
	out := make([]AttemptProduceSegment, 0, len(subs))
	for _, sub := range subs {
		out = append(out, AttemptProduceSegment{
			SegmentOrder:     sub.SegmentOrder,
			HebrewText:       sub.HebrewText,
			ReferenceEnglish: sub.ReferenceEnglish,
			StudentText:      sub.StudentText,
			AiScore:          sub.AiScore,
			AiFeedback:       sub.AiFeedback,
		})
	}
	slices.SortFunc(out, func(a, b AttemptProduceSegment) int {
		return a.SegmentOrder - b.SegmentOrder
	})
	return out, nil
}

func liveProduceSegments(ctx context.Context, userID string, storyID int) ([]AttemptProduceSegment, error) {
	authored, err := GetStoryProduceSegments(ctx, storyID)
	if err != nil {
		return nil, err
	}
	subs, err := GetUserStoryProduceSubmissions(ctx, userID, storyID)
	if err != nil {
		return nil, err
	}
	bySeg := make(map[int]ProduceSubmission, len(subs))
	for _, sub := range subs {
		bySeg[sub.SegmentID] = sub
	}
	out := make([]AttemptProduceSegment, 0, len(authored))
	for _, seg := range authored {
		item := AttemptProduceSegment{
			SegmentOrder:     seg.SegmentOrder,
			HebrewText:       seg.HebrewText,
			ReferenceEnglish: seg.ReferenceEnglish,
		}
		if sub, ok := bySeg[seg.ID]; ok {
			item.StudentText = sub.StudentText
			item.AiScore = sub.AiScore
			item.AiFeedback = sub.AiFeedback
		}
		out = append(out, item)
	}
	return out, nil
}

func unmarshalAttemptScore(raw []byte) (*AttemptScoreSnapshot, error) {
	if len(raw) == 0 {
		return nil, nil
	}
	var snap AttemptScoreSnapshot
	if err := json.Unmarshal(raw, &snap); err != nil {
		return nil, err
	}
	if snap.ProduceSegments == nil {
		snap.ProduceSegments = []AttemptProduceSegment{}
	}
	return &snap, nil
}

func applyOfficialSnapshot(p *CourseStudentPerformance, raw []byte) {
	snap, err := unmarshalAttemptScore(raw)
	if snap == nil || err != nil {
		return
	}
	p.OverallAccuracy = snap.OverallAccuracy
	p.IdentifyCorrect = snap.IdentifyCorrectCount
	p.IdentifyIncorrect = snap.IdentifyIncorrectCount
	p.IdentifyAccuracy = snap.IdentifyAccuracy
	p.TranslationCompleted = snap.TranslationCompleted
	if snap.RequestedLines != nil {
		p.RequestedLines = snap.RequestedLines
	}
	p.ProduceSubmitted = snap.ProduceSegmentsSubmitted
	p.ProduceGraded = snap.ProduceSegmentsGraded
	p.ProduceScore = snap.ProduceScore
	p.RecallCorrect = snap.RecallCorrectCount
	p.RecallIncorrect = snap.RecallIncorrectCount
	p.RecallAttempts = snap.RecallAttempts
	p.RecallAccuracy = snap.RecallAccuracy
	p.VocabCorrect = snap.VocabCorrectCount
	p.VocabIncorrect = snap.VocabIncorrectCount
	p.VocabAccuracy = snap.VocabAccuracy
	p.GrammarCorrect = snap.GrammarCorrectCount
	p.GrammarIncorrect = snap.GrammarIncorrectCount
	p.GrammarAccuracy = snap.GrammarAccuracy
	p.VideoTimeSeconds = int32(snap.VideoTimeSeconds)
	p.IdentifyTimeSeconds = int32(snap.IdentifyTimeSeconds)
	p.TranslationTimeSeconds = int32(snap.TranslationTimeSeconds)
	p.ProduceTimeSeconds = int32(snap.ProduceTimeSeconds)
	p.RecallTimeSeconds = int32(snap.RecallTimeSeconds)
	p.VocabTimeSeconds = int32(snap.VocabTimeSeconds)
	p.GrammarTimeSeconds = int32(snap.GrammarTimeSeconds)
	p.TotalTimeSeconds = int32(snap.TotalTimeSeconds)
}

func hasLiveExerciseWork(row db.GetStoryStudentPerformanceRow) bool {
	return row.IdentifyCorrect+row.IdentifyIncorrect+
		row.RecallCorrect+row.RecallIncorrect+
		row.ProduceSubmitted+row.VocabCorrect+row.VocabIncorrect+
		row.GrammarCorrect+row.GrammarIncorrect > 0 || row.TranslationCompleted
}
