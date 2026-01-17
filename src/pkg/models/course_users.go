package models

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"glossias/src/pkg/generated/db"

	"github.com/jackc/pgx/v5"
)

var ErrSomeUsersNotFound = errors.New("Some users not found")

// CourseUser represents a user enrolled in a course
type CourseUser struct {
	CourseID   int       `json:"course_id"`
	UserID     string    `json:"user_id"`
	Email      string    `json:"email"`
	Name       string    `json:"name"`
	EnrolledAt time.Time `json:"enrolled_at"`
}

// UserCourse represents a course a user is enrolled in
type UserCourse struct {
	CourseID     int       `json:"course_id"`
	CourseNumber string    `json:"course_number"`
	Name         string    `json:"name"`
	Description  string    `json:"description,omitempty"`
	EnrolledAt   time.Time `json:"enrolled_at"`
}

// AddUserToCourseByEmail adds a user to a course by email address
func AddUserToCourseByEmail(ctx context.Context, email string, courseID int) error {
	// First get the user by email
	user, err := queries.GetUserByEmail(ctx, email)
	if err == sql.ErrNoRows || err == pgx.ErrNoRows {
		return ErrNotFound
	}
	if err != nil {
		return err
	}

	// Add the user to the course
	return queries.AddUserToCourse(ctx, db.AddUserToCourseParams{
		CourseID: int32(courseID),
		UserID:   user.UserID,
	})
}

// RemoveUserFromCourse removes a user from a course
func RemoveUserFromCourse(ctx context.Context, courseID int, userID string) error {
	return queries.RemoveUserFromCourse(ctx, db.RemoveUserFromCourseParams{
		CourseID: int32(courseID),
		UserID:   userID,
	})
}

// DeleteAllUsersFromCourse removes all users from a course
func DeleteAllUsersFromCourse(ctx context.Context, courseID int) error {
	return queries.DeleteAllUsersFromCourse(ctx, int32(courseID))
}

// GetCoursesForUser returns all courses a user is enrolled in
func GetCoursesForUser(ctx context.Context, userID string) ([]UserCourse, error) {
	results, err := queries.GetCoursesForUser(ctx, userID)
	if err != nil {
		return nil, err
	}

	var courses []UserCourse
	for _, result := range results {
		course := UserCourse{
			CourseID:     int(result.CourseID),
			CourseNumber: result.CourseNumber,
			Name:         result.Name,
			EnrolledAt:   result.EnrolledAt.Time,
		}
		if result.Description.Valid {
			course.Description = result.Description.String
		}
		courses = append(courses, course)
	}

	return courses, nil
}

// GetUsersForCourse returns all users enrolled in a course
func GetUsersForCourse(ctx context.Context, courseID int) ([]CourseUser, error) {
	results, err := queries.GetUsersForCourse(ctx, int32(courseID))
	if err != nil {
		return nil, err
	}

	var users []CourseUser
	for _, result := range results {
		users = append(users, CourseUser{
			CourseID:   courseID,
			UserID:     result.UserID,
			Email:      result.Email,
			Name:       result.Name,
			EnrolledAt: result.EnrolledAt.Time,
		})
	}

	return users, nil
}

// MassImportUsersToCourse enrolls a list of users in a course
func MassImportUsersToCourse(ctx context.Context, courseID int, userEmails []string) ([]string, error) {
	// First get user IDs for the emails
	// One mass query to get all users by email
	users, err := queries.GetUsersByEmails(ctx, userEmails)
	if err != nil {
		return nil, err
	}

	if len(users) != len(userEmails) {
		// Some emails did not match any users
		var foundEmails []string
		for _, user := range users {
			foundEmails = append(foundEmails, user.Email)
		}
		var notFoundEmails []string
		emailSet := make(map[string]bool)
		for _, email := range foundEmails {
			emailSet[strings.ToLower(email)] = true
		}
		for _, email := range userEmails {
			if !emailSet[strings.ToLower(email)] {
				notFoundEmails = append(notFoundEmails, email)
			}
		}
		return notFoundEmails, ErrSomeUsersNotFound
	}

	// Get the IDs
	userIDs := make([]string, len(users))
	for i, user := range users {
		userIDs[i] = user.UserID
	}

	// Only add the new ones
	return nil, queries.AddMultiUsersToCourse(ctx, db.AddMultiUsersToCourseParams{
		CourseID: int32(courseID),
		Column2:  userIDs,
	})
}
