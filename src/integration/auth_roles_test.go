//go:build integration

package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"glossias/src/pkg/models"
	"glossias/src/pkg/testdb"
	"glossias/src/pkg/testutil"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// sqlRecorder passes every statement to the pool and keeps its text, so a
// test can count calls of one named SQLC query. It has no Begin, so only
// requests that open no transaction may run against it.
type sqlRecorder struct {
	pool *pgxpool.Pool
	mu   sync.Mutex
	sqls []string
}

func (r *sqlRecorder) record(sql string) {
	r.mu.Lock()
	r.sqls = append(r.sqls, sql)
	r.mu.Unlock()
}

func (r *sqlRecorder) count(name string) int {
	r.mu.Lock()
	defer r.mu.Unlock()
	n := 0
	for _, s := range r.sqls {
		if strings.Contains(s, "-- name: "+name+" ") {
			n++
		}
	}
	return n
}

func (r *sqlRecorder) reset() {
	r.mu.Lock()
	r.sqls = nil
	r.mu.Unlock()
}

func (r *sqlRecorder) Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
	r.record(sql)
	return r.pool.Exec(ctx, sql, args...)
}

func (r *sqlRecorder) Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
	r.record(sql)
	return r.pool.Query(ctx, sql, args...)
}

func (r *sqlRecorder) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	r.record(sql)
	return r.pool.QueryRow(ctx, sql, args...)
}

func (r *sqlRecorder) CopyFrom(ctx context.Context, table pgx.Identifier, cols []string, src pgx.CopyFromSource) (int64, error) {
	r.record("COPY " + table.Sanitize())
	return r.pool.CopyFrom(ctx, table, cols, src)
}

// TestAdminRequestsReadUserOnce serves every read-only admin case and checks
// the request memo: whatever role checks a handler makes, the user row and
// the course-admin rights are each read at most once per request.
func TestAdminRequestsReadUserOnce(t *testing.T) {
	fx := testdb.Shared(t)
	fillSeedUsers(fx.Data)
	f := newFixtures(t, fx.Data)
	rec := &sqlRecorder{pool: fx.Pool}
	models.SetDB(rec)
	t.Cleanup(func() { models.SetDB(fx.Pool) })

	measured := 0
	for _, c := range f.cases() {
		if c.Mutates || c.Method != http.MethodGet || !strings.HasPrefix(c.Name, "admin/") {
			continue
		}
		t.Run(c.Name, func(t *testing.T) {
			if c.Paywall {
				t.Setenv("PAYWALL_ENABLED", "true")
			}
			models.ClearAllCache()
			rec.reset()
			serve(t, c)
			if n := rec.count("GetUser"); n > 1 {
				t.Errorf("GetUser ran %d times, want at most 1", n)
			}
			if n := rec.count("GetUserCourseAdminRights"); n > 1 {
				t.Errorf("GetUserCourseAdminRights ran %d times, want at most 1", n)
			}
		})
		measured++
	}
	if measured == 0 {
		t.Fatal("no admin read cases")
	}
}

type rosterUser struct {
	ID   string `json:"id"`
	Role string `json:"role"`
}

func getRoster(t *testing.T, as string, courseID int) ([]rosterUser, int) {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/admin/course-users/%d", courseID), nil)
	req.Header.Set(testUserHeader, as)
	rr, n := testutil.ServeCounted(router, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("roster status %d: %s", rr.Code, truncate(rr.Body.String()))
	}
	var resp struct {
		Users []rosterUser `json:"users"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatal(err)
	}
	return resp.Users, n
}

func exec(t *testing.T, fx *testdb.Fixture, sql string, args ...any) {
	t.Helper()
	if _, err := fx.Pool.Exec(context.Background(), sql, args...); err != nil {
		t.Fatalf("%s: %v", sql, err)
	}
}

// TestCourseUsersRoles checks the roster's batched roles against the
// per-user checks it replaced, with the super admin and the course admin
// enrolled in the course so every role appears.
func TestCourseUsersRoles(t *testing.T) {
	fx := testdb.Fresh(t)
	fillSeedUsers(fx.Data)
	f := newFixtures(t, fx.Data)
	for _, u := range []string{f.super, f.admin} {
		exec(t, fx, "INSERT INTO course_users (course_id, user_id) VALUES ($1, $2) ON CONFLICT DO NOTHING", f.ownCourse, u)
	}

	users, _ := getRoster(t, f.admin, f.ownCourse)
	roles := map[string]int{}
	ctx := context.Background() // no request memo: the old per-user lookups
	for _, u := range users {
		want := "student"
		if models.IsUserSuperAdmin(ctx, u.ID) {
			want = "super_admin"
		} else if models.IsUserOnlyCourseAdmin(ctx, u.ID, int32(f.ownCourse)) {
			want = "course_admin"
		}
		if u.Role != want {
			t.Errorf("%s: role %q, per-user checks say %q", u.ID, u.Role, want)
		}
		roles[u.Role]++
	}
	for _, r := range []string{"student", "course_admin", "super_admin"} {
		if roles[r] == 0 {
			t.Errorf("no %s in the roster (%v); the test does not cover that role", r, roles)
		}
	}
}

// TestCourseUsersQueryCountConstant enrolls more users and checks the roster
// makes the same number of queries.
func TestCourseUsersQueryCountConstant(t *testing.T) {
	fx := testdb.Fresh(t)
	fillSeedUsers(fx.Data)
	f := newFixtures(t, fx.Data)

	before, n1 := getRoster(t, f.admin, f.ownCourse)
	for i := range 20 {
		id := fmt.Sprintf("extra_roster_user_%02d", i)
		exec(t, fx, "INSERT INTO users (user_id, email, name) VALUES ($1, $2, $3)", id, id+"@example.test", "Extra "+id)
		exec(t, fx, "INSERT INTO course_users (course_id, user_id) VALUES ($1, $2)", f.ownCourse, id)
	}
	models.ClearAllCache()
	after, n2 := getRoster(t, f.admin, f.ownCourse)

	if len(after) != len(before)+20 {
		t.Fatalf("roster grew from %d to %d, want +20", len(before), len(after))
	}
	if n1 != n2 {
		t.Errorf("roster queries: %d with %d users, %d with %d users; want constant", n1, len(before), n2, len(after))
	}
}
