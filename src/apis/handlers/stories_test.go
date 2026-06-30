package handlers

import (
	"context"
	"encoding/json"
	"glossias/src/apis/types"
	"glossias/src/auth"
	"glossias/src/pkg/database"
	"glossias/src/pkg/models"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
)

func TestGetCourseStories(t *testing.T) {
	// Initialize Handler with a discard logger
	logger := slog.New(slog.DiscardHandler)
	h := NewHandler(logger)

	tests := []struct {
		name           string
		courseIDStr    string
		authUserID     string
		hasAuthContext bool
		stubAccess     bool
		expectedStatus int
	}{
		{
			name:           "success",
			courseIDStr:    "101",
			authUserID:     "user-123",
			hasAuthContext: true,
			stubAccess:     true,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "invalid course ID",
			courseIDStr:    "abc",
			authUserID:     "user-123",
			hasAuthContext: true,
			stubAccess:     true,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "unauthorized (no context user)",
			courseIDStr:    "101",
			authUserID:     "",
			hasAuthContext: false,
			stubAccess:     true,
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "forbidden (no course access)",
			courseIDStr:    "101",
			authUserID:     "user-123",
			hasAuthContext: true,
			stubAccess:     false,
			expectedStatus: http.StatusForbidden,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set up Mock DB
			mockDB := database.NewMockDBTX()

			// Stub permission query: CanUserAccessCourse
			mockDB.StubQuery("CanUserAccessCourse", [][]interface{}{
				{tt.stubAccess},
			}, nil)

			// Stub stories query: GetCourseStoriesWithTitles
			if tt.stubAccess {
				mockDB.StubQuery("GetCourseStoriesWithTitles", [][]interface{}{
					{int32(1), int32(1), "A", "Story Title 1"},
				}, nil)
			}

			models.SetDB(mockDB)
			defer func() {
				models.SetDB(struct{}{})
			}()

			// Build request
			req := httptest.NewRequest("GET", "/api/stories/by-course/"+tt.courseIDStr, nil)

			// Set route variables using gorilla mux context helper
			req = mux.SetURLVars(req, map[string]string{"course_id": tt.courseIDStr})

			// Inject user ID to request context if requested
			if tt.hasAuthContext {
				ctx := context.WithValue(req.Context(), auth.UserIDKey, tt.authUserID)
				req = req.WithContext(ctx)
			}

			rr := httptest.NewRecorder()

			// Execute handler
			h.GetCourseStories(rr, req)

			// Assert status code
			if rr.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, rr.Code)
			}

			// For successful case, assert json body parsing
			if tt.expectedStatus == http.StatusOK {
				var resp types.APIResponse
				if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
					t.Fatalf("failed to unmarshal JSON response: %v", err)
				}
				if !resp.Success {
					t.Errorf("expected APIResponse.Success to be true, got false")
				}
			}
		})
	}
}
