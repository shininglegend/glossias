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
	"strings"
	"testing"

	"github.com/gorilla/mux"
	"github.com/jackc/pgx/v5/pgtype"
)

// stubNavigationDB wires a mock DB with an uncoursed story 2 and the given
// page-completion row (see GetUserStoryPageCompletion column order).
func stubNavigationDB(t *testing.T, completion []any) {
	t.Helper()
	mockDB := database.NewMockDBTX()
	// "GetStory" alone would also match GetStoryTitles/GetStoryLines/...
	mockDB.StubQuery("name: GetStory :one", [][]any{{
		int32(2), int32(1), "A", pgtype.Text{}, pgtype.Timestamp{}, "author", "Author", pgtype.Int4{},
	}}, nil)
	mockDB.StubQuery("GetUserStoryPageCompletion", [][]any{completion}, nil)
	models.SetDB(mockDB)
	t.Cleanup(func() { models.SetDB(struct{}{}) })
}

func TestNavigate(t *testing.T) {
	h := NewHandler(slog.New(slog.DiscardHandler), nil)

	// Column order: identify total/correct, translate, recall total/correct,
	// produce total/submitted.
	allDone := []any{int32(4), int32(4), true, int32(5), int32(5), int32(2), int32(2)}
	freshStory := []any{int32(4), int32(0), false, int32(5), int32(0), int32(2), int32(0)}
	identifyDone := []any{int32(4), int32(4), false, int32(5), int32(0), int32(2), int32(0)}
	produceLeft := []any{int32(4), int32(4), true, int32(5), int32(5), int32(2), int32(1)}

	tests := []struct {
		name        string
		currentPage string
		completion  []any
		wantNext    string
	}{
		{"fresh story: video leads to identify", "video", freshStory, "identify"},
		{"identify done skips to translate", "video", identifyDone, "translate"},
		{"everything done goes to score", "video", allDone, "score"},
		{"unknown page restarts at video", "bogus", freshStory, "video"},
		{"legacy vocab page is not in the flow", "vocab", freshStory, "video"},
		{"produce incomplete stops there", "translate", produceLeft, "produce"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stubNavigationDB(t, tt.completion)

			body := strings.NewReader(`{"currentPage":"` + tt.currentPage + `"}`)
			req := httptest.NewRequest("POST", "/api/stories/2/next", body)
			req = mux.SetURLVars(req, map[string]string{"id": "2"})
			req = req.WithContext(context.WithValue(req.Context(), auth.UserIDKey, "user-1"))

			// Budget: 10 for the uncached GetStoryData access/content load
			// (cached in production) + exactly 1 page-completion query.
			rr := assertQueryBudget(t, 11, h.Navigate, req)
			if rr.Code != http.StatusOK {
				t.Fatalf("status = %d, body = %s", rr.Code, rr.Body.String())
			}

			var resp struct {
				types.APIResponse
				Data NavigationGuidanceResponse `json:"data"`
			}
			if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
				t.Fatalf("unmarshal: %v", err)
			}
			if resp.Data.NextPage != tt.wantNext {
				t.Errorf("nextPage = %q, want %q", resp.Data.NextPage, tt.wantNext)
			}
		})
	}
}

func TestNavigateUnauthenticated(t *testing.T) {
	h := NewHandler(slog.New(slog.DiscardHandler), nil)
	req := httptest.NewRequest("POST", "/api/stories/2/next", strings.NewReader(`{"currentPage":"video"}`))
	req = mux.SetURLVars(req, map[string]string{"id": "2"})
	rr := assertQueryBudget(t, 0, h.Navigate, req)
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", rr.Code)
	}
}
