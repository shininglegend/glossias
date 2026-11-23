package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"glossias/src/pkg/testutil"
)

// assertQueryBudget is testutil.AssertQueryBudget for handler funcs.
func assertQueryBudget(t *testing.T, max int, handler http.HandlerFunc, req *http.Request) *httptest.ResponseRecorder {
	t.Helper()
	return testutil.AssertQueryBudget(t, max, handler, req)
}
