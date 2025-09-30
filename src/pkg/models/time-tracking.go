package models

import (
	"context"
	"crypto/md5"
	"fmt"
	"glossias/src/pkg/generated/db"
	"log/slog"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

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

// StartTimeTracking creates a new time tracking entry and returns the tracking ID
// If an active session exists, it returns that ID instead of creating a new one
// If an active session is older than 5 minutes, it closes it and creates a new one
// Uses separate table for anonymous users
func StartTimeTracking(ctx context.Context, userID, route string, storyID *int32, clientIP string) (int32, error) {
	var pgStoryID pgtype.Int4
	if storyID != nil {
		pgStoryID = pgtype.Int4{Int32: *storyID, Valid: true}
	}

	isAnonymous := userID == "anonymous"

	if isAnonymous {
		sessionID := generateSessionID(clientIP)

		// Check for existing active anonymous session
		activeEntry, err := queries.GetActiveAnonymousTimeEntry(ctx, db.GetActiveAnonymousTimeEntryParams{
			SessionID: sessionID,
			Route:     route,
			StoryID:   pgStoryID,
		})

		if err == nil {
			// Found active entry, check if it's within 5 minutes
			if time.Since(activeEntry.StartedAt.Time) <= 5*time.Minute {
				return activeEntry.TrackingID, nil
			}

			// Close old entry before creating new one
			endTime := time.Now()
			totalSeconds := int32(endTime.Sub(activeEntry.StartedAt.Time).Seconds())
			err = queries.CloseAnonymousTimeEntry(ctx, db.CloseAnonymousTimeEntryParams{
				TrackingID:       activeEntry.TrackingID,
				EndedAt:          pgtype.Timestamp{Time: endTime, Valid: true},
				TotalTimeSeconds: pgtype.Int4{Int32: totalSeconds, Valid: true},
			})
			if err != nil {
				return 0, err
			}
		}

		entry, err := queries.CreateAnonymousTimeEntry(ctx, db.CreateAnonymousTimeEntryParams{
			SessionID: sessionID,
			Route:     route,
			StoryID:   pgStoryID,
			StartedAt: pgtype.Timestamp{Time: time.Now(), Valid: true},
		})
		if err != nil {
			return 0, err
		}

		return entry.TrackingID, nil
	}

	// Authenticated user logic
	activeEntry, err := queries.GetActiveTimeEntry(ctx, db.GetActiveTimeEntryParams{
		UserID:  userID,
		Route:   route,
		StoryID: pgStoryID,
	})

	if err == nil {
		// Found active entry, check if it's within 5 minutes
		if time.Since(activeEntry.StartedAt.Time) <= 5*time.Minute {
			return activeEntry.TrackingID, nil
		}

		// Close old entry before creating new one
		endTime := time.Now()
		totalSeconds := int32(endTime.Sub(activeEntry.StartedAt.Time).Seconds())
		err = queries.CloseTimeEntry(ctx, db.CloseTimeEntryParams{
			TrackingID:       activeEntry.TrackingID,
			EndedAt:          pgtype.Timestamp{Time: endTime, Valid: true},
			TotalTimeSeconds: pgtype.Int4{Int32: totalSeconds, Valid: true},
		})
		if err != nil {
			return 0, err
		}
	}

	entry, err := queries.CreateTimeEntry(ctx, db.CreateTimeEntryParams{
		UserID:    userID,
		Route:     route,
		StoryID:   pgStoryID,
		StartedAt: pgtype.Timestamp{Time: time.Now(), Valid: true},
	})
	if err != nil {
		return 0, err
	}

	return entry.TrackingID, nil
}

// EndTimeTrackingByID updates a time tracking entry with end time and duration using tracking ID
// If the entry is already ended, updates with the later end time
// Handles both authenticated and anonymous tracking
func EndTimeTrackingByID(ctx context.Context, trackingID int32) error {
	// Try authenticated table first
	entry, err := queries.GetTimeEntryByID(ctx, trackingID)
	if err == nil {
		endTime := time.Now()
		totalSeconds := int32(endTime.Sub(entry.StartedAt.Time).Seconds())

		_, err = queries.UpdateTimeEntry(ctx, db.UpdateTimeEntryParams{
			TrackingID:       trackingID,
			EndedAt:          pgtype.Timestamp{Time: endTime, Valid: true},
			TotalTimeSeconds: pgtype.Int4{Int32: totalSeconds, Valid: true},
		})
		return err
	}

	// Try anonymous table
	anonEntry, err := queries.GetAnonymousTimeEntryByID(ctx, trackingID)
	if err != nil {
		return err // Not found in either table
	}

	endTime := time.Now()
	totalSeconds := int32(endTime.Sub(anonEntry.StartedAt.Time).Seconds())

	_, err = queries.UpdateAnonymousTimeEntry(ctx, db.UpdateAnonymousTimeEntryParams{
		TrackingID:       trackingID,
		EndedAt:          pgtype.Timestamp{Time: endTime, Valid: true},
		TotalTimeSeconds: pgtype.Int4{Int32: totalSeconds, Valid: true},
	})
	return err
}

// EndTimeTracking updates a time tracking entry with end time and duration
func EndTimeTracking(ctx context.Context, trackingID int32, startTime time.Time, logger *slog.Logger) error {
	endTime := time.Now()
	totalSeconds := int32(endTime.Sub(startTime).Seconds())

	_, err := queries.UpdateTimeEntry(ctx, db.UpdateTimeEntryParams{
		TrackingID:       trackingID,
		EndedAt:          pgtype.Timestamp{Time: endTime, Valid: true},
		TotalTimeSeconds: pgtype.Int4{Int32: totalSeconds, Valid: true},
	})
	if err != nil {
		logger.Error("failed to update time entry", "error", err, "tracking_id", trackingID)
		return err
	}

	return nil
}

// ExtractStoryIDFromRoute extracts story ID from URL path if present
func ExtractStoryIDFromRoute(route string) *int32 {
	if !strings.Contains(route, "/stories/") {
		return nil
	}

	parts := strings.Split(route, "/")
	for i, part := range parts {
		if part == "stories" && i+1 < len(parts) {
			if id, err := strconv.Atoi(parts[i+1]); err == nil {
				storyID := int32(id)
				return &storyID
			}
			break
		}
	}
	return nil
}

// GetTimeEntriesForUser returns time tracking entries for a specific user
func GetTimeEntriesForUser(ctx context.Context, userID string) ([]db.UserTimeTracking, error) {
	entries, err := queries.GetTimeEntriesForUser(ctx, userID)
	if err != nil {
		return nil, err
	}

	return entries, nil
}

// GetTimeEntriesForStory returns time tracking entries for a specific story
func GetTimeEntriesForStory(ctx context.Context, storyID int32) ([]db.UserTimeTracking, error) {
	entries, err := queries.GetTimeEntriesForStory(ctx, pgtype.Int4{Int32: storyID, Valid: true})
	if err != nil {
		return nil, err
	}

	return entries, nil
}
