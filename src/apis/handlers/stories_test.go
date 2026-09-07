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
	"github.com/jackc/pgx/v5/pgtype"
)

func TestGetCourseStories(t *testing.T) {
	// Initialize Handler with a discard logger
	logger := slog.New(slog.DiscardHandler)
	h := NewHandler(logger, nil)

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
			mockDB.StubQuery("CanUserAccessCourse", [][]any{
				{tt.stubAccess},
			}, nil)

			// Stub stories query: GetCourseStoriesWithTitles
			if tt.stubAccess {
				mockDB.StubQuery("GetCourseStoriesWithTitles", [][]any{
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

			// Execute handler. Budget: access check + stories query.
			rr := assertQueryBudget(t, 2, h.GetCourseStories, req)

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

func TestGetStories(t *testing.T) {
	logger := slog.New(slog.DiscardHandler)
	h := NewHandler(logger, nil)

	storyRows := [][]any{
		{int32(1), int32(1), "A", "Story 1", pgtype.Int4{}},
		{int32(2), int32(1), "B", "Story 2", pgtype.Int4{}},
		{int32(3), int32(2), "A", "Story 3", pgtype.Int4{}},
	}

	t.Run("student skips readiness", func(t *testing.T) {
		mockDB := database.NewMockDBTX()
		mockDB.StubQuery("GetAllStoriesForUser", storyRows, nil)
		mockDB.StubQuery("name: GetUser :one", [][]any{{
			"user-1", "u@example.com", "User", pgtype.Bool{Bool: false, Valid: true}, pgtype.Timestamp{}, pgtype.Timestamp{},
		}}, nil)
		mockDB.StubQuery("IsUserAdminOfAnyCourse", [][]any{{false}}, nil)
		mockDB.StubQuery("GetUserStoriesPageCompletion", [][]any{
			{int32(1), int32(4), int32(0), false, int32(5), int32(0), int32(2), int32(0), int32(0)},
			{int32(2), int32(4), int32(2), false, int32(5), int32(0), int32(2), int32(0), int32(0)},
			{int32(3), int32(4), int32(4), true, int32(5), int32(5), int32(2), int32(2), int32(0)},
		}, nil)
		models.SetDB(mockDB)
		t.Cleanup(func() { models.SetDB(struct{}{}) })

		req := httptest.NewRequest("GET", "/api/stories", nil)
		req = req.WithContext(context.WithValue(req.Context(), auth.UserIDKey, "user-1"))

		// List + GetUser + course-admin check + batched page completion.
		rr := assertQueryBudget(t, 4, h.GetStories, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("status %d", rr.Code)
		}
		stories := decodeStories(t, rr)
		if len(stories) != 3 {
			t.Fatalf("got %d stories, want 3", len(stories))
		}
		want := []struct {
			status, page, name string
		}{
			{"not_started", "video", "Watch"},
			{"in_progress", "identify", "Identify"},
			{"complete", "score", "Score"},
		}
		for i, story := range stories {
			if len(story.MissingPhases) != 0 {
				t.Errorf("student saw missing_phases on story %d: %v", story.ID, story.MissingPhases)
			}
			if story.Status != want[i].status || story.NextPage != want[i].page || story.NextPageName != want[i].name {
				t.Errorf("story %d status=%q next=%q/%q, want %q %q/%q",
					story.ID, story.Status, story.NextPage, story.NextPageName,
					want[i].status, want[i].page, want[i].name)
			}
		}
	})

	t.Run("admin readiness is batched", func(t *testing.T) {
		mockDB := database.NewMockDBTX()
		mockDB.StubQuery("GetAllStoriesForUser", storyRows, nil)
		mockDB.StubQuery("name: GetUser :one", [][]any{{
			"admin-1", "a@example.com", "Admin", pgtype.Bool{Bool: true, Valid: true}, pgtype.Timestamp{}, pgtype.Timestamp{},
		}}, nil)
		stubStoryReadinessQueries(mockDB, []int32{1, 2, 3})
		models.SetDB(mockDB)
		t.Cleanup(func() { models.SetDB(struct{}{}) })

		req := httptest.NewRequest("GET", "/api/stories", nil)
		req = req.WithContext(context.WithValue(req.Context(), auth.UserIDKey, "admin-1"))

		// List + super-admin GetUser + batched progress + 8 batched readiness queries.
		rr := assertQueryBudget(t, 11, h.GetStories, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("status %d", rr.Code)
		}
		stories := decodeStories(t, rr)
		if len(stories) != 3 {
			t.Fatalf("got %d stories, want 3", len(stories))
		}
		for _, story := range stories {
			if len(story.MissingPhases) == 0 {
				t.Errorf("admin expected missing_phases on incomplete story %d", story.ID)
			}
		}
	})
}

func stubStoryReadinessQueries(mockDB *database.MockDBTX, storyIDs []int32) {
	videoRows := make([][]any, len(storyIDs))
	for i, id := range storyIDs {
		videoRows[i] = []any{id, pgtype.Text{}}
	}
	mockDB.StubQuery("name: GetStoriesVideoURLs", videoRows, nil)
	mockDB.StubQuery("name: GetStoriesTargetVocabulary", nil, nil)
	mockDB.StubQuery("name: GetStoriesLexicalFormCounts", nil, nil)
	mockDB.StubQuery("name: GetStoriesProduceSegments", nil, nil)
	mockDB.StubQuery("name: GetStoriesProduceExplanations", nil, nil)
	mockDB.StubQuery("name: GetStoriesRecallSentences", nil, nil)
	mockDB.StubQuery("name: GetStoriesLines", nil, nil)
	mockDB.StubQuery("name: GetStoriesAudioFilesByLabel", nil, nil)
}

func decodeStories(t *testing.T, rr *httptest.ResponseRecorder) []types.Story {
	t.Helper()
	var resp types.APIResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if !resp.Success {
		t.Fatalf("expected success, got %#v", resp)
	}
	raw, err := json.Marshal(resp.Data)
	if err != nil {
		t.Fatalf("marshal data: %v", err)
	}
	var payload types.StoriesResponse
	if err := json.Unmarshal(raw, &payload); err != nil {
		t.Fatalf("unmarshal stories: %v", err)
	}
	return payload.Stories
}
