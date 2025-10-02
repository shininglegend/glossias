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
const SESSION_MAX_AGE = 2 * time.Hour // Sessions expire after 2 hours of inactivity

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
	CreatedAt time.Time
}

// MakeTimeTrackingSession creates or returns existing time tracking session
// Ensures only one active session per user/route/story combination
func MakeTimeTrackingSession(ctx context.Context, userID, route string, storyID *int32) (string, error) {
	// Check for existing active session for this user/route/story combination
	if cacheInstance != nil && keyBuilder != nil {
		activeKey := keyBuilder.ActiveTimeTrackingSession(userID, route, storyID)
		var existingSessionID string
		err := cacheInstance.GetJSON(activeKey, &existingSessionID)
		if err == nil && existingSessionID != "" {
			// Return existing session ID
			return existingSessionID, nil
		}
	}

	// Clear any old active sessions for this user on different routes
	// This handles navigation to a new page
	if cacheInstance != nil && keyBuilder != nil {
		ClearActiveSessionsForUser(ctx, userID, route, storyID)
	}

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
		CreatedAt: time.Now(),
	}

	if cacheInstance != nil && keyBuilder != nil {
		// Store session data
		cacheKey := keyBuilder.TimeTrackingSession(sessionID)
		_ = cacheInstance.SetJSON(cacheKey, session)

		// Mark this as the active session for this user/route/story
		activeKey := keyBuilder.ActiveTimeTrackingSession(userID, route, storyID)
		_ = cacheInstance.SetJSON(activeKey, sessionID)
	}

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

	// Check if session is too old (expired)
	if time.Since(session.CreatedAt) > SESSION_MAX_AGE {
		_ = cacheInstance.Delete(cacheKey)
		return nil, nil
	}

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

// ClearActiveSessionsForUser clears active sessions for a user when navigating to a new route
// Only clears sessions that don't match the current route/story combination
func ClearActiveSessionsForUser(ctx context.Context, userID, currentRoute string, currentStoryID *int32) {
	// Note: This is a simplified implementation. In a production system with many users,
	// you'd want to maintain a user -> sessions index in cache to avoid scanning.
	// For now, we rely on the cache TTL to naturally expire old sessions.
	// The main protection is that MakeTimeTrackingSession checks for existing sessions
	// before creating new ones, preventing duplicates for the same route.
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

		// Do NOT invalidate session - allow tab-switch-return pattern
		// Session will naturally expire after TTL or be replaced when user navigates to new route

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

	// Do NOT invalidate session - allow tab-switch-return pattern
	// Session will naturally expire after TTL or be replaced when user navigates to new route

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
