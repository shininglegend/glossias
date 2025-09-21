package models

import (
	"context"
	"database/sql"
	"time"

	"glossias/src/pkg/generated/db"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

// User represents a user in the system
type User struct {
	UserID       string    `json:"user_id"`
	Email        string    `json:"email"`
	Name         string    `json:"name"`
	IsSuperAdmin bool      `json:"is_super_admin"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// CourseAdminRight represents a user's course admin assignment
type CourseAdminRight struct {
	CourseID     int32     `json:"course_id"`
	CourseNumber string    `json:"course_number"`
	CourseName   string    `json:"course_name"`
	AssignedAt   time.Time `json:"assigned_at"`
}

// UpsertUser creates or updates a user record
func UpsertUser(ctx context.Context, userID, email, name string) (*User, error) {
	result, err := queries.UpsertUser(ctx, db.UpsertUserParams{
		UserID:       userID,
		Email:        email,
		Name:         name,
		IsSuperAdmin: pgtype.Bool{Bool: false, Valid: true},
	})
	if err != nil {
		return nil, err
	}

	return &User{
		UserID:       result.UserID,
		Email:        result.Email,
		Name:         result.Name,
		IsSuperAdmin: result.IsSuperAdmin.Bool,
		CreatedAt:    result.CreatedAt.Time,
		UpdatedAt:    result.UpdatedAt.Time,
	}, nil
}

// GetUser retrieves a user by ID
func GetUser(ctx context.Context, userID string) (*User, error) {
	result, err := queries.GetUser(ctx, userID)
	if err == sql.ErrNoRows || err == pgx.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	return &User{
		UserID:       result.UserID,
		Email:        result.Email,
		Name:         result.Name,
		IsSuperAdmin: result.IsSuperAdmin.Bool,
		CreatedAt:    result.CreatedAt.Time,
		UpdatedAt:    result.UpdatedAt.Time,
	}, nil
}

// CanUserAccessCourse checks if user can access a specific course
func CanUserAccessCourse(ctx context.Context, userID string, courseID int32) bool {
	canAccess, err := queries.CanUserAccessCourse(ctx, db.CanUserAccessCourseParams{
		UserID:   userID,
		CourseID: courseID,
	})
	return err == nil && canAccess
}

// IsUserAdmin checks if user is admin of any course or super admin
func IsUserAdmin(ctx context.Context, userID string) bool {
	// Check if super admin
	user, err := queries.GetUser(ctx, userID)
	if err == nil && user.IsSuperAdmin.Bool {
		return true
	}

	// Check if course admin
	isAdmin, err := queries.IsUserAdminOfAnyCourse(ctx, userID)
	return err == nil && isAdmin
}

// IsUserCourseAdmin checks if user is admin of a specific course
func IsUserCourseAdmin(ctx context.Context, userID string, courseID int32) bool {
	isAdmin, err := queries.IsUserCourseAdmin(ctx, db.IsUserCourseAdminParams{
		CourseID: courseID,
		UserID:   userID,
	})
	return err == nil && isAdmin
}

// CanUserEditStory checks if user can edit a specific story
func CanUserEditStory(ctx context.Context, userID string, storyID int32) bool {
	if IsUserSuperAdmin(ctx, userID) {
		return true
	}
	courseID, err := queries.GetCourseIdForStory(ctx, storyID)
	if err != nil {
		return false
	}
	isAdmin, err := queries.IsUserCourseAdmin(ctx, db.IsUserCourseAdminParams{
		CourseID: courseID.Int32,
		UserID:   userID,
	})
	return err == nil && isAdmin
}

// GetUserCourseAdminRights returns all courses a user is admin of
func GetUserCourseAdminRights(ctx context.Context, userID string) ([]CourseAdminRight, error) {
	results, err := queries.GetUserCourseAdminRights(ctx, userID)
	if err != nil {
		return nil, err
	}

	var rights []CourseAdminRight
	for _, result := range results {
		rights = append(rights, CourseAdminRight{
			CourseID:     result.CourseID,
			CourseNumber: result.CourseNumber,
			CourseName:   result.CourseName,
			AssignedAt:   result.AssignedAt.Time,
		})
	}

	return rights, nil
}
