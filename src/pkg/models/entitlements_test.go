package models

import (
	"context"
	"errors"
	"slices"
	"strings"
	"testing"

	"glossias/src/pkg/database"

	"github.com/jackc/pgx/v5/pgconn"
)

func TestPaywallEnabled(t *testing.T) {
	t.Setenv("PAYWALL_ENABLED", "")
	if PaywallEnabled() {
		t.Fatal("empty env should disable paywall")
	}
	t.Setenv("PAYWALL_ENABLED", "true")
	if !PaywallEnabled() {
		t.Fatal("true should enable paywall")
	}
	t.Setenv("PAYWALL_ENABLED", "yes")
	if !PaywallEnabled() {
		t.Fatal("yes should enable paywall")
	}
	t.Setenv("PAYWALL_ENABLED", "1")
	if !PaywallEnabled() {
		t.Fatal("1 should enable paywall")
	}
	t.Setenv("PAYWALL_ENABLED", "false")
	if PaywallEnabled() {
		t.Fatal("false should disable paywall")
	}
}

func TestPaywallLiftedWhilePaymentsPaused(t *testing.T) {
	t.Setenv("PAYWALL_ENABLED", "true")
	SetPaymentsPaused(true)
	t.Cleanup(func() { SetPaymentsPaused(false) })
	if !PaymentsPaused() {
		t.Fatal("payments should report paused")
	}
	if paywallActive() {
		t.Fatal("paywall must lift while payments are paused")
	}

	// Only the visibility check runs; no lock-state queries are issued.
	mockDB := database.NewMockDBTX()
	mockDB.StubQuery("CanUserAccessStory", [][]any{{true}}, nil)
	SetDB(mockDB)
	t.Cleanup(func() { SetDB(struct{}{}) })
	if err := CheckStoryContentAccess(context.Background(), "user-1", 10); err != nil {
		t.Fatalf("paused payments should open paid content: %v", err)
	}
	if n := len(mockDB.Calls("ListCourseAccessForUser")); n != 0 {
		t.Fatalf("entitlement queries = %d, want 0", n)
	}

	stories := []Story{{Metadata: StoryMetadata{LinkedCourseIDs: []int{3}}}}
	if err := AnnotateStoryLocks(context.Background(), "user-1", stories); err != nil {
		t.Fatal(err)
	}
	if stories[0].Metadata.Locked {
		t.Fatal("stories must not be locked while payments are paused")
	}

	SetPaymentsPaused(false)
	if !paywallActive() {
		t.Fatal("paywall should return once payments resume")
	}
}

// accessRow is one ListCourseAccessForUser row:
// course_id, is_trial, enrolled, entitled.
func accessRow(courseID int32, trial, enrolled, entitled bool) []any {
	return []any{courseID, trial, enrolled, entitled}
}

func stubStoryAccess(mock *database.MockDBTX, canAccess, admin bool, courseIDs []int32, access [][]any) {
	mock.StubQuery("CanUserAccessStory", [][]any{{canAccess}}, nil)
	mock.StubQuery("IsUserAdminOfLinkedStory", [][]any{{admin}}, nil)
	rows := make([][]any, len(courseIDs))
	for i, id := range courseIDs {
		rows[i] = []any{id}
	}
	mock.StubQuery("ListStoryCourseIDs", rows, nil)
	if access != nil {
		mock.StubQuery("ListCourseAccessForUser", access, nil)
	}
}

func TestLockState(t *testing.T) {
	access := map[int]courseAccess{
		1: {trial: true},
		2: {enrolled: true, entitled: true},
		3: {enrolled: true},
		4: {},
		5: {enrolled: true},
		6: {trial: true, enrolled: true},
	}
	tests := []struct {
		name    string
		ids     []int
		locked  bool
		payable []int
	}{
		{"no courses", nil, false, nil},
		{"trial course opens", []int{3, 1}, false, nil},
		{"entitled course opens", []int{3, 2}, false, nil},
		{"enrolled unpaid locks", []int{3}, true, []int{3}},
		{"payable keeps input order", []int{5, 4, 3}, true, []int{5, 3}},
		{"not enrolled anywhere stays unlocked", []int{4}, false, nil},
		{"unknown course counts as nothing", []int{99, 3}, true, []int{3}},
		{"enrolled trial still opens", []int{6, 3}, false, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			locked, payable := lockState(tt.ids, access)
			if locked != tt.locked || !slices.Equal(payable, tt.payable) {
				t.Fatalf("lockState(%v) = %v, %v; want %v, %v", tt.ids, locked, payable, tt.locked, tt.payable)
			}
		})
	}
}

func TestAnnotateStoryLocks_OneAccessQuery(t *testing.T) {
	t.Setenv("PAYWALL_ENABLED", "true")
	SetPaymentsPaused(false)
	mockDB := database.NewMockDBTX()
	// Course 7 is administered by the user; 3 is enrolled unpaid; 4 entitled.
	mockDB.StubQuery("GetUserCourseAdminRights", [][]any{{int32(7), nil, "C7", "Course 7"}}, nil)
	mockDB.StubQuery("ListCourseAccessForUser", [][]any{
		accessRow(3, false, true, false),
		accessRow(4, false, true, true),
	}, nil)
	SetDB(mockDB)
	t.Cleanup(func() { SetDB(struct{}{}) })

	stories := []Story{
		{Metadata: StoryMetadata{LinkedCourseIDs: []int{3}}},
		{Metadata: StoryMetadata{LinkedCourseIDs: []int{3, 4}}},
		{Metadata: StoryMetadata{LinkedCourseIDs: []int{7, 3}}},
		{Metadata: StoryMetadata{}},
		{Metadata: StoryMetadata{LinkedCourseIDs: []int{4, 3}}},
	}
	if err := AnnotateStoryLocks(context.Background(), "user-1", stories); err != nil {
		t.Fatal(err)
	}
	want := []bool{true, false, false, false, false}
	for i, s := range stories {
		if s.Metadata.Locked != want[i] {
			t.Errorf("story %d locked = %v, want %v", i, s.Metadata.Locked, want[i])
		}
	}
	if !slices.Equal(stories[0].Metadata.PayableCourseIDs, []int{3}) {
		t.Errorf("story 0 payable = %v, want [3]", stories[0].Metadata.PayableCourseIDs)
	}
	calls := mockDB.Calls("ListCourseAccessForUser")
	if len(calls) != 1 {
		t.Fatalf("access queries = %d, want 1", len(calls))
	}
	// Admin-linked stories are skipped, so course 7 is never asked about.
	if ids, _ := calls[0].Args[1].([]int32); !slices.Equal(ids, []int32{3, 4}) {
		t.Fatalf("course ids = %v, want [3 4]", calls[0].Args[1])
	}
}

func TestCheckStoryContentAccess_TrialWithoutEnrollment(t *testing.T) {
	t.Setenv("PAYWALL_ENABLED", "true")
	mockDB := database.NewMockDBTX()
	stubStoryAccess(mockDB, true, false, []int32{2}, [][]any{accessRow(2, true, false, false)})
	SetDB(mockDB)
	t.Cleanup(func() { SetDB(struct{}{}) })

	if err := CheckStoryContentAccess(context.Background(), "user-1", 10); err != nil {
		t.Fatalf("trial story should be open: %v", err)
	}
}

func TestCheckStoryContentAccess_EnrolledUnpaid(t *testing.T) {
	t.Setenv("PAYWALL_ENABLED", "true")
	mockDB := database.NewMockDBTX()
	stubStoryAccess(mockDB, true, false, []int32{3}, [][]any{accessRow(3, false, true, false)})
	SetDB(mockDB)
	t.Cleanup(func() { SetDB(struct{}{}) })

	err := CheckStoryContentAccess(context.Background(), "user-1", 10)
	var pay *PaymentRequiredError
	if !errors.As(err, &pay) {
		t.Fatalf("want PaymentRequiredError, got %v", err)
	}
	if len(pay.PayableCourseIDs) != 1 || pay.PayableCourseIDs[0] != 3 {
		t.Fatalf("payable = %v, want [3]", pay.PayableCourseIDs)
	}
}

func TestCheckStoryContentAccess_PaidCourseUnlocksSharedStory(t *testing.T) {
	t.Setenv("PAYWALL_ENABLED", "true")
	mockDB := database.NewMockDBTX()
	stubStoryAccess(mockDB, true, false, []int32{1, 2}, [][]any{
		accessRow(1, false, false, true),
		accessRow(2, false, true, false),
	})
	SetDB(mockDB)
	t.Cleanup(func() { SetDB(struct{}{}) })

	if err := CheckStoryContentAccess(context.Background(), "user-1", 10); err != nil {
		t.Fatalf("entitlement on one linked course should unlock: %v", err)
	}
}

func TestCheckStoryContentAccess_AdminExempt(t *testing.T) {
	t.Setenv("PAYWALL_ENABLED", "true")
	mockDB := database.NewMockDBTX()
	stubStoryAccess(mockDB, true, true, []int32{3}, nil)
	SetDB(mockDB)
	t.Cleanup(func() { SetDB(struct{}{}) })

	if err := CheckStoryContentAccess(context.Background(), "admin-1", 10); err != nil {
		t.Fatalf("linked course admin should skip paywall: %v", err)
	}
}

func TestCheckStoryContentAccess_OrphanStory(t *testing.T) {
	t.Setenv("PAYWALL_ENABLED", "true")
	mockDB := database.NewMockDBTX()
	stubStoryAccess(mockDB, true, false, nil, nil)
	SetDB(mockDB)
	t.Cleanup(func() { SetDB(struct{}{}) })

	if err := CheckStoryContentAccess(context.Background(), "user-1", 10); err != nil {
		t.Fatalf("orphan story should be open: %v", err)
	}
}

func TestCheckStoryContentAccess_NotEnrolled(t *testing.T) {
	t.Setenv("PAYWALL_ENABLED", "true")
	mockDB := database.NewMockDBTX()
	stubStoryAccess(mockDB, false, false, []int32{3}, nil)
	SetDB(mockDB)
	t.Cleanup(func() { SetDB(struct{}{}) })

	if err := CheckStoryContentAccess(context.Background(), "user-1", 10); !errors.Is(err, ErrNotFound) {
		t.Fatalf("want ErrNotFound, got %v", err)
	}
}

func TestCheckStoryContentAccess_PaywallOff(t *testing.T) {
	t.Setenv("PAYWALL_ENABLED", "")
	mockDB := database.NewMockDBTX()
	stubStoryAccess(mockDB, true, false, []int32{3}, [][]any{accessRow(3, false, true, false)})
	SetDB(mockDB)
	t.Cleanup(func() { SetDB(struct{}{}) })

	if err := CheckStoryContentAccess(context.Background(), "user-1", 10); err != nil {
		t.Fatalf("disabled paywall should open enrolled stories: %v", err)
	}
}

func TestGrantFromCheckoutSession_Idempotent(t *testing.T) {
	// The session is already on record: ON CONFLICT DO NOTHING affects 0 rows.
	mockDB := database.NewMockDBTX()
	mockDB.StubQuery("GetLatestEntitlementExpiryForUserCourse", nil, nil)
	mockDB.StubExecRows("InsertAccessEntitlement", 0)
	SetDB(mockDB)
	t.Cleanup(func() { SetDB(struct{}{}) })

	if err := GrantFromCheckoutSession(context.Background(), "user-1", 3, "cs_1", "pi_1", 0, "purchase"); err != nil {
		t.Fatalf("repeat session should be a no-op: %v", err)
	}
	calls := mockDB.Calls("InsertAccessEntitlement")
	if len(calls) != 1 {
		t.Fatalf("insert calls = %d, want 1", len(calls))
	}
	if !strings.Contains(calls[0].SQL, "ON CONFLICT (stripe_checkout_session_id) DO NOTHING") {
		t.Fatalf("insert must be idempotent on session id: %s", calls[0].SQL)
	}
}

func TestGetStoryData_PaymentRequiredWithCache(t *testing.T) {
	t.Setenv("PAYWALL_ENABLED", "true")
	if err := SetCache(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		cacheInstance = nil
		keyBuilder = nil
	})
	mockDB := database.NewMockDBTX()
	stubStoryAccess(mockDB, true, false, []int32{3}, [][]any{accessRow(3, false, true, false)})
	SetDB(mockDB)
	t.Cleanup(func() { SetDB(struct{}{}) })

	_, err := GetStoryData(context.Background(), 10, "user-1")
	var pay *PaymentRequiredError
	if !errors.As(err, &pay) {
		t.Fatalf("want PaymentRequiredError, got %v", err)
	}
	if len(pay.PayableCourseIDs) != 1 || pay.PayableCourseIDs[0] != 3 {
		t.Fatalf("payable = %v, want [3]", pay.PayableCourseIDs)
	}
}

func TestGrantCourseAccess_UnknownUserIsTyped(t *testing.T) {
	mockDB := database.NewMockDBTX()
	mockDB.StubQuery("GetLatestEntitlementExpiryForUserCourse", nil, nil)
	mockDB.StubExec("InsertAccessEntitlement", &pgconn.PgError{
		Code: "23503", ConstraintName: "access_entitlements_user_id_fkey",
	})
	SetDB(mockDB)
	t.Cleanup(func() { SetDB(struct{}{}) })

	_, err := GrantCourseAccess(context.Background(), GrantAccessParams{UserID: "ghost", CourseID: 1})
	if !errors.Is(err, ErrUnknownUser) {
		t.Fatalf("err = %v, want ErrUnknownUser", err)
	}

	// Any other constraint, or any other error, passes through untyped.
	mockDB.StubExec("InsertAccessEntitlement", &pgconn.PgError{
		Code: "23503", ConstraintName: "access_entitlements_course_id_fkey",
	})
	_, err = GrantCourseAccess(context.Background(), GrantAccessParams{UserID: "u", CourseID: 99})
	if err == nil || errors.Is(err, ErrUnknownUser) {
		t.Fatalf("err = %v, want a plain error", err)
	}
}
