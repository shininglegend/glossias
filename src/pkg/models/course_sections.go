package models

import (
	"context"
	"database/sql"
	"errors"
	"strconv"
	"strings"

	"glossias/src/pkg/generated/db"

	"github.com/jackc/pgx/v5"
)

var (
	ErrInvalidSection       = errors.New("invalid section")
	ErrSectionCycle         = errors.New("a section cannot itself have sections")
	ErrAlreadySection       = errors.New("course is already a section")
	ErrSectionNotChild      = errors.New("course is not a section of this parent")
	ErrInvalidSectionFilter = errors.New("invalid section filter")
)

type CourseSection struct {
	CourseID     int32  `json:"course_id"`
	CourseNumber string `json:"course_number"`
	Name         string `json:"name"`
}

type StoryRosterScope struct {
	CourseIDs      []int32
	UnassignedOnly bool
	SectionIDs     []int32
}

func ListCourseSections(ctx context.Context, parentID int32) ([]CourseSection, error) {
	rows, err := queries.ListCourseSections(ctx, parentID)
	if err != nil {
		return nil, err
	}
	out := make([]CourseSection, len(rows))
	for i, row := range rows {
		out[i] = CourseSection{
			CourseID:     row.CourseID,
			CourseNumber: row.CourseNumber,
			Name:         row.Name,
		}
	}
	return out, nil
}

func GetSectionParent(ctx context.Context, sectionID int32) (int32, error) {
	parent, err := queries.GetSectionParent(ctx, sectionID)
	if err != nil {
		if err == sql.ErrNoRows || err == pgx.ErrNoRows {
			return 0, ErrNotFound
		}
		return 0, err
	}
	return parent, nil
}

func AttachCourseSection(ctx context.Context, parentID, sectionID int32) error {
	if parentID == 0 || sectionID == 0 || parentID == sectionID {
		return ErrInvalidSection
	}
	if _, err := GetCourse(ctx, parentID); err != nil {
		return err
	}
	if _, err := GetCourse(ctx, sectionID); err != nil {
		return err
	}

	parentIsSection, err := queries.CourseIsSection(ctx, parentID)
	if err != nil {
		return err
	}
	if parentIsSection {
		return ErrSectionCycle
	}
	sectionHasChildren, err := queries.CourseHasSections(ctx, sectionID)
	if err != nil {
		return err
	}
	if sectionHasChildren {
		return ErrSectionCycle
	}
	already, err := queries.CourseIsSection(ctx, sectionID)
	if err != nil {
		return err
	}
	if already {
		return ErrAlreadySection
	}

	if err := queries.AttachCourseSection(ctx, db.AttachCourseSectionParams{
		ParentCourseID:  parentID,
		SectionCourseID: sectionID,
	}); err != nil {
		return err
	}
	return queries.CopyCourseStoryLinks(ctx, db.CopyCourseStoryLinksParams{
		ToCourseID:   sectionID,
		FromCourseID: parentID,
	})
}

func DetachCourseSection(ctx context.Context, parentID, sectionID int32) error {
	parent, err := GetSectionParent(ctx, sectionID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return ErrSectionNotChild
		}
		return err
	}
	if parent != parentID {
		return ErrSectionNotChild
	}
	return queries.DetachCourseSection(ctx, db.DetachCourseSectionParams{
		ParentCourseID:  parentID,
		SectionCourseID: sectionID,
	})
}

// AssignUsersToSection enrolls users in sectionID (and drops sibling-section
// enrollments). sectionID 0 removes them from every section of the parent.
func AssignUsersToSection(ctx context.Context, parentID, sectionID int32, userIDs []string) error {
	if len(userIDs) == 0 {
		return nil
	}
	if sectionID != 0 {
		parent, err := GetSectionParent(ctx, sectionID)
		if err != nil {
			if errors.Is(err, ErrNotFound) {
				return ErrSectionNotChild
			}
			return err
		}
		if parent != parentID {
			return ErrSectionNotChild
		}
	} else {
		sections, err := ListCourseSections(ctx, parentID)
		if err != nil {
			return err
		}
		if len(sections) == 0 {
			return ErrInvalidSection
		}
	}

	if err := queries.RemoveUsersFromSiblingSections(ctx, db.RemoveUsersFromSiblingSectionsParams{
		UserIds:       userIDs,
		ParentID:      parentID,
		KeepSectionID: sectionID,
	}); err != nil {
		return err
	}
	if sectionID == 0 {
		return nil
	}
	return queries.AddMultiUsersToCourse(ctx, db.AddMultiUsersToCourseParams{
		CourseID: sectionID,
		Column2:  userIDs,
	})
}

func ListUserIDsForCourses(ctx context.Context, courseIDs []int32) ([]string, error) {
	if len(courseIDs) == 0 {
		return nil, nil
	}
	return queries.ListUserIDsForCourses(ctx, courseIDs)
}

func ListUserSectionsForParent(ctx context.Context, parentID int32) (map[string][]CourseSection, error) {
	rows, err := queries.ListUserSectionsForParent(ctx, parentID)
	if err != nil {
		return nil, err
	}
	out := make(map[string][]CourseSection)
	for _, row := range rows {
		out[row.UserID] = append(out[row.UserID], CourseSection{
			CourseID:     row.CourseID,
			CourseNumber: row.CourseNumber,
			Name:         row.Name,
		})
	}
	return out, nil
}

func ResolveStoryRosterScope(ctx context.Context, viewedCourseID int32, sectionParam string) (StoryRosterScope, error) {
	sections, err := ListCourseSections(ctx, viewedCourseID)
	if err != nil {
		return StoryRosterScope{}, err
	}
	sectionIDs := make([]int32, len(sections))
	for i, s := range sections {
		sectionIDs[i] = s.CourseID
	}

	param := strings.TrimSpace(strings.ToLower(sectionParam))
	if param == "" || param == "all" {
		ids := make([]int32, 0, 1+len(sectionIDs))
		ids = append(ids, viewedCourseID)
		ids = append(ids, sectionIDs...)
		return StoryRosterScope{CourseIDs: ids, SectionIDs: sectionIDs}, nil
	}
	if len(sections) == 0 {
		return StoryRosterScope{}, ErrInvalidSectionFilter
	}
	if param == "unassigned" {
		return StoryRosterScope{
			CourseIDs:      []int32{viewedCourseID},
			UnassignedOnly: true,
			SectionIDs:     sectionIDs,
		}, nil
	}
	sid, err := strconv.ParseInt(param, 10, 32)
	if err != nil {
		return StoryRosterScope{}, ErrInvalidSectionFilter
	}
	for _, s := range sections {
		if s.CourseID == int32(sid) {
			return StoryRosterScope{CourseIDs: []int32{s.CourseID}, SectionIDs: sectionIDs}, nil
		}
	}
	return StoryRosterScope{}, ErrInvalidSectionFilter
}

func parentIDsBySection(ctx context.Context) (map[int32]int32, error) {
	rows, err := queries.ListAllCourseSections(ctx)
	if err != nil {
		return nil, err
	}
	out := make(map[int32]int32, len(rows))
	for _, row := range rows {
		out[row.SectionCourseID] = row.ParentCourseID
	}
	return out, nil
}

func applyParentIDs(courses []Course, parents map[int32]int32) {
	for i := range courses {
		if parent, ok := parents[courses[i].CourseID]; ok {
			p := parent
			courses[i].ParentCourseID = &p
		}
	}
}
