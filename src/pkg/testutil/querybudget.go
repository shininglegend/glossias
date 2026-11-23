// Package testutil holds helpers shared by handler tests in several packages.
// Only test code imports it.
package testutil

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"glossias/src/pkg/database"

	"github.com/jackc/pgx/v5/pgtype"
)

// ServeCounted serves req through h with a DB query counter on its context and
// returns the recorder and the number of DB calls the request made.
func ServeCounted(h http.Handler, req *http.Request) (*httptest.ResponseRecorder, int) {
	ctx, _ := database.WithQueryCounter(req.Context())
	req = req.WithContext(ctx)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	return rr, database.QueryCount(ctx)
}

// CourseAdminRightsRows builds MockDBTX rows for GetUserCourseAdminRights
// that make the user an admin of each course in courseIDs. Stub them with
// mockDB.StubQuery("GetUserCourseAdminRights", rows, nil); no rows means an
// admin of no course.
func CourseAdminRightsRows(courseIDs ...int32) [][]any {
	rows := make([][]any, len(courseIDs))
	for i, id := range courseIDs {
		rows[i] = []any{id, pgtype.Timestamp{}, fmt.Sprintf("C-%d", id), fmt.Sprintf("Course %d", id)}
	}
	return rows
}

// AssertQueryBudget serves req through h and fails the test if the request
// made more than max DB calls. It returns the recorder so callers can go on to
// assert status and body as usual.
//
// Use it in every handler test's success case. If a budget has to go up,
// first ask whether a model function is being called in a loop; the fix is
// normally a batch query (WHERE id = ANY($1)), not a bigger number.
func AssertQueryBudget(t testing.TB, max int, h http.Handler, req *http.Request) *httptest.ResponseRecorder {
	t.Helper()
	rr, got := ServeCounted(h, req)
	if got > max {
		t.Errorf("%s %s made %d DB queries, budget %d (possible N+1)",
			req.Method, req.URL.Path, got, max)
	}
	return rr
}
