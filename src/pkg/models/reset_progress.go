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
type ResetPhase string

const (
	ResetAll       ResetPhase = "all"
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
	ResetAll, ResetVideo, ResetIdentify, ResetTranslate, ResetProduce, ResetRecall, ResetVocab, ResetGrammar,
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
// user_time_tracking route patterns GetStoryStudentPerformance (scores.sql)
// buckets under that phase. Keep the two in sync.
type phaseSpec struct {
	tables        map[string]rowDeleter
	routePatterns []string
}

var phaseSpecs = map[ResetPhase]phaseSpec{
	ResetVideo: {routePatterns: []string{"%video%", "%audio%"}},
	ResetIdentify: {
		tables: map[string]rowDeleter{
			"identify_correct_answers": func(ctx context.Context, u string, s int32) (int64, error) {
				return queries.DeleteUserStoryIdentifyCorrect(ctx, db.DeleteUserStoryIdentifyCorrectParams{UserID: u, StoryID: s})
			},
			"identify_incorrect_answers": func(ctx context.Context, u string, s int32) (int64, error) {
				return queries.DeleteUserStoryIdentifyIncorrect(ctx, db.DeleteUserStoryIdentifyIncorrectParams{UserID: u, StoryID: s})
			},
		},
		routePatterns: []string{"%identify%"},
	},
	ResetTranslate: {
		tables: map[string]rowDeleter{
			"translation_requests": func(ctx context.Context, u string, s int32) (int64, error) {
				return queries.DeleteUserStoryTranslationRequest(ctx, db.DeleteUserStoryTranslationRequestParams{UserID: u, StoryID: s})
			},
		},
		routePatterns: []string{"%translate%"},
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
		routePatterns: []string{"%produce%"},
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
		routePatterns: []string{"%recall%"},
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
		routePatterns: []string{"%vocab%"},
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
		routePatterns: []string{"%grammar%"},
	},
}

// ResetUserStoryProgress deletes one student's progress on one story: either
// everything (ResetAll) or a single phase, including that phase's time rows.
// It runs in a transaction; deleting zero rows is not an error. A whole-story
// reset is two statements regardless of table count.
//
// No cache work is needed: per-user scores are never cached, and the
// time-tracking session cache holds only session IDs, not DB row references.
func ResetUserStoryProgress(ctx context.Context, userID string, storyID int32, phase ResetPhase) (ResetResult, error) {
	if !slices.Contains(ResetPhases, phase) {
		return ResetResult{}, fmt.Errorf("%w: %q", ErrInvalidResetPhase, phase)
	}

	result := ResetResult{Phase: phase, Deleted: map[string]int64{}}

	err := withTransaction(ctx, func(txCtx context.Context) error {
		if phase == ResetAll {
			return resetAll(txCtx, userID, storyID, result.Deleted)
		}

		spec := phaseSpecs[phase]
		for table, del := range spec.tables {
			n, err := del(txCtx, userID, storyID)
			if err != nil {
				return fmt.Errorf("delete %s: %w", table, err)
			}
			result.Deleted[table] = n
		}
		for _, pattern := range spec.routePatterns {
			n, err := queries.DeleteUserStoryTimeTrackingByRoute(txCtx, db.DeleteUserStoryTimeTrackingByRouteParams{
				UserID: userID, StoryID: pgtype.Int4{Int32: storyID, Valid: true}, Route: pattern,
			})
			if err != nil {
				return fmt.Errorf("delete time tracking: %w", err)
			}
			result.Deleted["time_tracking"] += n
		}
		return nil
	})
	if err != nil {
		return ResetResult{}, err
	}
	return result, nil
}

func resetAll(ctx context.Context, userID string, storyID int32, deleted map[string]int64) error {
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

	n, err := queries.DeleteUserStoryTimeTracking(ctx, db.DeleteUserStoryTimeTrackingParams{UserID: userID, StoryID: pgtype.Int4{Int32: storyID, Valid: true}})
	if err != nil {
		return fmt.Errorf("delete time tracking: %w", err)
	}
	deleted["time_tracking"] = n
	return nil
}
