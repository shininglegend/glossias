package models_test

import (
	"glossias/src/pkg/models"
	"testing"
)

func TestCalculateScoreWithRetriesAllowed(t *testing.T) {
	tests := []struct {
		name string // description of this test case
		// Named input parameters for target function.
		correctCount   int64
		incorrectCount int64
		totalPossible  int64
		want           float64
	}{
		{
			name:           "0 correct",
			correctCount:   0,
			incorrectCount: 5,
			totalPossible:  5,
			want:           0.0,
		},
		{
			name:           "None possible",
			correctCount:   1,
			incorrectCount: 3,
			totalPossible:  0,
			want:           100.0,
		},
		{
			name:           "100%",
			correctCount:   5,
			incorrectCount: 0,
			totalPossible:  5,
			want:           100.0,
		},
		{
			name:           "4/5 completed, no incorrect",
			correctCount:   4,
			incorrectCount: 0,
			totalPossible:  5,
			want:           80.0,
		},
		{
			name:           "5/5 completed, 3 incorrect",
			correctCount:   5,
			incorrectCount: 3,
			totalPossible:  5,
			want:           62.5,
		},
		{
			name:           "4/5 completed, 4 incorrect",
			correctCount:   4,
			incorrectCount: 4,
			totalPossible:  5,
			want:           40.0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := models.CalculateScoreWithRetriesAllowed(tt.correctCount, tt.incorrectCount, tt.totalPossible)
			if got != tt.want {
				t.Errorf("CalculateScoreWithRetriesAllowed() = %v, want %v", got, tt.want)
			}
		})
	}
}
