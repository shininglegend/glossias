-- name: ListCourseAccessForUser :many
-- One row per existing course in course_ids: its trial flag, whether the user
-- is enrolled (any status, like IsUserEnrolledInCourse) and whether the user
-- holds an unexpired entitlement (like ListActiveEntitlementsForUser).
SELECT
    c.course_id,
    c.is_trial,
    EXISTS(
        SELECT 1 FROM course_users cu
        WHERE cu.user_id = sqlc.arg(user_id) AND cu.course_id = c.course_id
    ) AS enrolled,
    EXISTS(
        SELECT 1 FROM access_entitlements ae
        WHERE ae.user_id = sqlc.arg(user_id) AND ae.course_id = c.course_id
          AND ae.expires_at > CURRENT_TIMESTAMP
    ) AS entitled
FROM courses c
WHERE c.course_id = ANY(sqlc.arg(course_ids)::int[]);
