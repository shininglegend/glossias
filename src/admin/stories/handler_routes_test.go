package stories

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
)

// The phase-authoring routes sit under the same /{id} prefix as the existing
// story routes, and gorilla/mux resolves in registration order. These cases pin
// down that each new path reaches its own route rather than being shadowed by an
// earlier pattern.
func TestPhaseAuthoringRoutesResolve(t *testing.T) {
	router := mux.NewRouter()
	NewHandler(slog.New(slog.NewTextHandler(io.Discard, nil))).RegisterRoutes(router)

	tests := []struct {
		method string
		path   string
		want   string
	}{
		{http.MethodGet, "/stories/12/content-readiness", "/stories/{id:[0-9]+}/content-readiness"},
		{http.MethodPost, "/stories/12/phase-assets/upload", "/stories/{id:[0-9]+}/phase-assets/upload"},
		{http.MethodGet, "/stories/12/target-vocabulary", "/stories/{id:[0-9]+}/target-vocabulary"},
		{http.MethodPost, "/stories/12/target-vocabulary", "/stories/{id:[0-9]+}/target-vocabulary"},
		{http.MethodPut, "/stories/12/target-vocabulary/34", "/stories/{id:[0-9]+}/target-vocabulary/{wordId:[0-9]+}"},
		{http.MethodDelete, "/stories/12/target-vocabulary/34", "/stories/{id:[0-9]+}/target-vocabulary/{wordId:[0-9]+}"},
		{http.MethodGet, "/stories/12/produce", "/stories/{id:[0-9]+}/produce"},
		{http.MethodPut, "/stories/12/produce/explanation", "/stories/{id:[0-9]+}/produce/explanation"},
		{http.MethodPut, "/stories/12/produce/segments/2", "/stories/{id:[0-9]+}/produce/segments/{order:[0-9]+}"},
		{http.MethodGet, "/stories/12/recall", "/stories/{id:[0-9]+}/recall"},
		{http.MethodPut, "/stories/12/recall/sentences/5", "/stories/{id:[0-9]+}/recall/sentences/{order:[0-9]+}"},

		// Routes that existed before must keep resolving to themselves.
		{http.MethodGet, "/stories/12", "/stories/{id:[0-9]+}"},
		{http.MethodPost, "/stories/audio/upload", "/stories/audio/upload"},
		{http.MethodPost, "/stories/image/upload", "/stories/image/upload"},
	}

	for _, tt := range tests {
		t.Run(tt.method+" "+tt.path, func(t *testing.T) {
			var match mux.RouteMatch
			request := httptest.NewRequest(tt.method, tt.path, nil)

			if !router.Match(request, &match) {
				t.Fatalf("no route matched %s %s", tt.method, tt.path)
			}
			if match.MatchErr != nil {
				t.Fatalf("match error for %s %s: %v", tt.method, tt.path, match.MatchErr)
			}

			got, err := match.Route.GetPathTemplate()
			if err != nil {
				t.Fatalf("GetPathTemplate: %v", err)
			}
			if got != tt.want {
				t.Errorf("matched %q, want %q", got, tt.want)
			}
		})
	}
}
