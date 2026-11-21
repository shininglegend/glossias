-- name: InsertUserIfMissing :exec
-- Creates a placeholder row for a user auth could not load from Clerk, so
-- rows that reference users still have a parent. Never touches an existing
-- row: a Clerk outage must not blank a stored email or name.
INSERT INTO users (user_id, email, name)
VALUES ($1, '', '')
ON CONFLICT DO NOTHING;

-- name: GetUsersForCourseWithRoles :many
-- The course roster with each user's role in one query, replacing a super
-- admin and a course admin lookup per user.
SELECT u.user_id, u.email, u.name, cu.enrolled_at, cu.status,
       COALESCE(u.is_super_admin, false)::boolean AS is_super_admin,
       EXISTS(
           SELECT 1 FROM course_admins ca
           WHERE ca.course_id = cu.course_id AND ca.user_id = u.user_id
       )::boolean AS is_course_admin
FROM users u
JOIN course_users cu ON u.user_id = cu.user_id
WHERE cu.course_id = $1
ORDER BY u.name;
