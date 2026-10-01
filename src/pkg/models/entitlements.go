package models

import (
	"context"
	"errors"
	"os"
	"strings"
	"time"

	"glossias/src/pkg/generated/db"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

const accessYear = 365 * 24 * time.Hour

var (
	ErrPaymentRequired  = errors.New("payment_required")
	ErrTrialCourse      = errors.New("trial course does not require payment")
	ErrNotEnrolled      = errors.New("not enrolled in course")
	ErrAlreadyHasAccess = errors.New("already has access")
)

// PaymentRequiredError is returned for enrolled students who still need to pay.
type PaymentRequiredError struct {
	PayableCourseIDs []int
}

func (e *PaymentRequiredError) Error() string { return ErrPaymentRequired.Error() }
func (e *PaymentRequiredError) Unwrap() error { return ErrPaymentRequired }

// PaywallEnabled is true when PAYWALL_ENABLED is 1/true/yes.
func PaywallEnabled() bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv("PAYWALL_ENABLED"))) {
	case "1", "true", "yes":
		return true
	default:
		return false
	}
}

// CheckStoryContentAccess is nil when the user may open story content.
// ErrNotFound hides the story; *PaymentRequiredError is a 402.
func CheckStoryContentAccess(ctx context.Context, userID string, storyID int32) error {
	if !CanUserAccessStory(ctx, userID, storyID) {
		return ErrNotFound
	}
	if !PaywallEnabled() || userID == "" {
		return nil
	}
	if CanUserAdminLinkedStory(ctx, userID, storyID) {
		return nil
	}
	courseIDs, err := ListStoryCourseIDs(ctx, storyID)
	if err != nil {
		return err
	}
	if len(courseIDs) == 0 {
		return nil
	}
	locked, payable, err := storyLockState(ctx, userID, courseIDs)
	if err != nil {
		return err
	}
	if !locked {
		return nil
	}
	if len(payable) == 0 {
		return ErrNotFound
	}
	return &PaymentRequiredError{PayableCourseIDs: payable}
}

// AnnotateStoryLocks sets Locked and PayableCourseIDs on listed stories.
func AnnotateStoryLocks(ctx context.Context, userID string, stories []Story) error {
	if !PaywallEnabled() || userID == "" || len(stories) == 0 {
		return nil
	}
	super := IsUserSuperAdmin(ctx, userID)
	adminIDs := map[int]bool{}
	if !super {
		rights, err := GetUserCourseAdminRights(ctx, userID)
		if err != nil {
			return err
		}
		for _, r := range rights {
			adminIDs[int(r.CourseID)] = true
		}
	}
	for i := range stories {
		ids := stories[i].Metadata.LinkedCourseIDs
		if super {
			continue
		}
		adminLinked := false
		for _, id := range ids {
			if adminIDs[id] {
				adminLinked = true
				break
			}
		}
		if adminLinked || len(ids) == 0 {
			continue
		}
		locked, payable, err := storyLockState(ctx, userID, ids)
		if err != nil {
			return err
		}
		stories[i].Metadata.Locked = locked
		stories[i].Metadata.PayableCourseIDs = payable
	}
	return nil
}

func storyLockState(ctx context.Context, userID string, courseIDs []int) (locked bool, payable []int, err error) {
	if len(courseIDs) == 0 {
		return false, nil, nil
	}
	ids32 := make([]int32, len(courseIDs))
	for i, id := range courseIDs {
		ids32[i] = int32(id)
	}
	flags, err := queries.ListCoursesTrialFlags(ctx, ids32)
	if err != nil {
		return false, nil, err
	}
	trial := map[int32]bool{}
	for _, f := range flags {
		trial[f.CourseID] = f.IsTrial
		if f.IsTrial {
			return false, nil, nil
		}
	}
	ents, err := queries.ListActiveEntitlementsForUser(ctx, userID)
	if err != nil {
		return false, nil, err
	}
	entitled := map[int32]bool{}
	for _, e := range ents {
		entitled[e.CourseID] = true
	}
	for _, id := range ids32 {
		if entitled[id] {
			return false, nil, nil
		}
	}
	for _, id := range ids32 {
		if trial[id] {
			continue
		}
		enrolled, err := queries.IsUserEnrolledInCourse(ctx, db.IsUserEnrolledInCourseParams{
			UserID:   userID,
			CourseID: id,
		})
		if err != nil {
			return false, nil, err
		}
		if enrolled {
			payable = append(payable, int(id))
		}
	}
	return len(payable) > 0, payable, nil
}

// AnnotateUserCourses fills is_trial, has_access, and expires_at.
func AnnotateUserCourses(ctx context.Context, userID string, courses []UserCourse) error {
	if len(courses) == 0 {
		return nil
	}
	super := IsUserSuperAdmin(ctx, userID)
	adminIDs := map[int]bool{}
	if !super {
		rights, err := GetUserCourseAdminRights(ctx, userID)
		if err != nil {
			return err
		}
		for _, r := range rights {
			adminIDs[int(r.CourseID)] = true
		}
	}
	ents, err := queries.ListActiveEntitlementsForUser(ctx, userID)
	if err != nil {
		return err
	}
	expByCourse := map[int]time.Time{}
	for _, e := range ents {
		if e.ExpiresAt.Valid {
			expByCourse[int(e.CourseID)] = e.ExpiresAt.Time
		}
	}
	for i := range courses {
		if courses[i].IsTrial || super || adminIDs[courses[i].CourseID] || !PaywallEnabled() {
			courses[i].HasAccess = true
		}
		if exp, ok := expByCourse[courses[i].CourseID]; ok {
			courses[i].HasAccess = true
			t := exp
			courses[i].ExpiresAt = &t
		}
	}
	return nil
}

type GrantAccessParams struct {
	UserID          string
	CourseID        int32
	Source          string
	AmountCents     *int32
	SessionID       string
	PaymentIntentID string
	GrantedBy       string
}

// GrantCourseAccess writes a year of access, extending from the current expiry.
func GrantCourseAccess(ctx context.Context, p GrantAccessParams) (*time.Time, error) {
	if p.Source == "" {
		p.Source = "comp"
	}
	now := time.Now()
	start := now
	exp := now.Add(accessYear)
	latest, err := queries.GetLatestEntitlementExpiryForUserCourse(ctx, db.GetLatestEntitlementExpiryForUserCourseParams{
		UserID:   p.UserID,
		CourseID: p.CourseID,
	})
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return nil, err
	}
	if err == nil && latest.Valid && latest.Time.After(now) {
		exp = latest.Time.Add(accessYear)
	}
	rows, err := queries.InsertAccessEntitlement(ctx, db.InsertAccessEntitlementParams{
		UserID:                  p.UserID,
		CourseID:                p.CourseID,
		Source:                  p.Source,
		StartsAt:                pgtype.Timestamptz{Time: start, Valid: true},
		ExpiresAt:               pgtype.Timestamptz{Time: exp, Valid: true},
		AmountCents:             int4Ptr(p.AmountCents),
		StripeCheckoutSessionID: textOrNull(p.SessionID),
		StripePaymentIntentID:   textOrNull(p.PaymentIntentID),
		GrantedBy:               textOrNull(p.GrantedBy),
	})
	if err != nil {
		return nil, err
	}
	if rows == 0 {
		// The session was already granted by the other path (confirm vs.
		// webhook). Nothing new was written, so there is no new expiry.
		return nil, nil
	}
	return &exp, nil
}

// GrantFromCheckoutSession is idempotent on Stripe session id: a repeat of a
// session that is already on record is a no-op, including when two callers
// race.
func GrantFromCheckoutSession(ctx context.Context, userID string, courseID int32, sessionID, paymentIntentID string, amountCents int32, source string) error {
	_, err := GrantCourseAccess(ctx, GrantAccessParams{
		UserID:          userID,
		CourseID:        courseID,
		Source:          source,
		AmountCents:     &amountCents,
		SessionID:       sessionID,
		PaymentIntentID: paymentIntentID,
	})
	return err
}

func ExpireEntitlementsForPaymentIntent(ctx context.Context, paymentIntentID string) error {
	_, err := queries.ExpireEntitlementsByPaymentIntent(ctx, textOrNull(paymentIntentID))
	return err
}

func ActiveAccessExpiresAt(ctx context.Context, userID string, courseID int32) (*time.Time, error) {
	row, err := queries.GetActiveEntitlementForUserCourse(ctx, db.GetActiveEntitlementForUserCourseParams{
		UserID:   userID,
		CourseID: courseID,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if !row.ExpiresAt.Valid {
		return nil, nil
	}
	t := row.ExpiresAt.Time
	return &t, nil
}

func UserStripeCustomerID(ctx context.Context, userID string) (string, error) {
	id, err := queries.GetUserStripeCustomerID(ctx, userID)
	if err != nil {
		return "", err
	}
	if !id.Valid {
		return "", nil
	}
	return id.String, nil
}

func SetUserStripeCustomerID(ctx context.Context, userID, customerID string) error {
	return queries.SetUserStripeCustomerID(ctx, db.SetUserStripeCustomerIDParams{
		UserID:           userID,
		StripeCustomerID: textOrNull(customerID),
	})
}

func IsUserEnrolledInCourse(ctx context.Context, userID string, courseID int32) (bool, error) {
	return queries.IsUserEnrolledInCourse(ctx, db.IsUserEnrolledInCourseParams{
		UserID:   userID,
		CourseID: courseID,
	})
}

func StoryCountForCourse(ctx context.Context, courseID int32) (int, error) {
	rows, err := queries.CountStoriesForCourses(ctx, []int32{courseID})
	if err != nil {
		return 0, err
	}
	if len(rows) == 0 {
		return 0, nil
	}
	return int(rows[0].StoryCount), nil
}

func StoryCountsForCourses(ctx context.Context, courseIDs []int32) (map[int32]int, error) {
	out := map[int32]int{}
	if len(courseIDs) == 0 {
		return out, nil
	}
	rows, err := queries.CountStoriesForCourses(ctx, courseIDs)
	if err != nil {
		return nil, err
	}
	for _, r := range rows {
		out[r.CourseID] = int(r.StoryCount)
	}
	return out, nil
}

func LatestEntitlementsForCourseUsers(ctx context.Context, courseID int32, userIDs []string) (map[string]time.Time, error) {
	out := map[string]time.Time{}
	if len(userIDs) == 0 {
		return out, nil
	}
	rows, err := queries.ListLatestEntitlementsForCourseUsers(ctx, db.ListLatestEntitlementsForCourseUsersParams{
		CourseID: courseID,
		UserIds:  userIDs,
	})
	if err != nil {
		return nil, err
	}
	for _, r := range rows {
		if r.ExpiresAt.Valid {
			out[r.UserID] = r.ExpiresAt.Time
		}
	}
	return out, nil
}

func textOrNull(s string) pgtype.Text {
	if s == "" {
		return pgtype.Text{}
	}
	return pgtype.Text{String: s, Valid: true}
}

func int4Ptr(v *int32) pgtype.Int4 {
	if v == nil {
		return pgtype.Int4{}
	}
	return pgtype.Int4{Int32: *v, Valid: true}
}
