package stories

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"glossias/src/apis/types"
	"glossias/src/auth"
	"glossias/src/pkg/database"
	"glossias/src/pkg/models"

	"github.com/gorilla/mux"
	"github.com/jackc/pgx/v5/pgtype"
)

// assertQueryBudget mirrors src/apis/handlers/querybudget_test.go: serve req
// and fail if the handler exceeded max DB calls.
func assertQueryBudget(t *testing.T, max int, handler http.HandlerFunc, req *http.Request) *httptest.ResponseRecorder {
	t.Helper()
	ctx, _ := database.WithQueryCounter(req.Context())
	req = req.WithContext(ctx)
	rr := httptest.NewRecorder()
	handler(rr, req)
	if got := database.QueryCount(ctx); got > max {
		t.Errorf("%s %s made %d DB queries, budget %d", req.Method, req.URL.Path, got, max)
	}
	return rr
}

func TestResetStudentProgressHandler(t *testing.T) {
	h := NewHandler(slog.New(slog.DiscardHandler))

	tests := []struct {
		name           string
		phase          string
		authUserID     string
		hasAuth        bool
		courseAdmin    bool
		expectedStatus int
		queryBudget    int
	}{
		{name: "reset all", phase: "", authUserID: "admin-1", hasAuth: true, courseAdmin: true, expectedStatus: http.StatusOK, queryBudget: 6},
		{name: "reset current attempt", phase: "exercises", authUserID: "admin-1", hasAuth: true, courseAdmin: true, expectedStatus: http.StatusOK, queryBudget: 5},
		{name: "reset single phase", phase: "identify", authUserID: "admin-1", hasAuth: true, courseAdmin: true, expectedStatus: http.StatusOK, queryBudget: 6},
		{name: "reset video (time only)", phase: "video", authUserID: "admin-1", hasAuth: true, courseAdmin: true, expectedStatus: http.StatusOK, queryBudget: 5},
		{name: "invalid phase", phase: "bogus", authUserID: "admin-1", hasAuth: true, courseAdmin: true, expectedStatus: http.StatusBadRequest, queryBudget: 3},
		{name: "unauthorized without user", phase: "all", hasAuth: false, expectedStatus: http.StatusUnauthorized, queryBudget: 0},
		{name: "unauthorized non-admin", phase: "all", authUserID: "student-1", hasAuth: true, courseAdmin: false, expectedStatus: http.StatusUnauthorized, queryBudget: 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDB := database.NewMockDBTX()
			// GetUser is left unstubbed (ErrNoRows => not super admin) so the
			// course-admin path is exercised: GetCourseIdForStory -> IsUserCourseAdmin.
			mockDB.StubQuery("GetCourseIdForStory", [][]any{{pgtype.Int4{Int32: 3, Valid: true}}}, nil)
			mockDB.StubQuery("IsUserCourseAdmin", [][]any{{tt.courseAdmin}}, nil)
			mockDB.StubQuery("ResetUserStoryAnswers", [][]any{{
				int64(1), int64(2), int64(0), int64(0), int64(1), int64(3), int64(1), int64(2), int64(2), int64(5), int64(0),
			}}, nil)
			models.SetDB(mockDB)
			defer models.SetDB(struct{}{})

			url := "/api/admin/stories/7/students/student-9/progress"
			if tt.phase != "" {
				url += "?phase=" + tt.phase
			}
			req := httptest.NewRequest(http.MethodDelete, url, nil)
			req = mux.SetURLVars(req, map[string]string{"id": "7", "userId": "student-9"})
			if tt.hasAuth {
				req = req.WithContext(context.WithValue(req.Context(), auth.UserIDKey, tt.authUserID))
			}

			rr := assertQueryBudget(t, tt.queryBudget, h.resetStudentProgressHandler, req)

			if rr.Code != tt.expectedStatus {
				t.Fatalf("status = %d, want %d; body: %s", rr.Code, tt.expectedStatus, rr.Body.String())
			}
			if tt.expectedStatus != http.StatusOK {
				return
			}

			var resp types.APIResponse
			if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
				t.Fatalf("decode: %v", err)
			}
			if !resp.Success {
				t.Errorf("expected success=true")
			}
			data, _ := resp.Data.(map[string]any)
			wantPhase := tt.phase
			if wantPhase == "" {
				wantPhase = "all"
			}
			if data["phase"] != wantPhase {
				t.Errorf("phase = %v, want %s", data["phase"], wantPhase)
			}
			deleted, ok := data["deleted"].(map[string]any)
			if !ok {
				t.Errorf("expected deleted map in response, got %v", data["deleted"])
			}
			// Only a whole-story reset reaches the archived attempts; every
			// other phase acts on the live (current) attempt alone.
			_, touchedArchive := deleted["story_attempts"]
			if archiveCalls := len(mockDB.Calls("DELETE FROM story_attempts")); (wantPhase == "all") != (archiveCalls == 1 && touchedArchive) {
				t.Errorf("phase %s: story_attempts delete calls = %d, reported = %v", wantPhase, archiveCalls, touchedArchive)
			}
		})
	}
}

func TestStoryStudentDrilldownHandler(t *testing.T) {
	h := NewHandler(slog.New(slog.DiscardHandler))

	tests := []struct {
		name           string
		authUserID     string
		hasAuth        bool
		courseAdmin    bool
		studentFound   bool
		expectedStatus int
		queryBudget    int
	}{
		{name: "happy path", authUserID: "admin-1", hasAuth: true, courseAdmin: true, studentFound: true, expectedStatus: http.StatusOK, queryBudget: 12},
		{name: "student not found", authUserID: "admin-1", hasAuth: true, courseAdmin: true, studentFound: false, expectedStatus: http.StatusNotFound, queryBudget: 4},
		{name: "unauthorized without user", hasAuth: false, expectedStatus: http.StatusUnauthorized, queryBudget: 0},
		{name: "unauthorized non-admin", authUserID: "student-1", hasAuth: true, courseAdmin: false, expectedStatus: http.StatusUnauthorized, queryBudget: 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDB := database.NewMockDBTX()
			mockDB.StubQuery("GetCourseIdForStory", [][]any{{pgtype.Int4{Int32: 3, Valid: true}}}, nil)
			mockDB.StubQuery("IsUserCourseAdmin", [][]any{{tt.courseAdmin}}, nil)
			if tt.studentFound {
				mockDB.StubQuery("GetStudentStoryHeader", [][]any{{"student-9", "Student Nine", "nine@example.com", "A Story"}}, nil)
			}
			// GetUserStoryTimeTracking sums come back as interface{} columns.
			mockDB.StubQuery("GetUserStoryTimeTracking", [][]any{{
				int64(10), int64(20), int64(30), int64(40), int64(50), int64(60), int64(70),
			}}, nil)
			// The answer-log, translation and produce queries stay unstubbed:
			// list queries scan zero rows and GetTranslationRequest returns
			// ErrNoRows, which the model treats as "phase never started".
			models.SetDB(mockDB)
			defer models.SetDB(struct{}{})

			req := httptest.NewRequest(http.MethodGet, "/api/admin/stories/7/students/student-9", nil)
			req = mux.SetURLVars(req, map[string]string{"id": "7", "userId": "student-9"})
			if tt.hasAuth {
				req = req.WithContext(context.WithValue(req.Context(), auth.UserIDKey, tt.authUserID))
			}

			rr := assertQueryBudget(t, tt.queryBudget, h.storyStudentDrilldownHandler, req)

			if rr.Code != tt.expectedStatus {
				t.Fatalf("status = %d, want %d; body: %s", rr.Code, tt.expectedStatus, rr.Body.String())
			}
			if tt.expectedStatus != http.StatusOK {
				return
			}

			var resp types.APIResponse
			if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
				t.Fatalf("decode: %v", err)
			}
			if !resp.Success {
				t.Errorf("expected success=true")
			}
			data, _ := resp.Data.(map[string]any)
			if data["user_id"] != "student-9" || data["story_title"] != "A Story" {
				t.Errorf("unexpected header fields: %v", data)
			}
			for _, key := range []string{"identify_answers", "translate", "produce_segments", "recall_attempts", "time"} {
				if _, ok := data[key]; !ok {
					t.Errorf("missing %q in response: %v", key, data)
				}
			}
			translate, _ := data["translate"].(map[string]any)
			if translate["started"] != false {
				t.Errorf("expected translate.started=false for a fresh student, got %v", translate)
			}
			timeData, _ := data["time"].(map[string]any)
			if timeData["identify_seconds"] != float64(50) {
				t.Errorf("identify_seconds = %v, want 50", timeData["identify_seconds"])
			}
		})
	}
}

func TestStudentAttemptsHandler(t *testing.T) {
	h := NewHandler(slog.New(slog.DiscardHandler))

	tests := []struct {
		name           string
		authUserID     string
		hasAuth        bool
		courseAdmin    bool
		studentFound   bool
		expectedStatus int
		queryBudget    int
	}{
		{name: "happy path", authUserID: "admin-1", hasAuth: true, courseAdmin: true, studentFound: true, expectedStatus: http.StatusOK, queryBudget: 11},
		{name: "student not found", authUserID: "admin-1", hasAuth: true, courseAdmin: true, studentFound: false, expectedStatus: http.StatusNotFound, queryBudget: 4},
		{name: "unauthorized non-admin", authUserID: "student-1", hasAuth: true, courseAdmin: false, expectedStatus: http.StatusUnauthorized, queryBudget: 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDB := database.NewMockDBTX()
			mockDB.StubQuery("GetCourseIdForStory", [][]any{{pgtype.Int4{Int32: 3, Valid: true}}}, nil)
			mockDB.StubQuery("IsUserCourseAdmin", [][]any{{tt.courseAdmin}}, nil)
			if tt.studentFound {
				mockDB.StubQuery("GetStudentStoryHeader", [][]any{{"student-9", "Student Nine", "nine@example.com", "A Story"}}, nil)
			}
			mockDB.StubQuery("ListUserStoryAttemptSnapshots", [][]any{
				{int64(10), int32(1), pgtype.Timestamptz{Valid: true}, []byte(`{"story_title":"A Story","overall_accuracy":63.1}`)},
			}, nil)
			mockDB.StubQuery("GetUserStoryScoreSummary", [][]any{{
				int32(0), int32(0), int32(0), int32(0), int32(2), int32(0), int32(0), int32(0), int32(0), int32(0), float64(0), int32(0),
			}}, nil)
			mockDB.StubQuery("GetUserStoryTimeTracking", [][]any{{
				int64(0), int64(0), int64(0), int64(40), int64(50), int64(0), int64(0),
			}}, nil)
			mockDB.StubQuery("GetStoryPhaseTotals", [][]any{{int32(0), int32(0), int32(6), int32(2), int32(5)}}, nil)
			models.SetDB(mockDB)
			defer models.SetDB(struct{}{})

			req := httptest.NewRequest(http.MethodGet, "/api/admin/stories/7/students/student-9/attempts", nil)
			req = mux.SetURLVars(req, map[string]string{"id": "7", "userId": "student-9"})
			if tt.hasAuth {
				req = req.WithContext(context.WithValue(req.Context(), auth.UserIDKey, tt.authUserID))
			}

			rr := assertQueryBudget(t, tt.queryBudget, h.studentAttemptsHandler, req)
			if rr.Code != tt.expectedStatus {
				t.Fatalf("status = %d, want %d; body: %s", rr.Code, tt.expectedStatus, rr.Body.String())
			}
			if tt.expectedStatus != http.StatusOK {
				return
			}

			var resp struct {
				Success bool                    `json:"success"`
				Data    []models.AttemptSummary `json:"data"`
			}
			if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
				t.Fatalf("decode: %v", err)
			}
			if len(resp.Data) != 2 {
				t.Fatalf("attempts = %+v, want archived + live", resp.Data)
			}
			if a := resp.Data[0]; a.Number != 1 || a.IsCurrent || a.Score == nil || a.Score.OverallAccuracy != 63.1 || len(a.Stages) != 0 {
				t.Errorf("archived = %+v", a)
			}
			live := resp.Data[1]
			if live.Number != 2 || !live.IsCurrent || live.Score == nil || live.Score.StoryTitle != "A Story" {
				t.Errorf("live = %+v", live)
			}
			if len(live.Stages) != 2 || live.Stages[0].Phase != models.ResetVideo || live.Stages[1].Phase != models.ResetIdentify {
				t.Errorf("live stages = %+v, want video + identify", live.Stages)
			}
		})
	}
}

func TestDeleteStudentAttemptHandler(t *testing.T) {
	h := NewHandler(slog.New(slog.DiscardHandler))

	tests := []struct {
		name           string
		attempt        string
		exists         bool
		courseAdmin    bool
		expectedStatus int
		queryBudget    int
	}{
		{name: "deletes and renumbers", attempt: "1", exists: true, courseAdmin: true, expectedStatus: http.StatusOK, queryBudget: 6},
		{name: "missing attempt", attempt: "5", exists: false, courseAdmin: true, expectedStatus: http.StatusNotFound, queryBudget: 4},
		{name: "invalid attempt", attempt: "0", exists: true, courseAdmin: true, expectedStatus: http.StatusBadRequest, queryBudget: 3},
		{name: "unauthorized non-admin", attempt: "1", exists: true, courseAdmin: false, expectedStatus: http.StatusUnauthorized, queryBudget: 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDB := database.NewMockDBTX()
			mockDB.StubQuery("GetCourseIdForStory", [][]any{{pgtype.Int4{Int32: 3, Valid: true}}}, nil)
			mockDB.StubQuery("IsUserCourseAdmin", [][]any{{tt.courseAdmin}}, nil)
			if tt.exists {
				mockDB.StubExecRows("AND attempt_number = $3", 1)
				mockDB.StubQuery("ListUserStoryAttemptsAfter", [][]any{{int64(20), int32(2)}}, nil)
			}
			models.SetDB(mockDB)
			defer models.SetDB(struct{}{})

			req := httptest.NewRequest(http.MethodDelete, "/api/admin/stories/7/students/student-9/attempts/"+tt.attempt, nil)
			req = mux.SetURLVars(req, map[string]string{"id": "7", "userId": "student-9", "attempt": tt.attempt})
			req = req.WithContext(context.WithValue(req.Context(), auth.UserIDKey, "admin-1"))

			rr := assertQueryBudget(t, tt.queryBudget, h.deleteStudentAttemptHandler, req)
			if rr.Code != tt.expectedStatus {
				t.Fatalf("status = %d, want %d; body: %s", rr.Code, tt.expectedStatus, rr.Body.String())
			}
			shifts := mockDB.Calls("UPDATE story_attempts SET attempt_number")
			if tt.expectedStatus != http.StatusOK {
				if len(shifts) != 0 {
					t.Errorf("renumbered on a failed delete: %+v", shifts)
				}
				return
			}
			if len(shifts) != 1 || shifts[0].Args[1] != int32(1) {
				t.Errorf("renumber calls = %+v, want attempt 2 -> 1", shifts)
			}
			// Never the whole-story reset path.
			if n := len(mockDB.Calls("ResetUserStoryAnswers")); n != 0 {
				t.Errorf("deleting an archived attempt touched live answers (%d calls)", n)
			}
		})
	}
}

func TestStudentRoutesResolve(t *testing.T) {
	h := NewHandler(slog.New(slog.DiscardHandler))
	router := mux.NewRouter()
	h.RegisterRoutes(router.PathPrefix("/api/admin").Subrouter())

	cases := []struct {
		method, path, wantTemplate string
	}{
		{http.MethodGet, "/api/admin/stories/7/students", "/api/admin/stories/{id:[0-9]+}/students"},
		{http.MethodGet, "/api/admin/stories/7/students/user_abc", "/api/admin/stories/{id:[0-9]+}/students/{userId}"},
		{http.MethodDelete, "/api/admin/stories/7/students/user_abc/progress", "/api/admin/stories/{id:[0-9]+}/students/{userId}/progress"},
		{http.MethodGet, "/api/admin/stories/7/students/user_abc/attempts", "/api/admin/stories/{id:[0-9]+}/students/{userId}/attempts"},
		{http.MethodDelete, "/api/admin/stories/7/students/user_abc/attempts/2", "/api/admin/stories/{id:[0-9]+}/students/{userId}/attempts/{attempt:[0-9]+}"},
	}
	for _, c := range cases {
		req := httptest.NewRequest(c.method, c.path, nil)
		var match mux.RouteMatch
		if !router.Match(req, &match) || match.Route == nil {
			t.Fatalf("%s %s did not match any route", c.method, c.path)
		}
		got, _ := match.Route.GetPathTemplate()
		if got != c.wantTemplate {
			t.Errorf("%s %s matched %q, want %q", c.method, c.path, got, c.wantTemplate)
		}
	}
}
