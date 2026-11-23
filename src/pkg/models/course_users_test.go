package models

import (
	"context"
	"errors"
	"testing"
)

func TestAddUserToCourseByEmailWithStatus(t *testing.T) {
	cases := []struct {
		status     string
		wantErr    error
		wantStored string
	}{
		{"active", nil, "active"},
		{"past", nil, "past"},
		{"future", nil, "future"},
		{"", nil, "active"},
		{"graduated", ErrInvalidStatus, ""},
	}
	for _, c := range cases {
		t.Run(c.status, func(t *testing.T) {
			mockDB := memoMock(t)
			mockDB.StubQuery("name: GetUserByEmail", [][]any{userRow("u1", false)}, nil)

			err := AddUserToCourseByEmailWithStatus(context.Background(), "u1@example.com", 7, c.status)
			if !errors.Is(err, c.wantErr) {
				t.Fatalf("err = %v, want %v", err, c.wantErr)
			}
			calls := mockDB.Calls("name: AddUserToCourse :exec")
			if c.wantErr != nil {
				if len(calls) != 0 {
					t.Fatalf("invalid status wrote %d enrollments", len(calls))
				}
				return
			}
			if len(calls) != 1 {
				t.Fatalf("AddUserToCourse ran %d times, want 1", len(calls))
			}
			if got := calls[0].Args[2]; got != c.wantStored {
				t.Errorf("stored status %v, want %q", got, c.wantStored)
			}
		})
	}
}
