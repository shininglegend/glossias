//go:build integration

package integration

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"slices"
	"testing"
	"time"

	"glossias/src/apis/types"
	"glossias/src/pkg/models"
	"glossias/src/pkg/testdb"
	"glossias/src/pkg/testdb/seed"
	"glossias/src/pkg/testutil"
)

// loadSeedUsers fills seedUsers so fakeAuth accepts the dataset's users.
func loadSeedUsers(ds *seed.Dataset) {
	for _, r := range ds.Table("users").Rows {
		seedUsers[r[0].(string)] = [2]string{r[1].(string), r[2].(string)}
	}
}

// paywallOn turns the paywall on for t and checks that payments are not
// paused (a pause lifts the paywall and the test would measure nothing).
func paywallOn(t *testing.T) {
	t.Helper()
	t.Setenv("PAYWALL_ENABLED", "true")
	if models.PaymentsPaused() {
		t.Fatal("payments are paused, which lifts the paywall")
	}
}

// listStories serves GET /api/stories as user and returns the stories and
// the number of DB queries the request made.
func listStories(t *testing.T, user string) ([]types.Story, int) {
	t.Helper()
	req := httptest.NewRequest("GET", "/api/stories", nil)
	req.Header.Set(testUserHeader, user)
	rr, n := testutil.ServeCounted(router, req)
	if rr.Code != 200 {
		t.Fatalf("GET /api/stories as %s: %d %s", user, rr.Code, truncate(rr.Body.String()))
	}
	var resp struct {
		Success bool                  `json:"success"`
		Data    types.StoriesResponse `json:"data"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode stories: %v", err)
	}
	if !resp.Success {
		t.Fatalf("GET /api/stories as %s: not successful", user)
	}
	return resp.Data.Stories, n
}

// expectedLocks computes, from the seed alone, what the paywall should say
// about each story for user: open when any linked course is a trial or one
// the user holds an unexpired entitlement for; otherwise locked when the user
// is enrolled in a linked course, with those courses (ascending, as the list
// query returns linked ids) as payable.
func expectedLocks(ds *seed.Dataset, user string, now time.Time) map[int][]int {
	trial := map[int]bool{}
	for _, c := range ds.TrialCourses {
		trial[c] = true
	}
	enrolled := map[int]bool{}
	for _, r := range ds.Table("course_users").Rows {
		if r[1].(string) == user {
			enrolled[int(r[0].(int32))] = true
		}
	}
	entitled := map[int]bool{}
	for _, r := range ds.Table("access_entitlements").Rows {
		if r[0].(string) == user && r[4].(time.Time).After(now) {
			entitled[int(r[1].(int32))] = true
		}
	}
	out := map[int][]int{}
	for _, s := range ds.Stories {
		courses := slices.Sorted(slices.Values(s.Courses))
		if slices.ContainsFunc(courses, func(c int) bool { return trial[c] || entitled[c] }) {
			out[s.ID] = nil
			continue
		}
		var payable []int
		for _, c := range courses {
			if enrolled[c] {
				payable = append(payable, c)
			}
		}
		out[s.ID] = payable
	}
	return out
}

// TestStoryListLocks checks the lock outcome of every listed story for each
// student kind with the paywall on against what the seed implies.
func TestStoryListLocks(t *testing.T) {
	fx := testdb.Shared(t)
	loadSeedUsers(fx.Data)
	paywallOn(t)
	now := time.Now()

	for _, kind := range seed.Kinds {
		t.Run(string(kind), func(t *testing.T) {
			user := fx.Data.Students[kind][0]
			want := expectedLocks(fx.Data, user, now)
			models.ClearAllCache()
			stories, _ := listStories(t, user)
			if len(stories) == 0 {
				t.Fatal("no stories listed")
			}
			locked := 0
			for _, s := range stories {
				payable, ok := want[s.ID]
				if !ok {
					t.Errorf("story %d is not in the seed", s.ID)
					continue
				}
				wantLocked := len(payable) > 0
				if s.Locked != wantLocked || !slices.Equal(s.PayableCourseIDs, payable) {
					t.Errorf("story %d: locked=%v payable=%v, want locked=%v payable=%v",
						s.ID, s.Locked, s.PayableCourseIDs, wantLocked, payable)
				}
				if s.Locked {
					locked++
				}
			}
			switch kind {
			case seed.KindExpired, seed.KindUnpaid:
				if locked == 0 {
					t.Error("expected some locked stories")
				}
				for _, s := range stories {
					if s.Locked && !slices.Equal(s.PayableCourseIDs, []int{fx.Data.PaidCourses[0]}) {
						t.Errorf("story %d payable=%v, want [%d]", s.ID, s.PayableCourseIDs, fx.Data.PaidCourses[0])
					}
				}
			default:
				if locked != 0 {
					t.Errorf("%d stories locked, want none", locked)
				}
			}
		})
	}
}

// TestStoryListPaywallConstantQueries links every story to the first paid
// course and checks that the paywall-on list costs the same number of
// queries while listing many more locked stories.
func TestStoryListPaywallConstantQueries(t *testing.T) {
	fx := testdb.Fresh(t)
	loadSeedUsers(fx.Data)
	paywallOn(t)

	unpaid := fx.Data.Students[seed.KindUnpaid][0]
	models.ClearAllCache()
	before, nBefore := listStories(t, unpaid)

	if _, err := fx.Pool.Exec(context.Background(),
		`INSERT INTO course_stories (course_id, story_id)
		 SELECT $1, story_id FROM stories
		 ON CONFLICT DO NOTHING`, fx.Data.PaidCourses[0]); err != nil {
		t.Fatal(err)
	}

	models.ClearAllCache()
	after, nAfter := listStories(t, unpaid)
	lockedBefore, lockedAfter := countLocked(before), countLocked(after)
	t.Logf("unpaid: %d stories (%d locked), %d queries -> %d stories (%d locked), %d queries",
		len(before), lockedBefore, nBefore, len(after), lockedAfter, nAfter)
	if len(after) <= len(before) || lockedAfter <= lockedBefore {
		t.Fatalf("linking did not grow the locked list: %d/%d -> %d/%d", len(before), lockedBefore, len(after), lockedAfter)
	}
	if nAfter != nBefore {
		t.Fatalf("paywall-on list queries grew with story count: %d -> %d", nBefore, nAfter)
	}
}

func countLocked(stories []types.Story) int {
	n := 0
	for _, s := range stories {
		if s.Locked {
			n++
		}
	}
	return n
}
