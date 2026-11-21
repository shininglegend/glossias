package models

import (
	"context"
	"database/sql"
	"slices"
	"strconv"
	"strings"
	"sync"
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

// requestMemo holds, for one HTTP request, the users and course-admin rights
// already read, so the several role checks a request makes (auth, the
// handler's permission check, the paywall) cost one query each instead of one
// per check. It lives only as long as the request's context.
type requestMemo struct {
	mu sync.Mutex
	// users maps a user id to the row read, or to nil when the user does not
	// exist (GetUser returned ErrNotFound).
	users  map[string]*User
	rights map[string][]CourseAdminRight
}

type requestMemoKey struct{}

// WithRequestMemo returns ctx carrying a fresh request memo (or ctx itself if
// it already has one). The router installs it per request. Without a memo,
// GetUser and GetUserCourseAdminRights query every time, as before.
//
// Only successes and ErrNotFound are remembered, never other errors. Model
// functions that change a user or their course-admin rights forget the
// entry. A goroutine handed the request context shares the memo, so work
// that outlives the request should start from a context without it.
func WithRequestMemo(ctx context.Context) context.Context {
	if memoFrom(ctx) != nil {
		return ctx
	}
	return context.WithValue(ctx, requestMemoKey{}, &requestMemo{
		users:  map[string]*User{},
		rights: map[string][]CourseAdminRight{},
	})
}

func memoFrom(ctx context.Context) *requestMemo {
	m, _ := ctx.Value(requestMemoKey{}).(*requestMemo)
	return m
}

// memoUser returns the remembered user (nil, ErrNotFound for a remembered
// miss) and whether there was an entry.
func memoUser(ctx context.Context, userID string) (*User, bool) {
	m := memoFrom(ctx)
	if m == nil {
		return nil, false
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	u, ok := m.users[userID]
	if !ok || u == nil {
		return nil, ok
	}
	c := *u
	return &c, true
}

// rememberUser records u (nil for a user that does not exist).
func rememberUser(ctx context.Context, userID string, u *User) {
	m := memoFrom(ctx)
	if m == nil {
		return
	}
	var stored *User
	if u != nil {
		c := *u
		stored = &c
	}
	m.mu.Lock()
	m.users[userID] = stored
	m.mu.Unlock()
}

func memoRights(ctx context.Context, userID string) ([]CourseAdminRight, bool) {
	m := memoFrom(ctx)
	if m == nil {
		return nil, false
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	r, ok := m.rights[userID]
	return slices.Clone(r), ok
}

func rememberRights(ctx context.Context, userID string, rights []CourseAdminRight) {
	m := memoFrom(ctx)
	if m == nil {
		return
	}
	m.mu.Lock()
	m.rights[userID] = slices.Clone(rights)
	m.mu.Unlock()
}

// forgetUser drops everything remembered about userID; call it after any
// write to the user's row.
func forgetUser(ctx context.Context, userID string) {
	if m := memoFrom(ctx); m != nil {
		m.mu.Lock()
		delete(m.users, userID)
		delete(m.rights, userID)
		m.mu.Unlock()
	}
}

// forgetCourseAdminRights drops the remembered rights of userID; call it
// after adding or removing one of their course-admin assignments.
func forgetCourseAdminRights(ctx context.Context, userID string) {
	if m := memoFrom(ctx); m != nil {
		m.mu.Lock()
		delete(m.rights, userID)
		m.mu.Unlock()
	}
}

// forgetAllCourseAdminRights drops every remembered rights list; call it
// after a write that changes courses (rights carry the course's number and
// name, and deleting a course cascades to its admins).
func forgetAllCourseAdminRights(ctx context.Context) {
	if m := memoFrom(ctx); m != nil {
		m.mu.Lock()
		clear(m.rights)
		m.mu.Unlock()
	}
}

// UpsertUser creates or updates a user record
func UpsertUser(ctx context.Context, userID, email, name string) (*User, error) {
	forgetUser(ctx, userID)
	result, err := queries.UpsertUser(ctx, db.UpsertUserParams{
		UserID:       userID,
		Email:        email,
		Name:         name,
		IsSuperAdmin: pgtype.Bool{Bool: false, Valid: true},
	})
	if err != nil {
		return nil, err
	}

	u := &User{
		UserID:       result.UserID,
		Email:        result.Email,
		Name:         result.Name,
		IsSuperAdmin: result.IsSuperAdmin.Bool,
		CreatedAt:    result.CreatedAt.Time,
		UpdatedAt:    result.UpdatedAt.Time,
	}
	// The RETURNING row is the current row, so later GetUser calls in this
	// request need not read it again.
	rememberUser(ctx, userID, u)
	return u, nil
}

// EnsureUserExists inserts a placeholder row (empty email and name) for a
// user that has none, and leaves an existing row untouched. Auth uses it when
// Clerk cannot be reached, so a stored profile is never overwritten with
// empty values.
func EnsureUserExists(ctx context.Context, userID string) error {
	forgetUser(ctx, userID)
	return queries.InsertUserIfMissing(ctx, userID)
}

// userSyncTTL is how long auth trusts an earlier Clerk profile sync before
// fetching the profile and upserting the user again.
const userSyncTTL = 10 * time.Minute

// UserRecentlySynced reports whether MarkUserSynced recorded a sync of userID
// within the last userSyncTTL on this instance. Without a cache it is always
// false, so auth syncs on every request as before.
func UserRecentlySynced(userID string) bool {
	if cacheInstance == nil || keyBuilder == nil {
		return false
	}
	raw, err := cacheInstance.Get(keyBuilder.UserSync(userID))
	if err != nil {
		return false
	}
	at, err := strconv.ParseInt(string(raw), 10, 64)
	if err != nil {
		return false
	}
	age := time.Since(time.Unix(at, 0))
	return age >= 0 && age < userSyncTTL
}

// MarkUserSynced records that userID's Clerk profile was just written to the
// users table. Call it only when both the Clerk fetch and the upsert
// succeeded.
func MarkUserSynced(userID string) {
	MarkUserSyncedAt(userID, time.Now())
}

// MarkUserSyncedAt records a sync of userID at the given time (tests use it
// to age a sync past the TTL).
func MarkUserSyncedAt(userID string, at time.Time) {
	if cacheInstance == nil || keyBuilder == nil {
		return
	}
	_ = cacheInstance.Set(keyBuilder.UserSync(userID), []byte(strconv.FormatInt(at.Unix(), 10)))
}

// GetUser retrieves a user by ID. Within a request (WithRequestMemo) the row
// is read at most once.
func GetUser(ctx context.Context, userID string) (*User, error) {
	if u, ok := memoUser(ctx, userID); ok {
		if u == nil {
			return nil, ErrNotFound
		}
		return u, nil
	}
	result, err := queries.GetUser(ctx, userID)
	if err == sql.ErrNoRows || err == pgx.ErrNoRows {
		rememberUser(ctx, userID, nil)
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	u := &User{
		UserID:       result.UserID,
		Email:        result.Email,
		Name:         result.Name,
		IsSuperAdmin: result.IsSuperAdmin.Bool,
		CreatedAt:    result.CreatedAt.Time,
		UpdatedAt:    result.UpdatedAt.Time,
	}
	rememberUser(ctx, userID, u)
	return u, nil
}

// GetUserByEmail retrieves a user by email address (case-insensitive match)
func GetUserByEmail(ctx context.Context, email string) (*User, error) {
	result, err := queries.GetUserByEmail(ctx, strings.TrimSpace(email))
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

// CanUserAccessCourse checks if user can access a specific course (non-admin)
func CanUserAccessCourse(ctx context.Context, userID string, courseID int32) bool {
	canAccess, err := queries.CanUserAccessCourse(ctx, db.CanUserAccessCourseParams{
		UserID:   userID,
		CourseID: courseID,
	})
	return err == nil && canAccess
}

// isCourseAdminOf reports whether userID is an admin of courseID, from the
// (memoised) course-admin rights. A lookup error counts as not an admin.
func isCourseAdminOf(ctx context.Context, userID string, courseID int32) bool {
	rights, err := GetUserCourseAdminRights(ctx, userID)
	if err != nil {
		return false
	}
	return slices.ContainsFunc(rights, func(r CourseAdminRight) bool { return r.CourseID == courseID })
}

// IsUserAnyAdmin checks if user is admin of any course or super admin
func IsUserAnyAdmin(ctx context.Context, userID string) bool {
	if IsUserSuperAdmin(ctx, userID) {
		return true
	}
	rights, err := GetUserCourseAdminRights(ctx, userID)
	return err == nil && len(rights) > 0
}

// IsUserOnlyCourseAdmin checks if user is admin of a specific course (no
// super admin override)
func IsUserOnlyCourseAdmin(ctx context.Context, userID string, courseID int32) bool {
	return isCourseAdminOf(ctx, userID, courseID)
}

// IsUserCourseOrSuperAdmin checks if user is admin of a specific course or super admin
func IsUserCourseOrSuperAdmin(ctx context.Context, userID string, courseID int32) bool {
	return IsUserSuperAdmin(ctx, userID) || isCourseAdminOf(ctx, userID, courseID)
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
	return isCourseAdminOf(ctx, userID, courseID.Int32)
}

// GetUserCourseAdminRights returns all courses a user is admin of. Within a
// request (WithRequestMemo) the rights are read at most once.
func GetUserCourseAdminRights(ctx context.Context, userID string) ([]CourseAdminRight, error) {
	if rights, ok := memoRights(ctx, userID); ok {
		return rights, nil
	}
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

	rememberRights(ctx, userID, rights)
	return rights, nil
}
