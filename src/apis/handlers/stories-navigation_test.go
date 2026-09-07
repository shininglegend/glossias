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
	// produce total/submitted, completed attempts.
	allDone := []any{int32(4), int32(4), true, int32(5), int32(5), int32(2), int32(2), int32(0)}
	freshStory := []any{int32(4), int32(0), false, int32(5), int32(0), int32(2), int32(0), int32(1)}
	identifyDone := []any{int32(4), int32(4), false, int32(5), int32(0), int32(2), int32(0), int32(0)}
	identifyInProgress := []any{int32(4), int32(2), false, int32(5), int32(0), int32(2), int32(0), int32(0)}
	produceLeft := []any{int32(4), int32(4), true, int32(5), int32(5), int32(2), int32(1), int32(0)}

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
		{"identify continue always goes to translate", "identify", identifyDone, "translate"},
		{"identify continue does not skip a completed translate", "identify", allDone, "translate"},
		{"translate continue always goes to produce", "translate", allDone, "produce"},
		{"list + fresh opens watch", "list", freshStory, "video"},
		{"list + identify in progress resumes there", "list", identifyInProgress, "identify"},
		{"list + identify done skips to translate", "list", identifyDone, "translate"},
		{"list + all done goes to score", "list", allDone, "score"},
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
			wantName := map[string]string{
				"video": "Watch", "identify": "Identify", "translate": "Translate",
				"produce": "Produce", "recall": "Recall", "score": "Score",
			}[tt.wantNext]
			if resp.Data.DisplayName != wantName {
				t.Errorf("displayName = %q, want %q", resp.Data.DisplayName, wantName)
			}
			// An archived run leaves the live rows fresh; the count is what
			// lets the Video page still offer the Score page.
			if want := int(tt.completion[7].(int32)); resp.Data.CompletedAttempts != want {
				t.Errorf("completedAttempts = %d, want %d", resp.Data.CompletedAttempts, want)
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
