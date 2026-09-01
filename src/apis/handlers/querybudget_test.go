package handlers

import (
	"glossias/src/pkg/database"
	"net/http"
	"net/http/httptest"
	"testing"
)

// assertQueryBudget serves req through handler and fails the test if the
// handler made more than max DB calls. It returns the recorder so callers can
// go on to assert status and body as usual.
//
// Use it in every handler test's success case. If a budget has to go up,
// first ask whether a model function is being called in a loop; the fix is
// normally a batch query (WHERE id = ANY($1)), not a bigger number.
func assertQueryBudget(t *testing.T, max int, handler http.HandlerFunc, req *http.Request) *httptest.ResponseRecorder {
	t.Helper()

	ctx, _ := database.WithQueryCounter(req.Context())
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler(rr, req)

	if got := database.QueryCount(ctx); got > max {
		t.Errorf("%s %s made %d DB queries, budget %d (possible N+1)",
			req.Method, req.URL.Path, got, max)
	}
	return rr
}
