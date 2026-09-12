package models

import (
	"context"
	"errors"

	"glossias/src/pkg/generated/db"
)

var ErrUnlinkOwner = errors.New("cannot unlink the owner course")

type LinkedCourse struct {
	CourseID     int    `json:"course_id"`
	CourseNumber string `json:"course_number"`
	Name         string `json:"name"`
}

func CanUserAccessStory(ctx context.Context, userID string, storyID int32) bool {
	ok, err := queries.CanUserAccessStory(ctx, db.CanUserAccessStoryParams{
		UserID:  userID,
		StoryID: storyID,
	})
	return err == nil && ok
}

func CanUserAdminLinkedStory(ctx context.Context, userID string, storyID int32) bool {
	ok, err := queries.IsUserAdminOfLinkedStory(ctx, db.IsUserAdminOfLinkedStoryParams{
		UserID:  userID,
		StoryID: storyID,
	})
	return err == nil && ok
}

func ListStoryCourses(ctx context.Context, storyID int32) ([]LinkedCourse, error) {
	rows, err := queries.ListCoursesForStory(ctx, storyID)
	if err != nil {
		return nil, err
	}
	out := make([]LinkedCourse, len(rows))
	for i, row := range rows {
		out[i] = LinkedCourse{
			CourseID:     int(row.CourseID),
			CourseNumber: row.CourseNumber,
			Name:         row.Name,
		}
	}
	return out, nil
}

func ListStoryCourseIDs(ctx context.Context, storyID int32) ([]int, error) {
	ids, err := queries.ListStoryCourseIDs(ctx, storyID)
	if err != nil {
		return nil, err
	}
	return int32sToInts(ids), nil
}

func LinkStoryCourse(ctx context.Context, storyID, courseID int32) error {
	return queries.LinkStoryToCourse(ctx, db.LinkStoryToCourseParams{
		CourseID: courseID,
		StoryID:  storyID,
	})
}

func UnlinkStoryCourse(ctx context.Context, storyID, courseID int32) error {
	owner, err := queries.GetCourseIdForStory(ctx, storyID)
	if err != nil {
		return err
	}
	if owner.Valid && owner.Int32 == courseID {
		return ErrUnlinkOwner
	}
	return queries.UnlinkStoryFromCourse(ctx, db.UnlinkStoryFromCourseParams{
		CourseID: courseID,
		StoryID:  storyID,
	})
}

func StoryLinkedToCourse(ctx context.Context, storyID, courseID int32) (bool, error) {
	return queries.StoryLinkedToCourse(ctx, db.StoryLinkedToCourseParams{
		CourseID: courseID,
		StoryID:  storyID,
	})
}

func ensureOwnerLinked(ctx context.Context, storyID int32, courseID *int) error {
	if courseID == nil {
		return nil
	}
	return queries.LinkStoryToCourse(ctx, db.LinkStoryToCourseParams{
		CourseID: int32(*courseID),
		StoryID:  storyID,
	})
}

func int32sToInts(ids []int32) []int {
	out := make([]int, len(ids))
	for i, id := range ids {
		out[i] = int(id)
	}
	return out
}
