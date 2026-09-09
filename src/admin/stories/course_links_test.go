package stories

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"glossias/src/auth"
	"glossias/src/pkg/database"
	"glossias/src/pkg/models"

	"github.com/gorilla/mux"
	"github.com/jackc/pgx/v5/pgtype"
)

func TestLinkStoryCourse(t *testing.T) {
	h := NewHandler(slog.New(slog.DiscardHandler))

	mockDB := database.NewMockDBTX()
	mockDB.StubQuery("name: GetUser :one", [][]any{{
		"admin-1", "a@example.com", "Admin", pgtype.Bool{Bool: false, Valid: true}, pgtype.Timestamp{}, pgtype.Timestamp{},
	}}, nil)
	mockDB.StubQuery("IsUserCourseAdmin", [][]any{{true}}, nil)
	mockDB.StubQuery("IsUserAdminOfLinkedStory", [][]any{{true}}, nil)
	mockDB.StubQuery("name: GetCourse :one", [][]any{{
		int32(5), "LATN-102", "Latin 102", pgtype.Text{}, pgtype.Timestamp{}, pgtype.Timestamp{},
	}}, nil)
	models.SetDB(mockDB)
	t.Cleanup(func() { models.SetDB(struct{}{}) })

	body, _ := json.Marshal(map[string]int{"courseId": 5})
	req := httptest.NewRequest(http.MethodPost, "/api/admin/stories/7/courses", bytes.NewReader(body))
	req = mux.SetURLVars(req, map[string]string{"id": "7"})
	req = req.WithContext(context.WithValue(req.Context(), auth.UserIDKey, "admin-1"))

	rr := assertQueryBudget(t, 5, h.linkStoryCourse, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, body: %s", rr.Code, rr.Body.String())
	}
	if n := len(mockDB.Calls("INSERT INTO course_stories")); n != 1 {
		t.Errorf("link inserts = %d, want 1", n)
	}
}

func TestUnlinkStoryCourse_RejectsOwner(t *testing.T) {
	h := NewHandler(slog.New(slog.DiscardHandler))

	mockDB := database.NewMockDBTX()
	mockDB.StubQuery("name: GetUser :one", [][]any{{
		"admin-1", "a@example.com", "Admin", pgtype.Bool{Bool: false, Valid: true}, pgtype.Timestamp{}, pgtype.Timestamp{},
	}}, nil)
	mockDB.StubQuery("IsUserCourseAdmin", [][]any{{true}}, nil)
	mockDB.StubQuery("GetCourseIdForStory", [][]any{{pgtype.Int4{Int32: 3, Valid: true}}}, nil)
	models.SetDB(mockDB)
	t.Cleanup(func() { models.SetDB(struct{}{}) })

	req := httptest.NewRequest(http.MethodDelete, "/api/admin/stories/7/courses/3", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "7", "courseId": "3"})
	req = req.WithContext(context.WithValue(req.Context(), auth.UserIDKey, "admin-1"))

	rr := assertQueryBudget(t, 3, h.unlinkStoryCourse, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body: %s", rr.Code, rr.Body.String())
	}
	if n := len(mockDB.Calls("DELETE FROM course_stories")); n != 0 {
		t.Errorf("owner unlink deleted a row (%d calls)", n)
	}
}

func TestUnlinkStoryCourse_NonOwner(t *testing.T) {
	h := NewHandler(slog.New(slog.DiscardHandler))

	mockDB := database.NewMockDBTX()
	mockDB.StubQuery("name: GetUser :one", [][]any{{
		"admin-1", "a@example.com", "Admin", pgtype.Bool{Bool: false, Valid: true}, pgtype.Timestamp{}, pgtype.Timestamp{},
	}}, nil)
	mockDB.StubQuery("IsUserCourseAdmin", [][]any{{true}}, nil)
	mockDB.StubQuery("GetCourseIdForStory", [][]any{{pgtype.Int4{Int32: 3, Valid: true}}}, nil)
	models.SetDB(mockDB)
	t.Cleanup(func() { models.SetDB(struct{}{}) })

	req := httptest.NewRequest(http.MethodDelete, "/api/admin/stories/7/courses/5", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "7", "courseId": "5"})
	req = req.WithContext(context.WithValue(req.Context(), auth.UserIDKey, "admin-1"))

	rr := assertQueryBudget(t, 4, h.unlinkStoryCourse, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, body: %s", rr.Code, rr.Body.String())
	}
	if n := len(mockDB.Calls("DELETE FROM course_stories")); n != 1 {
		t.Errorf("unlink deletes = %d, want 1", n)
	}
}

func TestListStoryCourses(t *testing.T) {
	h := NewHandler(slog.New(slog.DiscardHandler))

	mockDB := database.NewMockDBTX()
	mockDB.StubQuery("IsUserAdminOfLinkedStory", [][]any{{true}}, nil)
	mockDB.StubQuery("ListCoursesForStory", [][]any{
		{int32(3), "LATN-101", "Latin 101"},
		{int32(5), "LATN-102", "Latin 102"},
	}, nil)
	models.SetDB(mockDB)
	t.Cleanup(func() { models.SetDB(struct{}{}) })

	req := httptest.NewRequest(http.MethodGet, "/api/admin/stories/7/courses", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "7"})
	req = req.WithContext(context.WithValue(req.Context(), auth.UserIDKey, "admin-1"))

	rr := assertQueryBudget(t, 2, h.listStoryCourses, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, body: %s", rr.Code, rr.Body.String())
	}
	var resp struct {
		Success bool                  `json:"success"`
		Courses []models.LinkedCourse `json:"courses"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if !resp.Success || len(resp.Courses) != 2 {
		t.Fatalf("got %+v", resp)
	}
}

func TestStoryStudentsHandler_RequiresCourseID(t *testing.T) {
	h := NewHandler(slog.New(slog.DiscardHandler))
	models.SetDB(database.NewMockDBTX())
	t.Cleanup(func() { models.SetDB(struct{}{}) })

	req := httptest.NewRequest(http.MethodGet, "/api/admin/stories/7/students", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "7"})
	req = req.WithContext(context.WithValue(req.Context(), auth.UserIDKey, "admin-1"))

	rr := assertQueryBudget(t, 0, h.storyStudentsHandler, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rr.Code)
	}
}

func TestStoryStudentsHandler_LinkedCourseRoster(t *testing.T) {
	h := NewHandler(slog.New(slog.DiscardHandler))

	mockDB := database.NewMockDBTX()
	mockDB.StubQuery("name: GetUser :one", [][]any{{
		"admin-1", "a@example.com", "Admin", pgtype.Bool{Bool: false, Valid: true}, pgtype.Timestamp{}, pgtype.Timestamp{},
	}}, nil)
	mockDB.StubQuery("IsUserCourseAdmin", [][]any{{true}}, nil)
	mockDB.StubQuery("StoryLinkedToCourse", [][]any{{true}}, nil)
	mockDB.StubQuery("GetStoryPhaseTotals", [][]any{{int32(0), int32(0), int32(0), int32(0), int32(0)}}, nil)
	mockDB.StubQuery("GetStoryStudentPerformance", nil, nil)
	models.SetDB(mockDB)
	t.Cleanup(func() { models.SetDB(struct{}{}) })

	req := httptest.NewRequest(http.MethodGet, "/api/admin/stories/7/students?course_id=5&status=active", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "7"})
	req = req.WithContext(context.WithValue(req.Context(), auth.UserIDKey, "admin-1"))

	rr := assertQueryBudget(t, 6, h.storyStudentsHandler, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, body: %s", rr.Code, rr.Body.String())
	}
}

func TestStoryStudentsHandler_InvalidSection(t *testing.T) {
	h := NewHandler(slog.New(slog.DiscardHandler))
	mockDB := database.NewMockDBTX()
	mockDB.StubQuery("name: GetUser :one", [][]any{{
		"admin-1", "a@example.com", "Admin", pgtype.Bool{Bool: false, Valid: true}, pgtype.Timestamp{}, pgtype.Timestamp{},
	}}, nil)
	mockDB.StubQuery("IsUserCourseAdmin", [][]any{{true}}, nil)
	mockDB.StubQuery("StoryLinkedToCourse", [][]any{{true}}, nil)
	models.SetDB(mockDB)
	t.Cleanup(func() { models.SetDB(struct{}{}) })

	req := httptest.NewRequest(http.MethodGet, "/api/admin/stories/7/students?course_id=5&section=unassigned", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "7"})
	req = req.WithContext(context.WithValue(req.Context(), auth.UserIDKey, "admin-1"))

	rr := httptest.NewRecorder()
	h.storyStudentsHandler(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body: %s", rr.Code, rr.Body.String())
	}
}
