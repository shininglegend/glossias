package models

import (
	"context"
	"errors"
	"testing"
	"time"

	"glossias/src/pkg/database"

	"github.com/jackc/pgx/v5/pgtype"
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

func stubStoryAccess(mock *database.MockDBTX, canAccess, admin bool, courseIDs []int32, trialFlags [][]any) {
	mock.StubQuery("CanUserAccessStory", [][]any{{canAccess}}, nil)
	mock.StubQuery("IsUserAdminOfLinkedStory", [][]any{{admin}}, nil)
	rows := make([][]any, len(courseIDs))
	for i, id := range courseIDs {
		rows[i] = []any{id}
	}
	mock.StubQuery("ListStoryCourseIDs", rows, nil)
	if trialFlags != nil {
		mock.StubQuery("ListCoursesTrialFlags", trialFlags, nil)
	}
}

func TestCheckStoryContentAccess_TrialWithoutEnrollment(t *testing.T) {
	t.Setenv("PAYWALL_ENABLED", "true")
	mockDB := database.NewMockDBTX()
	stubStoryAccess(mockDB, true, false, []int32{2}, [][]any{{int32(2), true}})
	SetDB(mockDB)
	t.Cleanup(func() { SetDB(struct{}{}) })

	if err := CheckStoryContentAccess(context.Background(), "user-1", 10); err != nil {
		t.Fatalf("trial story should be open: %v", err)
	}
}

func TestCheckStoryContentAccess_EnrolledUnpaid(t *testing.T) {
	t.Setenv("PAYWALL_ENABLED", "true")
	mockDB := database.NewMockDBTX()
	stubStoryAccess(mockDB, true, false, []int32{3}, [][]any{{int32(3), false}})
	mockDB.StubQuery("ListActiveEntitlementsForUser", nil, nil)
	mockDB.StubQuery("IsUserEnrolledInCourse", [][]any{{true}}, nil)
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
		{int32(1), false},
		{int32(2), false},
	})
	mockDB.StubQuery("ListActiveEntitlementsForUser", [][]any{
		{int32(1), pgtype.Timestamptz{Time: time.Now().Add(time.Hour), Valid: true}},
	}, nil)
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
	stubStoryAccess(mockDB, true, false, []int32{3}, [][]any{{int32(3), false}})
	SetDB(mockDB)
	t.Cleanup(func() { SetDB(struct{}{}) })

	if err := CheckStoryContentAccess(context.Background(), "user-1", 10); err != nil {
		t.Fatalf("disabled paywall should open enrolled stories: %v", err)
	}
}

func TestGrantFromCheckoutSession_Idempotent(t *testing.T) {
	mockDB := database.NewMockDBTX()
	existing := []any{
		int64(1), "user-1", int32(3), "purchase",
		pgtype.Timestamptz{Time: time.Now(), Valid: true},
		pgtype.Timestamptz{Time: time.Now().Add(time.Hour), Valid: true},
		pgtype.Int4{},
		pgtype.Text{String: "cs_1", Valid: true},
		pgtype.Text{String: "pi_1", Valid: true},
		pgtype.Text{},
	}
	mockDB.StubQuery("GetAccessEntitlementBySessionID", [][]any{existing}, nil)
	SetDB(mockDB)
	t.Cleanup(func() { SetDB(struct{}{}) })

	if err := GrantFromCheckoutSession(context.Background(), "user-1", 3, "cs_1", "pi_1", 0, "purchase"); err != nil {
		t.Fatalf("repeat session should be a no-op: %v", err)
	}
	if n := len(mockDB.Calls("InsertAccessEntitlement")); n != 0 {
		t.Fatalf("inserted %d rows, want 0", n)
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
	stubStoryAccess(mockDB, true, false, []int32{3}, [][]any{{int32(3), false}})
	mockDB.StubQuery("ListActiveEntitlementsForUser", nil, nil)
	mockDB.StubQuery("IsUserEnrolledInCourse", [][]any{{true}}, nil)
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
