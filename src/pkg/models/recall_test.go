package models

import (
	"context"
	"errors"
	"slices"
	"testing"

	"glossias/src/pkg/database"

	"github.com/jackc/pgx/v5/pgtype"
)

// stubRecallSentences wires a mock DB that returns three recall sentences for
// story 1, with IDs 10/11/12 at sequence positions 1/2/3.
func stubRecallSentences(t *testing.T) {
	t.Helper()

	mockDB := database.NewMockDBTX()
	rows := make([][]any, 0, 3)
	for i, id := range []int32{10, 11, 12} {
		rows = append(rows, []any{
			id,                        // id
			int32(1),                  // story_id
			int32(i + 1),              // sequence_order
			"sentence",                // hebrew_text
			pgtype.Int4{Valid: false}, // target_vocab_id
			pgtype.Text{Valid: false}, // image_path
			pgtype.Text{Valid: false}, // image_bucket
		})
	}
	mockDB.StubQuery("FROM recall_sentences", rows, nil)

	SetDB(mockDB)
	t.Cleanup(func() { SetDB(struct{}{}) })
}

func TestSaveRecallAttempt(t *testing.T) {
	tests := []struct {
		name    string
		order   []int
		want    []bool
		wantErr error
	}{
		{
			name:  "perfect order scores every position correct",
			order: []int{10, 11, 12},
			want:  []bool{true, true, true},
		},
		{
			name:  "swapped pair scores only the untouched position correct",
			order: []int{11, 10, 12},
			want:  []bool{false, false, true},
		},
		{
			name:  "fully reversed order scores only the middle correct",
			order: []int{12, 11, 10},
			want:  []bool{false, true, false},
		},
		{
			name:    "short ordering is rejected",
			order:   []int{10, 11},
			wantErr: ErrInvalidRecallOrder,
		},
		{
			name:    "sentence from another story is rejected",
			order:   []int{10, 11, 99},
			wantErr: ErrInvalidRecallOrder,
		},
		{
			name:    "duplicate sentence is rejected",
			order:   []int{10, 10, 11},
			wantErr: ErrInvalidRecallOrder,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stubRecallSentences(t)

			got, err := SaveRecallAttempt(context.Background(), "u1", 1, tt.order)

			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("SaveRecallAttempt() error = %v, want %v", err, tt.wantErr)
				}
				if got != nil {
					t.Errorf("SaveRecallAttempt() returned %v alongside an error, want nil", got)
				}
				return
			}

			if err != nil {
				t.Fatalf("SaveRecallAttempt() unexpected error: %v", err)
			}
			if !slices.Equal(got, tt.want) {
				t.Errorf("SaveRecallAttempt() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSaveRecallAttemptStoryWithoutSentences(t *testing.T) {
	mockDB := database.NewMockDBTX()
	mockDB.StubQuery("FROM recall_sentences", nil, nil)
	SetDB(mockDB)
	t.Cleanup(func() { SetDB(struct{}{}) })

	_, err := SaveRecallAttempt(context.Background(), "u1", 1, []int{10})
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("SaveRecallAttempt() error = %v, want ErrNotFound", err)
	}
}
