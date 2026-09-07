// glossias/src/pkg/models/student_drilldown.go
package models

import (
	"context"
	"errors"
	"slices"
	"time"

	"glossias/src/pkg/generated/db"

	"github.com/jackc/pgx/v5"
)

// StudentStoryDrilldown is one student's actual answers and submissions on one
// story, phase by phase — the detail behind a CourseStudentPerformance row.
type StudentStoryDrilldown struct {
	UserID     string `json:"user_id"`
	UserName   string `json:"user_name"`
	Email      string `json:"email"`
	StoryID    int32  `json:"story_id"`
	StoryTitle string `json:"story_title"`

	IdentifyAnswers []IdentifyAnswerDetail `json:"identify_answers"`
	Translate       TranslateDetail        `json:"translate"`
	ProduceSegments []ProduceSegmentDetail `json:"produce_segments"`
	RecallAttempts  []RecallAttemptDetail  `json:"recall_attempts"`

	Time PhaseTimeBreakdown `json:"time"`
}

// IdentifyAnswerDetail is one picture-quiz pick, in the order it was made.
// SelectedWord is empty on correct picks (the pick was the target word).
type IdentifyAnswerDetail struct {
	LineNumber   int32     `json:"line_number"`
	Correct      bool      `json:"correct"`
	TargetWord   string    `json:"target_word"`
	SelectedWord string    `json:"selected_word,omitempty"`
	AttemptedAt  time.Time `json:"attempted_at"`
}

// TranslateDetail mirrors what the student saw: which lines they requested
// (0-based, as stored) and whether they finished the phase.
type TranslateDetail struct {
	Started        bool       `json:"started"`
	Completed      bool       `json:"completed"`
	RequestedLines []int32    `json:"requested_lines"`
	CompletedAt    *time.Time `json:"completed_at,omitempty"`
}

// ProduceSegmentDetail is one authored segment with every submission the
// student made for it, oldest first — not just the latest one that is scored.
type ProduceSegmentDetail struct {
	SegmentOrder     int32                     `json:"segment_order"`
	HebrewText       string                    `json:"hebrew_text"`
	ReferenceEnglish string                    `json:"reference_english"`
	GrammarPointName string                    `json:"grammar_point_name,omitempty"`
	Submissions      []ProduceSubmissionDetail `json:"submissions"`
}

type ProduceSubmissionDetail struct {
	StudentText string     `json:"student_text"`
	AiScore     *int32     `json:"ai_score,omitempty"`
	AiFeedback  string     `json:"ai_feedback,omitempty"`
	GradedAt    *time.Time `json:"graded_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
}

// RecallAttemptDetail is one full ordering attempt: where the student placed
// each sentence, in the position order they submitted.
type RecallAttemptDetail struct {
	AttemptedAt time.Time         `json:"attempted_at"`
	AllCorrect  bool              `json:"all_correct"`
	Placements  []RecallPlacement `json:"placements"`
}

type RecallPlacement struct {
	SelectedPosition int32  `json:"selected_position"`
	CorrectPosition  int32  `json:"correct_position"`
	HebrewText       string `json:"hebrew_text"`
	Correct          bool   `json:"correct"`
}

// PhaseTimeBreakdown is the student's completed time per phase, in seconds.
type PhaseTimeBreakdown struct {
	VideoSeconds     int `json:"video_seconds"`
	IdentifySeconds  int `json:"identify_seconds"`
	TranslateSeconds int `json:"translate_seconds"`
	ProduceSeconds   int `json:"produce_seconds"`
	RecallSeconds    int `json:"recall_seconds"`
	VocabSeconds     int `json:"vocab_seconds"`
	GrammarSeconds   int `json:"grammar_seconds"`
}

// GetStudentStoryDrilldown assembles the per-phase answer detail for one
// student on one story. Returns ErrNotFound if the user does not exist.
// Seven queries regardless of how much the student has done.
func GetStudentStoryDrilldown(ctx context.Context, storyID int32, userID string) (*StudentStoryDrilldown, error) {
	header, err := queries.GetStudentStoryHeader(ctx, db.GetStudentStoryHeaderParams{StoryID: int32(storyID), UserID: userID})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	result := &StudentStoryDrilldown{
		UserID:     header.UserID,
		UserName:   header.UserName,
		Email:      header.Email,
		StoryID:    storyID,
		StoryTitle: header.StoryTitle,

		IdentifyAnswers: []IdentifyAnswerDetail{},
		ProduceSegments: []ProduceSegmentDetail{},
		RecallAttempts:  []RecallAttemptDetail{},
	}

	identifyRows, err := queries.GetUserStoryIdentifyAnswerLog(ctx, db.GetUserStoryIdentifyAnswerLogParams{UserID: userID, StoryID: storyID})
	if err != nil {
		return nil, err
	}
	for _, row := range identifyRows {
		result.IdentifyAnswers = append(result.IdentifyAnswers, IdentifyAnswerDetail{
			LineNumber:   row.LineNumber,
			Correct:      row.Correct,
			TargetWord:   row.TargetWord,
			SelectedWord: row.SelectedWord,
			AttemptedAt:  row.AttemptedAt.Time,
		})
	}

	result.Translate, err = translateDetail(ctx, userID, storyID)
	if err != nil {
		return nil, err
	}

	result.ProduceSegments, err = produceDetail(ctx, userID, storyID)
	if err != nil {
		return nil, err
	}

	recallRows, err := queries.GetUserStoryRecallAnswerLog(ctx, db.GetUserStoryRecallAnswerLogParams{UserID: userID, StoryID: storyID})
	if err != nil {
		return nil, err
	}
	result.RecallAttempts = groupRecallAttempts(recallRows)

	timeData, err := GetUserStoryTimeTracking(ctx, userID, storyID)
	if err != nil {
		return nil, err
	}
	result.Time = PhaseTimeBreakdown{
		VideoSeconds:     timeData.VideoTimeSeconds,
		IdentifySeconds:  timeData.IdentifyTimeSeconds,
		TranslateSeconds: timeData.TranslationTimeSeconds,
		ProduceSeconds:   timeData.ProduceTimeSeconds,
		RecallSeconds:    timeData.RecallTimeSeconds,
		VocabSeconds:     timeData.VocabTimeSeconds,
		GrammarSeconds:   timeData.GrammarTimeSeconds,
	}

	return result, nil
}

func translateDetail(ctx context.Context, userID string, storyID int32) (TranslateDetail, error) {
	detail := TranslateDetail{RequestedLines: []int32{}}

	tr, err := queries.GetTranslationRequest(ctx, db.GetTranslationRequestParams{UserID: userID, StoryID: storyID})
	if errors.Is(err, pgx.ErrNoRows) {
		return detail, nil // phase never started
	}
	if err != nil {
		return detail, err
	}

	detail.Started = true
	detail.RequestedLines = slices.Clone(tr.RequestedLines)
	slices.Sort(detail.RequestedLines)
	if tr.CompletedAt.Valid {
		detail.Completed = true
		completedAt := tr.CompletedAt.Time
		detail.CompletedAt = &completedAt
	}
	return detail, nil
}

func produceDetail(ctx context.Context, userID string, storyID int32) ([]ProduceSegmentDetail, error) {
	segments, err := queries.GetStoryProduceSegments(ctx, storyID)
	if err != nil {
		return nil, err
	}

	details := make([]ProduceSegmentDetail, 0, len(segments))
	byOrder := make(map[int32]int, len(segments))
	for _, seg := range segments {
		byOrder[seg.SegmentOrder] = len(details)
		details = append(details, ProduceSegmentDetail{
			SegmentOrder:     seg.SegmentOrder,
			HebrewText:       seg.HebrewText,
			ReferenceEnglish: seg.ReferenceEnglish,
			GrammarPointName: seg.GrammarPointName.String,
			Submissions:      []ProduceSubmissionDetail{},
		})
	}

	subs, err := queries.GetUserStoryProduceSubmissionHistory(ctx, db.GetUserStoryProduceSubmissionHistoryParams{UserID: userID, StoryID: storyID})
	if err != nil {
		return nil, err
	}
	for _, sub := range subs {
		i, ok := byOrder[sub.SegmentOrder]
		if !ok {
			continue // submission for a segment the author has since removed
		}
		detail := ProduceSubmissionDetail{
			StudentText: sub.StudentText,
			AiFeedback:  sub.AiFeedback.String,
			CreatedAt:   sub.CreatedAt.Time,
		}
		if sub.AiScore.Valid {
			score := sub.AiScore.Int32
			detail.AiScore = &score
		}
		if sub.GradedAt.Valid {
			gradedAt := sub.GradedAt.Time
			detail.GradedAt = &gradedAt
		}
		details[i].Submissions = append(details[i].Submissions, detail)
	}
	return details, nil
}

// groupRecallAttempts folds the flat placement log back into whole attempts.
// Rows arrive ordered by attempted_at; each attempt places every sentence
// exactly once, so a repeated selected_position starts the next attempt. That
// boundary rule doesn't depend on the per-row insert timestamps being equal
// within an attempt (they are not written in one transaction).
func groupRecallAttempts(rows []db.GetUserStoryRecallAnswerLogRow) []RecallAttemptDetail {
	attempts := []RecallAttemptDetail{}
	seen := map[int32]bool{}

	for _, row := range rows {
		if len(attempts) == 0 || seen[row.SelectedPosition] {
			attempts = append(attempts, RecallAttemptDetail{
				AttemptedAt: row.AttemptedAt.Time,
				AllCorrect:  true,
				Placements:  []RecallPlacement{},
			})
			seen = map[int32]bool{}
		}
		seen[row.SelectedPosition] = true

		current := &attempts[len(attempts)-1]
		current.Placements = append(current.Placements, RecallPlacement{
			SelectedPosition: row.SelectedPosition,
			CorrectPosition:  row.CorrectPosition,
			HebrewText:       row.HebrewText,
			Correct:          row.Correct,
		})
		if !row.Correct {
			current.AllCorrect = false
		}
	}

	for i := range attempts {
		slices.SortStableFunc(attempts[i].Placements, func(a, b RecallPlacement) int {
			return int(a.SelectedPosition - b.SelectedPosition)
		})
	}
	return attempts
}
