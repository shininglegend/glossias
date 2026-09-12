-- Sections are child courses that share the parent's stories.

-- name: AttachCourseSection :exec
INSERT INTO course_sections (parent_course_id, section_course_id)
VALUES ($1, $2);

-- name: DetachCourseSection :exec
DELETE FROM course_sections
WHERE parent_course_id = $1 AND section_course_id = $2;

-- name: ListCourseSections :many
SELECT c.course_id, c.course_number, c.name
FROM course_sections cs
JOIN courses c ON c.course_id = cs.section_course_id
WHERE cs.parent_course_id = $1
ORDER BY c.course_number;

-- name: ListAllCourseSections :many
SELECT cs.parent_course_id, cs.section_course_id
FROM course_sections cs;

-- name: GetSectionParent :one
SELECT parent_course_id
FROM course_sections
WHERE section_course_id = $1;

-- name: CourseIsSection :one
SELECT EXISTS(
    SELECT 1 FROM course_sections WHERE section_course_id = $1
) AS is_section;

-- name: CourseHasSections :one
SELECT EXISTS(
    SELECT 1 FROM course_sections WHERE parent_course_id = $1
) AS has_sections;

-- name: CopyCourseStoryLinks :exec
INSERT INTO course_stories (course_id, story_id)
SELECT sqlc.arg(to_course_id), cs.story_id
FROM course_stories cs
WHERE cs.course_id = sqlc.arg(from_course_id)
ON CONFLICT DO NOTHING;

-- name: ListUserIDsForCourses :many
SELECT DISTINCT user_id
FROM course_users
WHERE course_id = ANY(sqlc.arg(course_ids)::int[]);

-- name: ListUserSectionsForParent :many
SELECT cu.user_id, cu.course_id, c.course_number, c.name
FROM course_users cu
JOIN course_sections cs ON cs.section_course_id = cu.course_id
JOIN courses c ON c.course_id = cu.course_id
WHERE cs.parent_course_id = $1;

-- name: RemoveUsersFromSiblingSections :exec
DELETE FROM course_users
WHERE user_id = ANY(sqlc.arg(user_ids)::text[])
  AND course_id IN (
    SELECT section_course_id FROM course_sections
    WHERE parent_course_id = sqlc.arg(parent_id)
      AND section_course_id <> sqlc.arg(keep_section_id)
  );
