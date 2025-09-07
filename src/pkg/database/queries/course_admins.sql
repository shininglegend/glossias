-- Course admin management queries

-- name: AddCourseAdmin :one
INSERT INTO course_admins (course_id, user_id)
VALUES ($1, $2)
RETURNING course_id, user_id, assigned_at;

-- name: RemoveCourseAdmin :exec
DELETE FROM course_admins
WHERE course_id = $1 AND user_id = $2;

-- name: GetCourseAdmins :many
SELECT ca.course_id, ca.user_id, ca.assigned_at, u.email, u.name
FROM course_admins ca
JOIN users u ON ca.user_id = u.user_id
WHERE ca.course_id = $1
ORDER BY ca.assigned_at;

-- name: GetUserCourseAdminRights :many
SELECT ca.course_id, ca.assigned_at, c.course_number, c.name as course_name
FROM course_admins ca
JOIN courses c ON ca.course_id = c.course_id
WHERE ca.user_id = $1
ORDER BY c.course_number;

-- name: IsUserCourseAdmin :one
SELECT EXISTS(
    SELECT 1 FROM course_admins
    WHERE course_id = $1 AND user_id = $2
) as is_admin;

-- name: IsUserAdminOfAnyCourse :one
SELECT EXISTS(
    SELECT 1 FROM course_admins
    WHERE user_id = $1
) as is_admin;

-- name: CanUserAccessCourse :one
SELECT EXISTS(
    SELECT 1 FROM users u
    LEFT JOIN course_admins ca ON u.user_id = ca.user_id
    WHERE u.user_id = $1
    AND (u.is_super_admin = true OR ca.course_id = $2)
) as can_access;
