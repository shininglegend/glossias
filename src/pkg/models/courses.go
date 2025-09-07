package models

import (
	"context"
	"database/sql"
	"time"

	"glossias/src/pkg/generated/db"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

// Course represents a course in the system
type Course struct {
	CourseID     int32     `json:"course_id"`
	CourseNumber string    `json:"course_number"`
	Name         string    `json:"name"`
	Description  string    `json:"description"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// CourseAdmin represents a course admin assignment
type CourseAdmin struct {
	CourseID   int32     `json:"course_id"`
	UserID     string    `json:"user_id"`
	Email      string    `json:"email"`
	Name       string    `json:"name"`
	AssignedAt time.Time `json:"assigned_at"`
}

// CreateCourse creates a new course
func CreateCourse(ctx context.Context, courseNumber, name, description string) (*Course, error) {
	result, err := queries.CreateCourse(ctx, db.CreateCourseParams{
		CourseNumber: courseNumber,
		Name:         name,
		Description:  pgtype.Text{String: description, Valid: description != ""},
	})
	if err != nil {
		return nil, err
	}

	return &Course{
		CourseID:     result.CourseID,
		CourseNumber: result.CourseNumber,
		Name:         result.Name,
		Description:  result.Description.String,
		CreatedAt:    result.CreatedAt.Time,
		UpdatedAt:    result.UpdatedAt.Time,
	}, nil
}

// GetCourse retrieves a course by ID
func GetCourse(ctx context.Context, courseID int32) (*Course, error) {
	result, err := queries.GetCourse(ctx, courseID)
	if err == sql.ErrNoRows || err == pgx.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	return &Course{
		CourseID:     result.CourseID,
		CourseNumber: result.CourseNumber,
		Name:         result.Name,
		Description:  result.Description.String,
		CreatedAt:    result.CreatedAt.Time,
		UpdatedAt:    result.UpdatedAt.Time,
	}, nil
}

// GetCourseByNumber retrieves a course by course number
func GetCourseByNumber(ctx context.Context, courseNumber string) (*Course, error) {
	result, err := queries.GetCourseByNumber(ctx, courseNumber)
	if err == sql.ErrNoRows || err == pgx.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	return &Course{
		CourseID:     result.CourseID,
		CourseNumber: result.CourseNumber,
		Name:         result.Name,
		Description:  result.Description.String,
		CreatedAt:    result.CreatedAt.Time,
		UpdatedAt:    result.UpdatedAt.Time,
	}, nil
}

// ListAllCourses returns all courses
func ListAllCourses(ctx context.Context) ([]Course, error) {
	results, err := queries.ListCourses(ctx)
	if err != nil {
		return nil, err
	}

	courses := make([]Course, 0, len(results))
	for _, result := range results {
		courses = append(courses, Course{
			CourseID:     result.CourseID,
			CourseNumber: result.CourseNumber,
			Name:         result.Name,
			Description:  result.Description.String,
			CreatedAt:    result.CreatedAt.Time,
			UpdatedAt:    result.UpdatedAt.Time,
		})
	}

	return courses, nil
}

// UpdateCourse updates an existing course
func UpdateCourse(ctx context.Context, courseID int32, courseNumber, name, description string) (*Course, error) {
	result, err := queries.UpdateCourse(ctx, db.UpdateCourseParams{
		CourseID:     courseID,
		CourseNumber: courseNumber,
		Name:         name,
		Description:  pgtype.Text{String: description, Valid: description != ""},
	})
	if err == sql.ErrNoRows || err == pgx.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	return &Course{
		CourseID:     result.CourseID,
		CourseNumber: result.CourseNumber,
		Name:         result.Name,
		Description:  result.Description.String,
		CreatedAt:    result.CreatedAt.Time,
		UpdatedAt:    result.UpdatedAt.Time,
	}, nil
}

// DeleteCourse deletes a course
func DeleteCourse(ctx context.Context, courseID int32) error {
	err := queries.DeleteCourse(ctx, courseID)
	if err == sql.ErrNoRows || err == pgx.ErrNoRows {
		return ErrNotFound
	}
	return err
}

// GetCourseAdmins returns all admins for a specific course
func GetCourseAdmins(ctx context.Context, courseID int32) ([]CourseAdmin, error) {
	results, err := queries.GetCourseAdmins(ctx, courseID)
	if err != nil {
		return nil, err
	}

	admins := make([]CourseAdmin, 0, len(results))
	for _, result := range results {
		admins = append(admins, CourseAdmin{
			CourseID:   result.CourseID,
			UserID:     result.UserID,
			Email:      result.Email,
			Name:       result.Name,
			AssignedAt: result.AssignedAt.Time,
		})
	}

	return admins, nil
}

// AddCourseAdmin adds a user as admin to a course
func AddCourseAdmin(ctx context.Context, courseID int32, userID string) (*CourseAdmin, error) {
	result, err := queries.AddCourseAdmin(ctx, db.AddCourseAdminParams{
		CourseID: courseID,
		UserID:   userID,
	})
	if err != nil {
		return nil, err
	}

	// Get user details for the response
	user, err := GetUser(ctx, userID)
	if err != nil {
		return nil, err
	}

	return &CourseAdmin{
		CourseID:   result.CourseID,
		UserID:     result.UserID,
		Email:      user.Email,
		Name:       user.Name,
		AssignedAt: result.AssignedAt.Time,
	}, nil
}

// RemoveCourseAdmin removes a user as admin from a course
func RemoveCourseAdmin(ctx context.Context, courseID int32, userID string) error {
	return queries.RemoveCourseAdmin(ctx, db.RemoveCourseAdminParams{
		CourseID: courseID,
		UserID:   userID,
	})
}

// IsUserSuperAdmin checks if a user is a super admin
func IsUserSuperAdmin(ctx context.Context, userID string) bool {
	user, err := queries.GetUser(ctx, userID)
	return err == nil && user.IsSuperAdmin.Bool
}

// GetCoursesForUser returns all courses a user is admin of
func GetCoursesForUser(ctx context.Context, userID string) ([]Course, error) {
	results, err := queries.GetCoursesForUser(ctx, userID)
	if err != nil {
		return nil, err
	}

	var courses []Course
	for _, result := range results {
		courses = append(courses, Course{
			CourseID:     result.CourseID,
			CourseNumber: result.CourseNumber,
			Name:         result.Name,
			Description:  result.Description.String,
			CreatedAt:    result.CreatedAt.Time,
			UpdatedAt:    result.UpdatedAt.Time,
		})
	}

	return courses, nil
}
