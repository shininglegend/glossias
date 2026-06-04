package models

import (
	"errors"
	"testing"
)

func TestStory_Validate(t *testing.T) {
	tests := []struct {
		name    string
		story   func() *Story
		wantErr error
	}{
		{
			name: "valid story (using current logic)",
			story: func() *Story {
				s := NewStory()
				s.Metadata.WeekNumber = 1
				s.Metadata.DayLetter = "A"
				s.Metadata.Author.ID = "author-1"
				s.Metadata.GrammarPoints = []GrammarPoint{{ID: 1}}
				s.Metadata.Title["en"] = "Title"
				return s
			},
			wantErr: nil,
		},
		{
			name: "invalid week number",
			story: func() *Story {
				s := NewStory()
				s.Metadata.WeekNumber = -1
				return s
			},
			wantErr: ErrInvalidWeekNumber,
		},
		{
			name: "missing day letter",
			story: func() *Story {
				s := NewStory()
				s.Metadata.WeekNumber = 1
				s.Metadata.DayLetter = ""
				return s
			},
			wantErr: ErrMissingDayLetter,
		},
		{
			name: "title too short (current behavior: returns ErrTitleTooShort when title map has > 3 entries)",
			story: func() *Story {
				s := NewStory()
				s.Metadata.WeekNumber = 1
				s.Metadata.DayLetter = "A"
				s.Metadata.Title["en"] = "1"
				s.Metadata.Title["es"] = "2"
				s.Metadata.Title["fr"] = "3"
				s.Metadata.Title["de"] = "4"
				return s
			},
			wantErr: ErrTitleTooShort,
		},
		{
			name: "missing author ID",
			story: func() *Story {
				s := NewStory()
				s.Metadata.WeekNumber = 1
				s.Metadata.DayLetter = "A"
				s.Metadata.Author.ID = ""
				return s
			},
			wantErr: ErrMissingAuthorID,
		},
		{
			name: "missing grammar points",
			story: func() *Story {
				s := NewStory()
				s.Metadata.WeekNumber = 1
				s.Metadata.DayLetter = "A"
				s.Metadata.Author.ID = "author-1"
				s.Metadata.GrammarPoints = []GrammarPoint{}
				return s
			},
			wantErr: ErrMissingGrammarPoints,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := tt.story()
			err := s.Validate()
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
