//go:build integration

package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"glossias/src/pkg/models"
	"glossias/src/pkg/testdb"
)

// TestSaveStoryConflict covers the Raw JSON editor's full-story PUT: on a
// story that already has lines it answers 409 and writes nothing (the
// metadata update in the same transaction is rolled back too); on a story
// without lines it saves.
func TestSaveStoryConflict(t *testing.T) {
	fx := testdb.Fresh(t)
	fillSeedUsers(fx.Data)
	f := newFixtures(t, fx.Data)
	ctx := context.Background()
	path := fmt.Sprintf("/api/admin/stories/%d", f.story.ID)

	scalar := func(sql string) int {
		t.Helper()
		var n int
		if err := fx.Pool.QueryRow(ctx, sql, f.story.ID).Scan(&n); err != nil {
			t.Fatalf("%s: %v", sql, err)
		}
		return n
	}
	put := func(body any) *httptest.ResponseRecorder {
		t.Helper()
		raw, err := json.Marshal(body)
		if err != nil {
			t.Fatal(err)
		}
		req := httptest.NewRequest(http.MethodPut, path, bytes.NewReader(raw))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set(testUserHeader, f.admin)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)
		return rr
	}
	const linesSQL = "SELECT count(*) FROM story_lines WHERE story_id = $1"
	const weekSQL = "SELECT week_number FROM stories WHERE story_id = $1"

	story := getJSON(t, path, f.admin, "story").(map[string]any)
	meta := story["metadata"].(map[string]any)
	lines, week := scalar(linesSQL), scalar(weekSQL)
	if lines == 0 {
		t.Fatal("fixture story has no lines")
	}
	meta["weekNumber"] = week + 1

	t.Run("story with lines", func(t *testing.T) {
		rr := put(story)
		if rr.Code != http.StatusConflict {
			t.Fatalf("status %d, want 409: %s", rr.Code, truncate(rr.Body.String()))
		}
		if !bytes.Contains(rr.Body.Bytes(), []byte(models.ErrStoryHasLines.Error())) {
			t.Errorf("body %q does not explain the conflict", rr.Body.String())
		}
		if got := scalar(linesSQL); got != lines {
			t.Errorf("lines %d after a refused save, want %d", got, lines)
		}
		if got := scalar(weekSQL); got != week {
			t.Errorf("week_number %d after a refused save, want %d (not rolled back)", got, week)
		}
	})

	t.Run("story without lines", func(t *testing.T) {
		if _, err := fx.Pool.Exec(ctx, "DELETE FROM story_lines WHERE story_id = $1", f.story.ID); err != nil {
			t.Fatal(err)
		}
		models.ClearAllCache()
		rr := put(story)
		if rr.Code != http.StatusOK {
			t.Fatalf("status %d, want 200: %s", rr.Code, truncate(rr.Body.String()))
		}
		if got := scalar(linesSQL); got != lines {
			t.Errorf("lines %d after save, want %d", got, lines)
		}
		if got := scalar(weekSQL); got != week+1 {
			t.Errorf("week_number %d after save, want %d", got, week+1)
		}
	})
}
