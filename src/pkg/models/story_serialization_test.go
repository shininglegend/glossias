package models

import (
	"encoding/json"
	"testing"
	"time"
)

func TestNewStory(t *testing.T) {
	s := NewStory()
	if s == nil {
		t.Fatal("NewStory() returned nil")
	}
	if s.Metadata.Title == nil {
		t.Error("NewStory().Metadata.Title is nil")
	}
	if s.Metadata.GrammarPoints == nil {
		t.Error("NewStory().Metadata.GrammarPoints is nil")
	}
	if s.Content.Lines == nil {
		t.Error("NewStory().Content.Lines is nil")
	}
}

func TestStory_ToJSONAndFromJSON(t *testing.T) {
	original := NewStory()
	original.Metadata.StoryID = 42
	original.Metadata.WeekNumber = 2
	original.Metadata.DayLetter = "A"
	original.Metadata.Title["en"] = "English Title"
	original.Metadata.Author = Author{ID: "author-123", Name: "Jane Doe"}

	now := time.Date(2026, 6, 4, 12, 0, 0, 0, time.UTC)
	original.Metadata.LastRevision = &now

	original.Content.Lines = []StoryLine{
		{
			LineNumber: 1,
			Text:       "Hello world",
			Vocabulary: []VocabularyItem{
				{Word: "Hello", LexicalForm: "hello", Position: [2]int{0, 5}},
			},
		},
	}

	data, err := original.ToJSON()
	if err != nil {
		t.Fatalf("ToJSON() failed: %v", err)
	}

	var parsed Story
	err = parsed.FromJSON(data)
	if err != nil {
		t.Fatalf("FromJSON() failed: %v", err)
	}

	if parsed.Metadata.StoryID != original.Metadata.StoryID {
		t.Errorf("Expected StoryID %d, got %d", original.Metadata.StoryID, parsed.Metadata.StoryID)
	}
	if parsed.Metadata.Title["en"] != "English Title" {
		t.Errorf("Expected English Title, got %s", parsed.Metadata.Title["en"])
	}
	if parsed.Metadata.LastRevision == nil || !parsed.Metadata.LastRevision.Equal(now) {
		t.Errorf("Expected LastRevision %v, got %v", now, parsed.Metadata.LastRevision)
	}
	if len(parsed.Content.Lines) != 1 || parsed.Content.Lines[0].Text != "Hello world" {
		t.Errorf("Expected 1 line with 'Hello world', got %+v", parsed.Content.Lines)
	}
}

func TestStoryMetadata_MarshalUnmarshalJSON(t *testing.T) {
	// Test without LastRevision
	meta1 := StoryMetadata{
		StoryID:      1,
		WeekNumber:   1,
		DayLetter:    "B",
		LastRevision: nil,
	}
	data, err := json.Marshal(meta1)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var meta1Parsed StoryMetadata
	err = json.Unmarshal(data, &meta1Parsed)
	if err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}
	if meta1Parsed.LastRevision != nil {
		t.Errorf("Expected nil LastRevision, got %v", meta1Parsed.LastRevision)
	}

	// Test with LastRevision
	now := time.Date(2026, 6, 4, 12, 0, 0, 0, time.UTC)
	meta2 := StoryMetadata{
		StoryID:      2,
		WeekNumber:   2,
		DayLetter:    "C",
		LastRevision: &now,
	}
	data2, err := json.Marshal(meta2)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var meta2Parsed StoryMetadata
	err = json.Unmarshal(data2, &meta2Parsed)
	if err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}
	if meta2Parsed.LastRevision == nil || !meta2Parsed.LastRevision.Equal(now) {
		t.Errorf("Expected LastRevision %v, got %v", now, meta2Parsed.LastRevision)
	}
}
