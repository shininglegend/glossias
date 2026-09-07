// glossias/src/pkg/models/reset_progress.go
package models

import (
	"context"
	"errors"
	"fmt"
	"slices"

	"glossias/src/pkg/generated/db"

	"github.com/jackc/pgx/v5/pgtype"
)

// ResetPhase names what part of a student's progress on one story to wipe.
// Phase completion is derived from the answer/submission rows (see
// SUMMER_2026.md), so deleting them reopens the phase for the student.
//
// Live rows are always the student's current attempt, so every phase except
// ResetAll acts on that attempt alone and leaves archived attempts (and the
// official first-attempt score) untouched. ResetAll also drops the archive.
type ResetPhase string

const (
	ResetAll       ResetPhase = "all"
	ResetExercises ResetPhase = "exercises" // current attempt: Identify → Recall, video kept
	ResetVideo     ResetPhase = "video"
	ResetIdentify  ResetPhase = "identify"
	ResetTranslate ResetPhase = "translate"
	ResetProduce   ResetPhase = "produce"
	ResetRecall    ResetPhase = "recall"
	ResetVocab     ResetPhase = "vocab"   // legacy flow
	ResetGrammar   ResetPhase = "grammar" // legacy flow
)

// ResetPhases lists every accepted value, in flow order.
var ResetPhases = []ResetPhase{
	ResetAll, ResetExercises, ResetVideo, ResetIdentify, ResetTranslate, ResetProduce, ResetRecall, ResetVocab, ResetGrammar,
}

var ErrInvalidResetPhase = errors.New("invalid reset phase")

// ResetResult reports how many rows were removed, keyed by table for the
// answer/submission tables and "time_tracking" for user_time_tracking.
type ResetResult struct {
	Phase   ResetPhase       `json:"phase"`
	Deleted map[string]int64 `json:"deleted"`
}

type rowDeleter func(ctx context.Context, userID string, storyID int32) (int64, error)

// phaseSpec lists the answer tables a single phase owns and the
// user_time_tracking.phase value its time rows carry (set from the route at
// write time by PhaseFromRoute, which GetStoryStudentPerformance buckets on).
type phaseSpec struct {
	tables    map[string]rowDeleter
	timePhase string
}

var phaseSpecs = map[ResetPhase]phaseSpec{
	ResetVideo: {timePhase: "video"},
	ResetIdentify: {
		tables: map[string]rowDeleter{
			"identify_correct_answers": func(ctx context.Context, u string, s int32) (int64, error) {
				return queries.DeleteUserStoryIdentifyCorrect(ctx, db.DeleteUserStoryIdentifyCorrectParams{UserID: u, StoryID: s})
			},
			"identify_incorrect_answers": func(ctx context.Context, u string, s int32) (int64, error) {
				return queries.DeleteUserStoryIdentifyIncorrect(ctx, db.DeleteUserStoryIdentifyIncorrectParams{UserID: u, StoryID: s})
			},
		},
		timePhase: "identify",
	},
	ResetTranslate: {
		tables: map[string]rowDeleter{
			"translation_requests": func(ctx context.Context, u string, s int32) (int64, error) {
				return queries.DeleteUserStoryTranslationRequest(ctx, db.DeleteUserStoryTranslationRequestParams{UserID: u, StoryID: s})
			},
		},
		timePhase: "translate",
	},
	ResetProduce: {
		tables: map[string]rowDeleter{
			"produce_submissions": func(ctx context.Context, u string, s int32) (int64, error) {
				return queries.DeleteUserStoryProduceSubmissions(ctx, db.DeleteUserStoryProduceSubmissionsParams{UserID: u, StoryID: s})
			},
			"produce_attempt_starts": func(ctx context.Context, u string, s int32) (int64, error) {
				return queries.DeleteUserStoryProduceAttemptStarts(ctx, db.DeleteUserStoryProduceAttemptStartsParams{UserID: u, StoryID: s})
			},
		},
		timePhase: "produce",
	},
	ResetRecall: {
		tables: map[string]rowDeleter{
			"recall_correct_answers": func(ctx context.Context, u string, s int32) (int64, error) {
				return queries.DeleteUserStoryRecallCorrect(ctx, db.DeleteUserStoryRecallCorrectParams{UserID: u, StoryID: s})
			},
			"recall_incorrect_answers": func(ctx context.Context, u string, s int32) (int64, error) {
				return queries.DeleteUserStoryRecallIncorrect(ctx, db.DeleteUserStoryRecallIncorrectParams{UserID: u, StoryID: s})
			},
		},
		timePhase: "recall",
	},
	ResetVocab: {
		tables: map[string]rowDeleter{
			"vocab_correct_answers": func(ctx context.Context, u string, s int32) (int64, error) {
				return queries.DeleteUserStoryVocabCorrect(ctx, db.DeleteUserStoryVocabCorrectParams{UserID: u, StoryID: s})
			},
			"vocab_incorrect_answers": func(ctx context.Context, u string, s int32) (int64, error) {
				return queries.DeleteUserStoryVocabIncorrect(ctx, db.DeleteUserStoryVocabIncorrectParams{UserID: u, StoryID: s})
			},
		},
		timePhase: "vocab",
	},
	ResetGrammar: {
		tables: map[string]rowDeleter{
			"grammar_correct_answers": func(ctx context.Context, u string, s int32) (int64, error) {
				return queries.DeleteUserStoryGrammarCorrect(ctx, db.DeleteUserStoryGrammarCorrectParams{UserID: u, StoryID: s})
			},
			"grammar_incorrect_answers": func(ctx context.Context, u string, s int32) (int64, error) {
				return queries.DeleteUserStoryGrammarIncorrect(ctx, db.DeleteUserStoryGrammarIncorrectParams{UserID: u, StoryID: s})
			},
		},
		timePhase: "grammar",
	},
}

// ResetUserStoryProgress deletes one student's progress on one story:
// everything including archived attempts (ResetAll), the current attempt's
// exercises (ResetExercises), or a single phase of the current attempt with
// its time rows. It runs in a transaction; deleting zero rows is not an
// error. A whole-story reset is three statements regardless of table count.
//
// No cache work is needed: per-user scores are never cached, and the
// time-tracking session cache holds only session IDs, not DB row references.
func ResetUserStoryProgress(ctx context.Context, userID string, storyID int32, phase ResetPhase) (ResetResult, error) {
	if !slices.Contains(ResetPhases, phase) {
		return ResetResult{}, fmt.Errorf("%w: %q", ErrInvalidResetPhase, phase)
	}

	result := ResetResult{Phase: phase, Deleted: map[string]int64{}}

	err := withTransaction(ctx, func(txCtx context.Context) error {
		switch phase {
		case ResetAll:
			return resetAll(txCtx, userID, storyID, result.Deleted)
		case ResetExercises:
			return resetExercises(txCtx, userID, storyID, result.Deleted)
		}

		spec := phaseSpecs[phase]
		for table, del := range spec.tables {
			n, err := del(txCtx, userID, storyID)
			if err != nil {
				return fmt.Errorf("delete %s: %w", table, err)
			}
			result.Deleted[table] = n
		}
		n, err := deleteTimeTrackingPhases(txCtx, userID, storyID, []string{spec.timePhase})
		if err != nil {
			return err
		}
		result.Deleted["time_tracking"] = n
		return nil
	})
	if err != nil {
		return ResetResult{}, err
	}
	return result, nil
}

func resetAll(ctx context.Context, userID string, storyID int32, deleted map[string]int64) error {
	if err := resetAnswers(ctx, userID, storyID, deleted); err != nil {
		return err
	}

	n, err := queries.DeleteUserStoryTimeTracking(ctx, db.DeleteUserStoryTimeTrackingParams{UserID: userID, StoryID: pgtype.Int4{Int32: storyID, Valid: true}})
	if err != nil {
		return fmt.Errorf("delete time tracking: %w", err)
	}
	deleted["time_tracking"] = n

	attempts, err := queries.DeleteUserStoryAttempts(ctx, db.DeleteUserStoryAttemptsParams{UserID: userID, StoryID: storyID})
	if err != nil {
		return fmt.Errorf("delete story attempts: %w", err)
	}
	deleted["story_attempts"] = attempts
	return nil
}

// exerciseTimePhases are the phases an exercise reset wipes. Video is omitted
// so watch time and the video page stay intact.
var exerciseTimePhases = []string{"identify", "translate", "produce", "recall", "vocab", "grammar"}

// resetExercises clears Identify/Translate/Produce/Recall (and leftover
// vocab/grammar) answers and their time rows in two statements. It is the
// body of an admin ResetExercises and of the automatic archive that runs when
// a student finishes a story, which needs it inside the snapshot transaction.
func resetExercises(ctx context.Context, userID string, storyID int32, deleted map[string]int64) error {
	if err := resetAnswers(ctx, userID, storyID, deleted); err != nil {
		return err
	}
	n, err := deleteTimeTrackingPhases(ctx, userID, storyID, exerciseTimePhases)
	if err != nil {
		return err
	}
	deleted["time_tracking"] = n
	return nil
}

func deleteTimeTrackingPhases(ctx context.Context, userID string, storyID int32, phases []string) (int64, error) {
	n, err := queries.DeleteUserStoryTimeTrackingByPhases(ctx, db.DeleteUserStoryTimeTrackingByPhasesParams{
		UserID:  userID,
		StoryID: storyID,
		Phases:  phases,
	})
	if err != nil {
		return 0, fmt.Errorf("delete time tracking %v: %w", phases, err)
	}
	return n, nil
}

func resetAnswers(ctx context.Context, userID string, storyID int32, deleted map[string]int64) error {
	counts, err := queries.ResetUserStoryAnswers(ctx, db.ResetUserStoryAnswersParams{UserID: userID, StoryID: storyID})
	if err != nil {
		return err
	}
	deleted["vocab_correct_answers"] = counts.VocabCorrect
	deleted["vocab_incorrect_answers"] = counts.VocabIncorrect
	deleted["grammar_correct_answers"] = counts.GrammarCorrect
	deleted["grammar_incorrect_answers"] = counts.GrammarIncorrect
	deleted["translation_requests"] = counts.TranslationRequests
	deleted["identify_correct_answers"] = counts.IdentifyCorrect
	deleted["identify_incorrect_answers"] = counts.IdentifyIncorrect
	deleted["produce_submissions"] = counts.ProduceSubmissions
	deleted["produce_attempt_starts"] = counts.ProduceAttemptStarts
	deleted["recall_correct_answers"] = counts.RecallCorrect
	deleted["recall_incorrect_answers"] = counts.RecallIncorrect
	return nil
}
