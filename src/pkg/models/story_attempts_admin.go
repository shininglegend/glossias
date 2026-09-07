package models

import (
	"context"
	"errors"
	"fmt"

	"glossias/src/pkg/generated/db"
)

var ErrAttemptNotFound = errors.New("attempt not found")

// AttemptSummary is one row of the instructor's Manage Attempts dialog: an
// archived attempt with its frozen score, or the live current attempt scored
// from the answer tables. Stages is set only for the current attempt and
// lists the phases that hold data, each deletable on its own via the
// progress reset endpoint; archived attempts go all-or-nothing.
type AttemptSummary struct {
	StoryAttemptInfo
	Score  *AttemptScoreSnapshot `json:"score,omitempty"`
	Stages []AttemptStage        `json:"stages,omitempty"`
}

// AttemptStage is one phase of the live attempt that has something to delete.
type AttemptStage struct {
	Phase   ResetPhase `json:"phase"`
	Detail  string     `json:"detail"`
	Seconds int        `json:"seconds"`
}

// ListUserStoryAttemptSummaries returns every archived attempt with its
// snapshot, then the current attempt built from the live rows (always
// present; an untouched attempt has no Stages). Two round trips plus the
// live score build.
func ListUserStoryAttemptSummaries(ctx context.Context, userID string, storyID int32, title string) ([]AttemptSummary, error) {
	if queries == nil {
		return nil, errors.New("database not initialized")
	}
	rows, err := queries.ListUserStoryAttemptSnapshots(ctx, db.ListUserStoryAttemptSnapshotsParams{
		UserID:  userID,
		StoryID: storyID,
	})
	if err != nil {
		return nil, err
	}

	out := make([]AttemptSummary, 0, len(rows)+1)
	for _, row := range rows {
		snap, err := unmarshalAttemptScore(row.Snapshot)
		if err != nil {
			return nil, fmt.Errorf("attempt %d snapshot: %w", row.AttemptNumber, err)
		}
		item := AttemptSummary{StoryAttemptInfo: StoryAttemptInfo{Number: int(row.AttemptNumber)}, Score: snap}
		if row.CompletedAt.Valid {
			t := row.CompletedAt.Time
			item.CompletedAt = &t
		}
		out = append(out, item)
	}

	summary, err := GetUserStoryScoreSummary(ctx, userID, int(storyID))
	if err != nil {
		return nil, err
	}
	live, err := BuildLiveAttemptScore(ctx, userID, int(storyID), title, summary)
	if err != nil {
		return nil, err
	}
	return append(out, AttemptSummary{
		StoryAttemptInfo: StoryAttemptInfo{Number: len(rows) + 1, IsCurrent: true},
		Score:            live,
		Stages:           liveAttemptStages(live),
	}), nil
}

// liveAttemptStages lists, in flow order, each phase of a live score that has
// answers or time recorded. Legacy vocab/grammar appear only when used.
func liveAttemptStages(s *AttemptScoreSnapshot) []AttemptStage {
	stages := []AttemptStage{}
	add := func(phase ResetPhase, has bool, seconds int, detail string) {
		if has || seconds > 0 {
			stages = append(stages, AttemptStage{Phase: phase, Detail: detail, Seconds: seconds})
		}
	}
	add(ResetVideo, false, s.VideoTimeSeconds, "watched")
	add(ResetIdentify, s.IdentifyCorrectCount+s.IdentifyIncorrectCount > 0, s.IdentifyTimeSeconds,
		fmt.Sprintf("%d correct / %d incorrect", s.IdentifyCorrectCount, s.IdentifyIncorrectCount))
	translateDetail := "started"
	if s.TranslationCompleted {
		translateDetail = "completed"
	}
	add(ResetTranslate, s.TranslationCompleted || len(s.RequestedLines) > 0, s.TranslationTimeSeconds, translateDetail)
	add(ResetProduce, s.ProduceSegmentsSubmitted > 0, s.ProduceTimeSeconds,
		fmt.Sprintf("%d/%d submitted", s.ProduceSegmentsSubmitted, s.ProduceTotal))
	add(ResetRecall, s.RecallCorrectCount+s.RecallIncorrectCount > 0, s.RecallTimeSeconds,
		fmt.Sprintf("%d attempt%s", s.RecallAttempts, plural(s.RecallAttempts)))
	add(ResetVocab, s.VocabCorrectCount+s.VocabIncorrectCount > 0, s.VocabTimeSeconds,
		fmt.Sprintf("%d correct / %d incorrect", s.VocabCorrectCount, s.VocabIncorrectCount))
	add(ResetGrammar, s.GrammarCorrectCount+s.GrammarIncorrectCount > 0, s.GrammarTimeSeconds,
		fmt.Sprintf("%d correct / %d incorrect", s.GrammarCorrectCount, s.GrammarIncorrectCount))
	return stages
}

func plural(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}

// DeleteArchivedAttempt removes one archived attempt (snapshot cascades) and
// closes the gap so attempt numbers stay contiguous — the live attempt is
// always archived count + 1 and attempt 1 stays the official score. Rows
// above the deleted number are shifted down one at a time in ascending
// order inside the same transaction, which never trips the unique key.
func DeleteArchivedAttempt(ctx context.Context, userID string, storyID int32, number int) error {
	if queries == nil {
		return errors.New("database not initialized")
	}
	return withTransaction(ctx, func(txCtx context.Context) error {
		n, err := queries.DeleteUserStoryAttemptByNumber(txCtx, db.DeleteUserStoryAttemptByNumberParams{
			UserID:        userID,
			StoryID:       storyID,
			AttemptNumber: int32(number),
		})
		if err != nil {
			return err
		}
		if n == 0 {
			return ErrAttemptNotFound
		}
		later, err := queries.ListUserStoryAttemptsAfter(txCtx, db.ListUserStoryAttemptsAfterParams{
			UserID:        userID,
			StoryID:       storyID,
			AttemptNumber: int32(number),
		})
		if err != nil {
			return err
		}
		for _, row := range later {
			if err := queries.SetStoryAttemptNumber(txCtx, db.SetStoryAttemptNumberParams{
				AttemptID:     row.AttemptID,
				AttemptNumber: row.AttemptNumber - 1,
			}); err != nil {
				return err
			}
		}
		return nil
	})
}
