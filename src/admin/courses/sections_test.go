package courses

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

func TestSectionRoutesResolve(t *testing.T) {
	router := mux.NewRouter()
	NewHandler(slog.New(slog.DiscardHandler)).RegisterRoutes(router.PathPrefix("/api/admin").Subrouter())

	cases := []struct {
		method, path, want string
	}{
		{http.MethodGet, "/api/admin/courses/1/sections", "/api/admin/courses/{id:[0-9]+}/sections"},
		{http.MethodPost, "/api/admin/courses/1/sections", "/api/admin/courses/{id:[0-9]+}/sections"},
		{http.MethodPost, "/api/admin/courses/1/sections/assignments", "/api/admin/courses/{id:[0-9]+}/sections/assignments"},
		{http.MethodDelete, "/api/admin/courses/1/sections/2", "/api/admin/courses/{id:[0-9]+}/sections/{sectionId:[0-9]+}"},
	}
	for _, c := range cases {
		req := httptest.NewRequest(c.method, c.path, nil)
		var match mux.RouteMatch
		if !router.Match(req, &match) || match.Route == nil {
			t.Fatalf("%s %s did not match", c.method, c.path)
		}
		got, _ := match.Route.GetPathTemplate()
		if got != c.want {
			t.Errorf("%s %s matched %q, want %q", c.method, c.path, got, c.want)
		}
	}
}

func TestListCourseSectionsHandler(t *testing.T) {
	h := NewHandler(slog.New(slog.DiscardHandler))
	mockDB := database.NewMockDBTX()
	mockDB.StubQuery("name: GetUser :one", [][]any{{
		"admin-1", "a@example.com", "Admin", pgtype.Bool{Bool: false, Valid: true}, pgtype.Timestamp{}, pgtype.Timestamp{},
	}}, nil)
	mockDB.StubQuery("IsUserCourseAdmin", [][]any{{true}}, nil)
	mockDB.StubQuery("ListCourseSections", [][]any{
		{int32(2), "101-A", "Section A"},
	}, nil)
	models.SetDB(mockDB)
	t.Cleanup(func() { models.SetDB(struct{}{}) })

	req := httptest.NewRequest(http.MethodGet, "/api/admin/courses/1/sections", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "1"})
	req = req.WithContext(context.WithValue(req.Context(), auth.UserIDKey, "admin-1"))
	rr := httptest.NewRecorder()
	h.listCourseSectionsHandler(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, body: %s", rr.Code, rr.Body.String())
	}
	var resp struct {
		Sections []models.CourseSection `json:"sections"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatal(err)
	}
	if len(resp.Sections) != 1 || resp.Sections[0].CourseNumber != "101-A" {
		t.Fatalf("got %+v", resp.Sections)
	}
}

func TestAssignSectionStudents_RequiresAdmin(t *testing.T) {
	h := NewHandler(slog.New(slog.DiscardHandler))
	mockDB := database.NewMockDBTX()
	mockDB.StubQuery("name: GetUser :one", [][]any{{
		"student-1", "s@example.com", "Student", pgtype.Bool{Bool: false, Valid: true}, pgtype.Timestamp{}, pgtype.Timestamp{},
	}}, nil)
	mockDB.StubQuery("IsUserCourseAdmin", [][]any{{false}}, nil)
	models.SetDB(mockDB)
	t.Cleanup(func() { models.SetDB(struct{}{}) })

	body, _ := json.Marshal(map[string]any{"user_ids": []string{"u1"}, "section_course_id": 2})
	req := httptest.NewRequest(http.MethodPost, "/api/admin/courses/1/sections/assignments", bytes.NewReader(body))
	req = mux.SetURLVars(req, map[string]string{"id": "1"})
	req = req.WithContext(context.WithValue(req.Context(), auth.UserIDKey, "student-1"))
	rr := httptest.NewRecorder()
	h.assignSectionStudentsHandler(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", rr.Code)
	}
}

func TestAssignSectionStudentsHandler(t *testing.T) {
	h := NewHandler(slog.New(slog.DiscardHandler))
	mockDB := database.NewMockDBTX()
	mockDB.StubQuery("name: GetUser :one", [][]any{{
		"admin-1", "a@example.com", "Admin", pgtype.Bool{Bool: true, Valid: true}, pgtype.Timestamp{}, pgtype.Timestamp{},
	}}, nil)
	mockDB.StubQuery("GetSectionParent", [][]any{{int32(1)}}, nil)
	models.SetDB(mockDB)
	t.Cleanup(func() { models.SetDB(struct{}{}) })

	body, _ := json.Marshal(map[string]any{"user_ids": []string{"u1"}, "section_course_id": 2})
	req := httptest.NewRequest(http.MethodPost, "/api/admin/courses/1/sections/assignments", bytes.NewReader(body))
	req = mux.SetURLVars(req, map[string]string{"id": "1"})
	req = req.WithContext(context.WithValue(req.Context(), auth.UserIDKey, "admin-1"))
	rr := httptest.NewRecorder()
	h.assignSectionStudentsHandler(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, body: %s", rr.Code, rr.Body.String())
	}
	if n := len(mockDB.Calls("RemoveUsersFromSiblingSections")); n != 1 {
		t.Errorf("sibling remove calls = %d", n)
	}
	if n := len(mockDB.Calls("AddMultiUsersToCourse")); n != 1 {
		t.Errorf("enroll calls = %d", n)
	}
}

func TestAttachCourseSection_RequiresSuperAdmin(t *testing.T) {
	h := NewHandler(slog.New(slog.DiscardHandler))
	mockDB := database.NewMockDBTX()
	mockDB.StubQuery("name: GetUser :one", [][]any{{
		"admin-1", "a@example.com", "Admin", pgtype.Bool{Bool: false, Valid: true}, pgtype.Timestamp{}, pgtype.Timestamp{},
	}}, nil)
	models.SetDB(mockDB)
	t.Cleanup(func() { models.SetDB(struct{}{}) })

	body, _ := json.Marshal(map[string]any{"section_course_id": 2})
	req := httptest.NewRequest(http.MethodPost, "/api/admin/courses/1/sections", bytes.NewReader(body))
	req = mux.SetURLVars(req, map[string]string{"id": "1"})
	req = req.WithContext(context.WithValue(req.Context(), auth.UserIDKey, "admin-1"))
	rr := httptest.NewRecorder()
	h.attachCourseSectionHandler(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", rr.Code)
	}
}
