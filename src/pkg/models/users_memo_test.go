package models

import (
	"context"
	"errors"
	"testing"

	"glossias/src/pkg/database"

	"github.com/jackc/pgx/v5/pgtype"
)

func userRow(id string, super bool) []any {
	return []any{id, id + "@example.com", "Name " + id, pgtype.Bool{Bool: super, Valid: true}, pgtype.Timestamp{}, pgtype.Timestamp{}}
}

func rightsRows(courseIDs ...int32) [][]any {
	rows := make([][]any, len(courseIDs))
	for i, id := range courseIDs {
		rows[i] = []any{id, pgtype.Timestamp{}, "C", "Course"}
	}
	return rows
}

func memoMock(t *testing.T) *database.MockDBTX {
	t.Helper()
	mockDB := database.NewMockDBTX()
	SetDB(mockDB)
	t.Cleanup(func() { SetDB(struct{}{}) })
	return mockDB
}

const (
	getUserSQL = "name: GetUser :one"
	rightsSQL  = "name: GetUserCourseAdminRights"
)

func TestRequestMemoReadsUserAndRightsOnce(t *testing.T) {
	mockDB := memoMock(t)
	mockDB.StubQuery(getUserSQL, [][]any{userRow("u1", false)}, nil)
	mockDB.StubQuery(rightsSQL, rightsRows(3, 5), nil)
	ctx := WithRequestMemo(context.Background())

	if !IsUserAnyAdmin(ctx, "u1") {
		t.Error("IsUserAnyAdmin = false, want true")
	}
	if !IsUserCourseOrSuperAdmin(ctx, "u1", 3) {
		t.Error("IsUserCourseOrSuperAdmin(3) = false, want true")
	}
	if IsUserCourseOrSuperAdmin(ctx, "u1", 4) {
		t.Error("IsUserCourseOrSuperAdmin(4) = true, want false")
	}
	if !IsUserOnlyCourseAdmin(ctx, "u1", 5) {
		t.Error("IsUserOnlyCourseAdmin(5) = false, want true")
	}
	if IsUserSuperAdmin(ctx, "u1") {
		t.Error("IsUserSuperAdmin = true, want false")
	}

	if n := len(mockDB.Calls(getUserSQL)); n != 1 {
		t.Errorf("GetUser queries = %d, want 1", n)
	}
	if n := len(mockDB.Calls(rightsSQL)); n != 1 {
		t.Errorf("rights queries = %d, want 1", n)
	}
}

func TestNoMemoQueriesEveryTime(t *testing.T) {
	mockDB := memoMock(t)
	mockDB.StubQuery(getUserSQL, [][]any{userRow("u1", false)}, nil)
	ctx := context.Background()

	GetUser(ctx, "u1")
	GetUser(ctx, "u1")
	if n := len(mockDB.Calls(getUserSQL)); n != 2 {
		t.Errorf("GetUser queries without a memo = %d, want 2", n)
	}
}

func TestSuperAdminSkipsRights(t *testing.T) {
	mockDB := memoMock(t)
	mockDB.StubQuery(getUserSQL, [][]any{userRow("s1", true)}, nil)
	ctx := WithRequestMemo(context.Background())

	if !IsUserAnyAdmin(ctx, "s1") || !IsUserCourseOrSuperAdmin(ctx, "s1", 99) {
		t.Fatal("super admin not recognised")
	}
	if IsUserOnlyCourseAdmin(ctx, "s1", 99) {
		t.Error("IsUserOnlyCourseAdmin must not apply the super admin override")
	}
	if n := len(mockDB.Calls(getUserSQL)); n != 1 {
		t.Errorf("GetUser queries = %d, want 1", n)
	}
}

func TestMemoCachesNotFoundButNotErrors(t *testing.T) {
	mockDB := memoMock(t)
	ctx := WithRequestMemo(context.Background())

	// Unstubbed: the mock answers ErrNoRows.
	for range 2 {
		if _, err := GetUser(ctx, "ghost"); !errors.Is(err, ErrNotFound) {
			t.Fatalf("err = %v, want ErrNotFound", err)
		}
	}
	if n := len(mockDB.Calls(getUserSQL)); n != 1 {
		t.Errorf("GetUser queries for a missing user = %d, want 1", n)
	}

	mockDB2 := memoMock(t)
	mockDB2.StubQuery(getUserSQL, nil, errors.New("connection reset"))
	mockDB2.StubQuery(rightsSQL, nil, errors.New("connection reset"))
	ctx = WithRequestMemo(context.Background())
	for range 2 {
		if _, err := GetUser(ctx, "u1"); err == nil {
			t.Fatal("want an error")
		}
		if _, err := GetUserCourseAdminRights(ctx, "u1"); err == nil {
			t.Fatal("want an error")
		}
	}
	if n := len(mockDB2.Calls(getUserSQL)); n != 2 {
		t.Errorf("GetUser queries after errors = %d, want 2 (errors are not memoised)", n)
	}
	if n := len(mockDB2.Calls(rightsSQL)); n != 2 {
		t.Errorf("rights queries after errors = %d, want 2 (errors are not memoised)", n)
	}
}

func TestUpsertUserSeedsMemo(t *testing.T) {
	mockDB := memoMock(t)
	mockDB.StubQuery("name: UpsertUser :one", [][]any{userRow("u1", true)}, nil)
	ctx := WithRequestMemo(context.Background())

	if _, err := UpsertUser(ctx, "u1", "u1@example.com", "Name u1"); err != nil {
		t.Fatal(err)
	}
	if !IsUserSuperAdmin(ctx, "u1") {
		t.Error("seeded row lost is_super_admin")
	}
	if n := len(mockDB.Calls(getUserSQL)); n != 0 {
		t.Errorf("GetUser queries after an upsert = %d, want 0", n)
	}
}

func TestCourseAdminWritesForgetRights(t *testing.T) {
	mockDB := memoMock(t)
	mockDB.StubQuery(getUserSQL, [][]any{userRow("u1", false)}, nil)
	mockDB.StubQuery(rightsSQL, rightsRows(3), nil)
	mockDB.StubQuery("name: AddCourseAdmin", [][]any{{int32(4), "u1", pgtype.Timestamp{}}}, nil)
	ctx := WithRequestMemo(context.Background())

	IsUserOnlyCourseAdmin(ctx, "u1", 3)
	if _, err := AddCourseAdmin(ctx, 4, "u1"); err != nil {
		t.Fatal(err)
	}
	IsUserOnlyCourseAdmin(ctx, "u1", 4)
	if err := RemoveCourseAdmin(ctx, 4, "u1"); err != nil {
		t.Fatal(err)
	}
	IsUserOnlyCourseAdmin(ctx, "u1", 4)
	if err := DeleteCourse(ctx, 3); err != nil {
		t.Fatal(err)
	}
	IsUserOnlyCourseAdmin(ctx, "u1", 3)

	// One read at the start and one after each of the three writes.
	if n := len(mockDB.Calls(rightsSQL)); n != 4 {
		t.Errorf("rights queries = %d, want 4", n)
	}
}

func TestCanUserEditStoryUsesRights(t *testing.T) {
	mockDB := memoMock(t)
	mockDB.StubQuery(getUserSQL, [][]any{userRow("u1", false)}, nil)
	mockDB.StubQuery(rightsSQL, rightsRows(3), nil)
	mockDB.StubQuery("GetCourseIdForStory", [][]any{{pgtype.Int4{Int32: 3, Valid: true}}}, nil)
	ctx := WithRequestMemo(context.Background())

	if !CanUserEditStory(ctx, "u1", 7) {
		t.Error("admin of the story's course cannot edit it")
	}
	if !IsUserCourseOrSuperAdmin(ctx, "u1", 3) {
		t.Error("course admin check failed")
	}
	if n := len(mockDB.Calls(rightsSQL)); n != 1 {
		t.Errorf("rights queries = %d, want 1", n)
	}
}
