package models

import (
	"context"
	"glossias/src/pkg/database"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

func TestTimeTrackingSession_CacheOperations(t *testing.T) {
	// Initialize cache
	err := SetCache()
	if err != nil {
		t.Fatalf("failed to initialize cache: %v", err)
	}

	ctx := context.Background()
	userID := "user-99"
	route := "/stories/12"
	storyID := int32(12)

	// 1. Create a new session
	sessID1, err := MakeTimeTrackingSession(ctx, userID, route, &storyID)
	if err != nil {
		t.Fatalf("failed to create session: %v", err)
	}
	if sessID1 == "" {
		t.Fatal("expected non-empty session ID")
	}

	// 2. Fetch existing session (should match)
	sessID2, err := MakeTimeTrackingSession(ctx, userID, route, &storyID)
	if err != nil {
		t.Fatalf("failed to create second session: %v", err)
	}
	if sessID1 != sessID2 {
		t.Errorf("expected session IDs to match, got %s and %s", sessID1, sessID2)
	}

	// 3. Retrieve session details
	session, err := GetTimeTrackingBySessionID(ctx, sessID1)
	if err != nil {
		t.Fatalf("failed to retrieve session: %v", err)
	}
	if session == nil {
		t.Fatal("expected retrieved session to be non-nil")
	}
	if session.UserID != userID || session.Route != route || *session.StoryID != storyID {
		t.Errorf("retrieved session details mismatch: %+v", session)
	}

	// 4. Invalidate session
	InvalidateTimeTrackingSession(ctx, sessID1)
	sessionAfterInvalid, err := GetTimeTrackingBySessionID(ctx, sessID1)
	if err != nil {
		t.Fatalf("failed checking session after invalidation: %v", err)
	}
	if sessionAfterInvalid != nil {
		t.Errorf("expected session to be nil after invalidation")
	}
}

func TestTimeTrackingSession_Expiration(t *testing.T) {
	err := SetCache()
	if err != nil {
		t.Fatalf("failed to initialize cache: %v", err)
	}

	ctx := context.Background()
	userID := "user-exp"
	route := "/stories/1"
	storyID := int32(1)

	sessID, err := MakeTimeTrackingSession(ctx, userID, route, &storyID)
	if err != nil {
		t.Fatalf("failed to create session: %v", err)
	}

	// Fetch session from cache directly and modify its CreatedAt
	cacheKey := keyBuilder.TimeTrackingSession(sessID)
	var session TimeTrackingSession
	err = cacheInstance.GetJSON(cacheKey, &session)
	if err != nil {
		t.Fatalf("failed to fetch session from cache directly: %v", err)
	}

	// Move CreatedAt 3 hours in the past
	session.CreatedAt = time.Now().Add(-3 * time.Hour)
	err = cacheInstance.SetJSON(cacheKey, &session)
	if err != nil {
		t.Fatalf("failed to update session back to cache: %v", err)
	}

	// GetTimeTrackingBySessionID should find it expired, delete it, and return nil
	retrieved, err := GetTimeTrackingBySessionID(ctx, sessID)
	if err != nil {
		t.Fatalf("failed to call GetTimeTrackingBySessionID: %v", err)
	}
	if retrieved != nil {
		t.Errorf("expected expired session to return nil, got %+v", retrieved)
	}
}

func TestRecordTimeTracking_Accumulate(t *testing.T) {
	mockDB := database.NewMockDBTX()

	// Stub FindRecentSimilarTimeEntry to return a recent entry (ID = 456, time = 100s)
	// FindRecentSimilarTimeEntry returns FindRecentSimilarTimeEntryRow: TrackingID, TotalTimeSeconds
	mockRow := []interface{}{
		int32(456),                           // TrackingID
		pgtype.Int4{Int32: 100, Valid: true}, // TotalTimeSeconds
	}
	mockDB.StubQuery("FindRecentSimilarTimeEntry", [][]interface{}{mockRow}, nil)
	// Stub AccumulateTimeEntry
	mockDB.StubExec("AccumulateTimeEntry", nil)

	SetDB(mockDB)
	defer func() {
		SetDB(struct{}{})
	}()

	storyID := int32(5)
	err := RecordTimeTracking(context.Background(), "user-123", "/stories/5", &storyID, 3000) // 3000ms = 3s
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

func TestRecordTimeTracking_Create(t *testing.T) {
	mockDB := database.NewMockDBTX()

	// Stub FindRecentSimilarTimeEntry to return no rows (not found)
	mockDB.StubQuery("FindRecentSimilarTimeEntry", nil, pgx.ErrNoRows)
	// Stub CreateCompleteTimeEntry to return a mocked UserTimeTracking record
	// CreateCompleteTimeEntry scans returns UserTimeTracking: TrackingID, UserID, Route, StoryID, StartedAt, EndedAt, TotalTimeSeconds, CreatedAt
	mockRow := []interface{}{
		int32(789),                         // TrackingID
		"user-123",                         // UserID
		"/stories/5",                       // Route
		pgtype.Int4{Int32: 5, Valid: true}, // StoryID
		pgtype.Timestamp{Valid: false},     // StartedAt
		pgtype.Timestamp{Valid: false},     // EndedAt
		pgtype.Int4{Int32: 3, Valid: true}, // TotalTimeSeconds
		pgtype.Timestamp{Valid: false},     // CreatedAt
	}
	mockDB.StubQuery("CreateCompleteTimeEntry", [][]interface{}{mockRow}, nil)

	SetDB(mockDB)
	defer func() {
		SetDB(struct{}{})
	}()

	storyID := int32(5)
	err := RecordTimeTracking(context.Background(), "user-123", "/stories/5", &storyID, 3000)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}
