-- name: AddUserToCourse :exec
INSERT INTO course_users (course_id, user_id, enrolled_at)
VALUES ($1, $2, CURRENT_TIMESTAMP);

-- name: AddMultiUsersToCourse :exec
INSERT INTO course_users (course_id, user_id, enrolled_at)
SELECT $1, unnest($2::text[]), CURRENT_TIMESTAMP
ON CONFLICT (course_id, user_id) DO NOTHING;

-- name: RemoveUserFromCourse :exec
DELETE FROM course_users
WHERE course_id = $1 AND user_id = $2;

-- name: DeleteAllUsersFromCourse :exec
DELETE FROM course_users
WHERE course_id = $1;

-- name: GetCoursesForUser :many
SELECT c.course_id, c.course_number, c.name, c.description, cu.enrolled_at
FROM courses c
JOIN course_users cu ON c.course_id = cu.course_id
WHERE cu.user_id = $1
ORDER BY c.course_number;

-- name: GetUsersForCourse :many
SELECT u.user_id, u.email, u.name, cu.enrolled_at
FROM users u
JOIN course_users cu ON u.user_id = cu.user_id
WHERE cu.course_id = $1
ORDER BY u.name;

-- name: CanUserAccessCourse :one
SELECT EXISTS(
    SELECT 1 FROM users u
    LEFT JOIN course_admins ca ON u.user_id = ca.user_id
    LEFT JOIN course_users cu ON u.user_id = cu.user_id
    WHERE u.user_id = $1
    AND (u.is_super_admin = true OR ca.course_id = $2 OR cu.course_id = $2)
) as can_access;
