-- Course management queries

-- name: CreateCourse :one
INSERT INTO courses (course_number, name, description)
VALUES ($1, $2, $3)
RETURNING course_id, course_number, name, description, created_at, updated_at;

-- name: GetCourse :one
SELECT course_id, course_number, name, description, created_at, updated_at
FROM courses
WHERE course_id = $1;

-- name: GetCourseByNumber :one
SELECT course_id, course_number, name, description, created_at, updated_at
FROM courses
WHERE course_number = $1;

-- name: ListCourses :many
SELECT course_id, course_number, name, description, created_at, updated_at
FROM courses
ORDER BY course_number;

-- name: UpdateCourse :one
UPDATE courses
SET course_number = $2, name = $3, description = $4, updated_at = CURRENT_TIMESTAMP
WHERE course_id = $1
RETURNING course_id, course_number, name, description, created_at, updated_at;

-- name: DeleteCourse :exec
DELETE FROM courses WHERE course_id = $1;

-- name: GetAdminCoursesForUser :many
SELECT c.course_id, c.course_number, c.name, c.description, c.created_at, c.updated_at
FROM courses c
JOIN course_admins ca ON c.course_id = ca.course_id
WHERE ca.user_id = $1
ORDER BY c.course_number;
