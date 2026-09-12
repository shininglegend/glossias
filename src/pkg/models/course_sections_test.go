package models

import (
	"context"
	"errors"
	"testing"

	"glossias/src/pkg/database"
)

func TestAttachCourseSection_RejectsSameCourse(t *testing.T) {
	if err := AttachCourseSection(context.Background(), 4, 4); !errors.Is(err, ErrInvalidSection) {
		t.Fatalf("err = %v, want ErrInvalidSection", err)
	}
}

func TestResolveStoryRosterScope(t *testing.T) {
	mockDB := database.NewMockDBTX()
	mockDB.StubQuery("ListCourseSections", [][]any{
		{int32(2), "101-A", "Section A"},
		{int32(3), "101-B", "Section B"},
	}, nil)
	SetDB(mockDB)
	t.Cleanup(func() { SetDB(struct{}{}) })

	ctx := context.Background()

	all, err := ResolveStoryRosterScope(ctx, 1, "")
	if err != nil {
		t.Fatalf("all: %v", err)
	}
	if len(all.CourseIDs) != 3 || all.CourseIDs[0] != 1 || all.UnassignedOnly {
		t.Fatalf("all scope = %+v", all)
	}

	unassigned, err := ResolveStoryRosterScope(ctx, 1, "unassigned")
	if err != nil {
		t.Fatalf("unassigned: %v", err)
	}
	if len(unassigned.CourseIDs) != 1 || unassigned.CourseIDs[0] != 1 || !unassigned.UnassignedOnly {
		t.Fatalf("unassigned scope = %+v", unassigned)
	}

	specific, err := ResolveStoryRosterScope(ctx, 1, "2")
	if err != nil {
		t.Fatalf("specific: %v", err)
	}
	if len(specific.CourseIDs) != 1 || specific.CourseIDs[0] != 2 {
		t.Fatalf("specific scope = %+v", specific)
	}

	if _, err := ResolveStoryRosterScope(ctx, 1, "99"); !errors.Is(err, ErrInvalidSectionFilter) {
		t.Fatalf("bogus id: %v", err)
	}
	if _, err := ResolveStoryRosterScope(ctx, 1, "nope"); !errors.Is(err, ErrInvalidSectionFilter) {
		t.Fatalf("bogus text: %v", err)
	}
}

func TestResolveStoryRosterScope_LeafRejectsFilter(t *testing.T) {
	mockDB := database.NewMockDBTX()
	SetDB(mockDB)
	t.Cleanup(func() { SetDB(struct{}{}) })

	scope, err := ResolveStoryRosterScope(context.Background(), 1, "all")
	if err != nil {
		t.Fatalf("all on leaf: %v", err)
	}
	if len(scope.CourseIDs) != 1 || scope.CourseIDs[0] != 1 {
		t.Fatalf("leaf all = %+v", scope)
	}
	if _, err := ResolveStoryRosterScope(context.Background(), 1, "unassigned"); !errors.Is(err, ErrInvalidSectionFilter) {
		t.Fatalf("unassigned on leaf: %v", err)
	}
}

func TestAssignUsersToSection_Empty(t *testing.T) {
	if err := AssignUsersToSection(context.Background(), 1, 2, nil); err != nil {
		t.Fatalf("empty assign: %v", err)
	}
}
