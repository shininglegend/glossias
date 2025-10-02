package models

import (
	"context"
	"crypto/md5"
	"fmt"
	"glossias/src/pkg/generated/db"
	"net"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

const DEDUP_WINDOW = 30 * time.Second
const ELAPSED_TIME_TOLERANCE = 5 * time.Second

// generateSessionID creates a session ID for anonymous users using IP address
func generateSessionID(ip string) string {
	// Remove port if present
	host, _, err := net.SplitHostPort(ip)
	if err != nil {
		host = ip
	}

	// Hash the IP to create anonymous session ID
	hash := md5.Sum([]byte(host + time.Now().Format("2006-01-02"))) // Daily rotation
	return fmt.Sprintf("anon_%x", hash[:8])
}

type TimeTrackingSession struct {
	SessionID string
	UserID    string
	Route     string
	StoryID   *int32
}

// MakeTimeTrackingSession creates a new time tracking entry and returns the tracking ID
func MakeTimeTrackingSession(ctx context.Context, userID, route string, storyID *int32) (string, error) {
	// Generate cryptographically secure UUID with user prefix for easier cache identification
	userPrefix := userID
	if len(userID) > 5 {
		userPrefix = userID[:5]
	}
	sessionID := fmt.Sprintf("%s_%s", userPrefix, uuid.New().String())

	// Add to cache to track active sessions
	session := &TimeTrackingSession{
		SessionID: sessionID,
		UserID:    userID,
		Route:     route,
		StoryID:   storyID,
	}

	if cacheInstance != nil && keyBuilder != nil {
		cacheKey := keyBuilder.TimeTrackingSession(sessionID)
		_ = cacheInstance.SetJSON(cacheKey, session)
	}

	fmt.Println("DEBUG: Created time tracking session", sessionID, "for user", userID, "route", route, "story", storyID)

	return sessionID, nil
}

// GetTimeTrackingBySessionID retrieves a time tracking session by its ID
func GetTimeTrackingBySessionID(ctx context.Context, sessionID string) (*TimeTrackingSession, error) {
	if cacheInstance == nil || keyBuilder == nil {
		return nil, nil
	}

	cacheKey := keyBuilder.TimeTrackingSession(sessionID)
	var session TimeTrackingSession
	err := cacheInstance.GetJSON(cacheKey, &session)
	if err != nil {
		return nil, nil
	}

	fmt.Println("DEBUG: Retrieved time tracking session", sessionID, "for user", session.UserID, "route", session.Route, "story", session.StoryID)

	return &session, nil
}

// InvalidateTimeTrackingSession removes a session from active tracking
func InvalidateTimeTrackingSession(ctx context.Context, sessionID string) {
	if cacheInstance == nil || keyBuilder == nil {
		return
	}

	cacheKey := keyBuilder.TimeTrackingSession(sessionID)
	_ = cacheInstance.Delete(cacheKey)
}

// GetTimeEntriesForUser returns time tracking entries for a specific user
func GetTimeEntriesForUser(ctx context.Context, userID string) ([]db.UserTimeTracking, error) {
	entries, err := queries.GetTimeEntriesForUser(ctx, userID)
	if err != nil {
		return nil, err
	}

	return entries, nil
}

// RecordTimeTracking creates a complete time tracking entry in one call
// Uses client-provided elapsed time for accuracy
// Updates existing entry with larger time value if found within deduplication window
func RecordTimeTracking(ctx context.Context, userID, route string, storyID *int32, elapsedMs int32) error {
	var pgStoryID pgtype.Int4
	if storyID != nil {
		pgStoryID = pgtype.Int4{Int32: *storyID, Valid: true}
	}

	now := time.Now()
	totalSeconds := elapsedMs / 1000
	startTime := now.Add(-time.Duration(elapsedMs) * time.Millisecond)

	// Check for similar recent entry using single query
	recentEntry, err := queries.FindRecentSimilarTimeEntry(ctx, db.FindRecentSimilarTimeEntryParams{
		UserID:    userID,
		Route:     route,
		StoryID:   pgStoryID,
		CreatedAt: pgtype.Timestamp{Time: now.Add(-DEDUP_WINDOW), Valid: true},
	})

	if err == nil && recentEntry.TotalTimeSeconds.Valid {
		// Found recent similar entry, accumulate time
		err = queries.AccumulateTimeEntry(ctx, db.AccumulateTimeEntryParams{
			TrackingID:       recentEntry.TrackingID,
			TotalTimeSeconds: pgtype.Int4{Int32: totalSeconds, Valid: true},
			EndedAt:          pgtype.Timestamp{Time: now, Valid: true},
		})
		return err
	}

	// No recent similar entry found, create new one
	_, err = queries.CreateCompleteTimeEntry(ctx, db.CreateCompleteTimeEntryParams{
		UserID:           userID,
		Route:            route,
		StoryID:          pgStoryID,
		StartedAt:        pgtype.Timestamp{Time: startTime, Valid: true},
		EndedAt:          pgtype.Timestamp{Time: now, Valid: true},
		TotalTimeSeconds: pgtype.Int4{Int32: totalSeconds, Valid: true},
	})
	return err
}

// GetTimeEntriesForStory returns time tracking entries for a specific story
func GetTimeEntriesForStory(ctx context.Context, storyID int32) ([]db.UserTimeTracking, error) {
	entries, err := queries.GetTimeEntriesForStory(ctx, pgtype.Int4{Int32: storyID, Valid: true})
	if err != nil {
		return nil, err
	}

	return entries, nil
}
