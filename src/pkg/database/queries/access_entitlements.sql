-- name: InsertAccessEntitlement :one
INSERT INTO access_entitlements (
    user_id, course_id, source, starts_at, expires_at, amount_cents,
    stripe_checkout_session_id, stripe_payment_intent_id, granted_by
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9
)
RETURNING entitlement_id, user_id, course_id, source, starts_at, expires_at,
          amount_cents, stripe_checkout_session_id, stripe_payment_intent_id, granted_by;

-- name: GetAccessEntitlementBySessionID :one
SELECT entitlement_id, user_id, course_id, source, starts_at, expires_at,
       amount_cents, stripe_checkout_session_id, stripe_payment_intent_id, granted_by
FROM access_entitlements
WHERE stripe_checkout_session_id = $1;

-- name: GetActiveEntitlementForUserCourse :one
SELECT entitlement_id, user_id, course_id, source, starts_at, expires_at,
       amount_cents, stripe_checkout_session_id, stripe_payment_intent_id, granted_by
FROM access_entitlements
WHERE user_id = $1 AND course_id = $2 AND expires_at > CURRENT_TIMESTAMP
ORDER BY expires_at DESC
LIMIT 1;

-- name: GetLatestEntitlementExpiryForUserCourse :one
SELECT expires_at
FROM access_entitlements
WHERE user_id = $1 AND course_id = $2
ORDER BY expires_at DESC
LIMIT 1;

-- name: ListActiveEntitlementsForUser :many
SELECT course_id, MAX(expires_at)::timestamptz AS expires_at
FROM access_entitlements
WHERE user_id = $1 AND expires_at > CURRENT_TIMESTAMP
GROUP BY course_id;

-- name: ListLatestEntitlementsForCourseUsers :many
SELECT DISTINCT ON (user_id) user_id, expires_at
FROM access_entitlements
WHERE course_id = $1 AND user_id = ANY(sqlc.arg(user_ids)::text[])
ORDER BY user_id, expires_at DESC;

-- name: ExpireEntitlementsByPaymentIntent :execrows
UPDATE access_entitlements
SET expires_at = CURRENT_TIMESTAMP
WHERE stripe_payment_intent_id = $1 AND expires_at > CURRENT_TIMESTAMP;

-- name: GetUserStripeCustomerID :one
SELECT stripe_customer_id
FROM users
WHERE user_id = $1;

-- name: SetUserStripeCustomerID :exec
UPDATE users
SET stripe_customer_id = $2, updated_at = CURRENT_TIMESTAMP
WHERE user_id = $1;

-- name: ListCoursesTrialFlags :many
SELECT course_id, is_trial
FROM courses
WHERE course_id = ANY(sqlc.arg(course_ids)::int[]);

-- name: CountStoriesForCourses :many
SELECT course_id, COUNT(*)::int AS story_count
FROM course_stories
WHERE course_id = ANY(sqlc.arg(course_ids)::int[])
GROUP BY course_id;

-- name: IsUserEnrolledInCourse :one
SELECT EXISTS(
    SELECT 1 FROM course_users
    WHERE user_id = $1 AND course_id = $2
) AS enrolled;
