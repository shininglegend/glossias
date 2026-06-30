package models

import (
	"context"
	"errors"
	"glossias/src/pkg/database"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

func TestGetStoryCourseID_Success(t *testing.T) {
	// 1. Create a new mock database connection
	mockDBTX := database.NewMockDBTX()

	// 2. Set up expectations for the GetStory query
	// GetStory scans: StoryID, WeekNumber, DayLetter, VideoUrl, LastRevision, AuthorID, AuthorName, CourseID
	expectedCourseID := int32(101)
	mockRow := []interface{}{
		int32(42),                             // StoryID
		int32(1),                              // WeekNumber
		"A",                                   // DayLetter
		pgtype.Text{String: "", Valid: false}, // VideoUrl
		pgtype.Timestamp{Valid: false},        // LastRevision
		"author-123",                          // AuthorID
		"Jane Doe",                            // AuthorName
		pgtype.Int4{Int32: expectedCourseID, Valid: true}, // CourseID
	}
	mockDBTX.StubQuery("SELECT s.story_id", [][]interface{}{mockRow}, nil)

	// 3. Inject our mock connection
	SetDB(mockDBTX)
	// Ensure we clean up by setting db back to nil/default afterwards
	defer func() {
		SetDB(struct{}{})
	}()

	// 4. Call the target function
	ctx := context.Background()
	courseID, err := GetStoryCourseID(ctx, 42)

	// 5. Assertions
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if courseID != expectedCourseID {
		t.Errorf("Expected CourseID %d, got %d", expectedCourseID, courseID)
	}
}

func TestGetStoryCourseID_NotFound(t *testing.T) {
	mockDBTX := database.NewMockDBTX()
	mockDBTX.StubQuery("SELECT s.story_id", nil, pgx.ErrNoRows)

	SetDB(mockDBTX)
	defer func() {
		SetDB(struct{}{})
	}()

	ctx := context.Background()
	_, err := GetStoryCourseID(ctx, 999)

	if !errors.Is(err, pgx.ErrNoRows) {
		t.Errorf("Expected pgx.ErrNoRows error, got %v", err)
	}
}
