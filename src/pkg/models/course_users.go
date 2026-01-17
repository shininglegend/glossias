package models

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"glossias/src/pkg/generated/db"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

var ErrSomeUsersNotFound = errors.New("some users not found")

var ErrInvalidStatus = errors.New("invalid status for course")

// CourseUser represents a user enrolled in a course
type CourseUser struct {
	CourseID   int       `json:"course_id"`
	UserID     string    `json:"user_id"`
	Email      string    `json:"email"`
	Name       string    `json:"name"`
	EnrolledAt time.Time `json:"enrolled_at"`
	Status     string    `json:"status,omitempty"`
}

// UserCourse represents a course a user is enrolled in
type UserCourse struct {
	CourseID     int       `json:"course_id"`
	CourseNumber string    `json:"course_number"`
	Name         string    `json:"name"`
	Description  string    `json:"description,omitempty"`
	EnrolledAt   time.Time `json:"enrolled_at"`
	Status       string    `json:"status"`
}

// AddUserToCourseByEmail adds a user to a course by email address
func AddUserToCourseByEmail(ctx context.Context, email string, courseID int) error {
	return AddUserToCourseByEmailWithStatus(ctx, email, courseID, "active")
}

// AddUserToCourseByEmailWithStatus adds a user to a course with a specific status
func AddUserToCourseByEmailWithStatus(ctx context.Context, email string, courseID int, status string) error {
	// First get the user by email
	user, err := queries.GetUserByEmail(ctx, email)
	if err == sql.ErrNoRows || err == pgx.ErrNoRows {
		return ErrNotFound
	}
	if err != nil {
		return err
	}

	if status != "" && status != "active" && status != "past" && status != "future" {
		return ErrInvalidStatus
	}

	// Add the user to the course with status
	var statusParam string
	if status != "" {
		statusParam = status
	} else {
		statusParam = "active"
	}

	return queries.AddUserToCourse(ctx, db.AddUserToCourseParams{
		CourseID: int32(courseID),
		UserID:   user.UserID,
		Column3:  statusParam,
	})
}

// RemoveUserFromCourse removes a user from a course
func RemoveUserFromCourse(ctx context.Context, courseID int, userID string) error {
	return queries.RemoveUserFromCourse(ctx, db.RemoveUserFromCourseParams{
		CourseID: int32(courseID),
		UserID:   userID,
	})
}

// UpdateCourseUserStatus updates the status of a user's enrollment in a course
func UpdateCourseUserStatus(ctx context.Context, courseID int, userID string, status string) error {
	return queries.UpdateCourseUserStatus(ctx, db.UpdateCourseUserStatusParams{
		CourseID: int32(courseID),
		UserID:   userID,
		Status:   pgtype.Text{String: status, Valid: status != ""},
	})
}

func BulkUpdateUserStatusInCourse(ctx context.Context, courseID int, userIDs []string, status string) error {
	if status != "active" && status != "past" && status != "future" {
		return ErrInvalidStatus
	}
	return queries.BulkUpdateCourseUserStatus(ctx, db.BulkUpdateCourseUserStatusParams{
		CourseID: int32(courseID),
		Status:   pgtype.Text{String: status, Valid: true},
		Column2:  userIDs,
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
			Status:       result.Status.String,
		}
		if result.Description.Valid {
			course.Description = result.Description.String
		}
		// Default to 'active' if status is empty (backward compatibility)
		if course.Status == "" {
			course.Status = "active"
		}
		courses = append(courses, course)
	}

	return courses, nil
}

// GetCoursesForUserByStatus returns courses for a user filtered by status
func GetCoursesForUserByStatus(ctx context.Context, userID string, status string) ([]UserCourse, error) {
	results, err := queries.GetCoursesForUserByStatus(ctx, db.GetCoursesForUserByStatusParams{
		UserID: userID,
		Status: pgtype.Text{String: status, Valid: status != ""},
	})
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
			Status:       result.Status.String,
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

	users := make([]CourseUser, len(results))
	for i, result := range results {
		users[i] = CourseUser{
			CourseID:   courseID,
			UserID:     result.UserID,
			Email:      result.Email,
			Name:       result.Name,
			EnrolledAt: result.EnrolledAt.Time,
			Status: "active",
		}
		// Override status if present
		if result.Status.Valid == true {
			users[i].Status = result.Status.String
		} 
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

	// Attempt to enroll all users; the SQL query uses ON CONFLICT DO NOTHING to skip users already enrolled.
	return nil, queries.AddMultiUsersToCourse(ctx, db.AddMultiUsersToCourseParams{
		CourseID: int32(courseID),
		Column2:  userIDs,
	})
}
