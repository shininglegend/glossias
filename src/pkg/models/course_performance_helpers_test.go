package models

import (
	"testing"
)

func TestConvertToInt32(t *testing.T) {
	tests := []struct {
		name  string
		input any
		want  int32
	}{
		{
			name:  "nil input",
			input: nil,
			want:  0,
		},
		{
			name:  "int32 input",
			input: int32(42),
			want:  42,
		},
		{
			name:  "int64 input",
			input: int64(100),
			want:  100,
		},
		{
			name:  "int input",
			input: 99,
			want:  99,
		},
		{
			name:  "unsupported type input (string)",
			input: "hello",
			want:  0,
		},
		{
			name:  "unsupported type input (float)",
			input: 42.5,
			want:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := convertToInt32(tt.input)
			if got != tt.want {
				t.Errorf("convertToInt32() = %v, want %v", got, tt.want)
			}
		})
	}
}
