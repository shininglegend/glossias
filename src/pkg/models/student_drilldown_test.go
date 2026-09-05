package models

import (
	"testing"
	"time"

	"glossias/src/pkg/generated/db"

	"github.com/jackc/pgx/v5/pgtype"
)

func recallRow(text string, correctPos, selectedPos int32, correct bool, at time.Time) db.GetUserStoryRecallAnswerLogRow {
	return db.GetUserStoryRecallAnswerLogRow{
		HebrewText:       text,
		CorrectPosition:  correctPos,
		SelectedPosition: selectedPos,
		Correct:          correct,
		AttemptedAt:      pgtype.Timestamp{Time: at, Valid: true},
	}
}

func TestGroupRecallAttempts(t *testing.T) {
	base := time.Date(2026, 8, 31, 10, 0, 0, 0, time.UTC)

	t.Run("empty log", func(t *testing.T) {
		if got := groupRecallAttempts(nil); len(got) != 0 {
			t.Errorf("expected no attempts, got %v", got)
		}
	})

	t.Run("two attempts split on repeated position", func(t *testing.T) {
		// Attempt 1: sentences A,B,C placed 1,2,3 — A correct, B/C swapped.
		// Attempt 2 (a minute later): all three correct.
		rows := []db.GetUserStoryRecallAnswerLogRow{
			recallRow("A", 1, 1, true, base),
			recallRow("C", 3, 2, false, base.Add(time.Millisecond)),
			recallRow("B", 2, 3, false, base.Add(2*time.Millisecond)),
			recallRow("A", 1, 1, true, base.Add(time.Minute)),
			recallRow("B", 2, 2, true, base.Add(time.Minute+time.Millisecond)),
			recallRow("C", 3, 3, true, base.Add(time.Minute+2*time.Millisecond)),
		}

		attempts := groupRecallAttempts(rows)
		if len(attempts) != 2 {
			t.Fatalf("expected 2 attempts, got %d: %v", len(attempts), attempts)
		}
		if attempts[0].AllCorrect {
			t.Errorf("first attempt should not be all correct")
		}
		if !attempts[1].AllCorrect {
			t.Errorf("second attempt should be all correct")
		}
		if len(attempts[0].Placements) != 3 || len(attempts[1].Placements) != 3 {
			t.Fatalf("expected 3 placements per attempt: %v", attempts)
		}
		// Placements come back ordered by the position the student filled.
		first := attempts[0].Placements
		if first[1].HebrewText != "C" || first[1].SelectedPosition != 2 || first[1].CorrectPosition != 3 {
			t.Errorf("placement 2 of attempt 1 = %+v, want sentence C at selected 2 / correct 3", first[1])
		}
	})
}
