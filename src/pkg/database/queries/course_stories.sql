-- Story-to-course membership (symlink). stories.course_id remains the owner.

-- name: LinkStoryToCourse :exec
INSERT INTO course_stories (course_id, story_id)
VALUES ($1, $2)
ON CONFLICT DO NOTHING;

-- name: UnlinkStoryFromCourse :exec
DELETE FROM course_stories
WHERE course_id = $1 AND story_id = $2;

-- name: ListCoursesForStory :many
SELECT c.course_id, c.course_number, c.name
FROM course_stories cs
JOIN courses c ON c.course_id = cs.course_id
WHERE cs.story_id = $1
ORDER BY c.course_number;

-- name: ListStoryCourseIDs :many
SELECT course_id
FROM course_stories
WHERE story_id = $1
ORDER BY course_id;

-- name: ListCourseIDsForStories :many
SELECT story_id, course_id
FROM course_stories
WHERE story_id = ANY(sqlc.arg(story_ids)::int[])
ORDER BY story_id, course_id;

-- name: StoryLinkedToCourse :one
SELECT EXISTS(
    SELECT 1 FROM course_stories
    WHERE course_id = $1 AND story_id = $2
) AS linked;

-- CanUserAccessStory: super admin, orphan (no links), or member/admin of any linked course.
-- Enrollment status is not filtered here — listing applies the active-only rule separately.
-- name: CanUserAccessStory :one
SELECT EXISTS(
    SELECT 1 FROM users u
    WHERE u.user_id = $1
      AND (
          u.is_super_admin = true
          OR NOT EXISTS (
              SELECT 1 FROM course_stories cs WHERE cs.story_id = $2
          )
          OR EXISTS (
              SELECT 1 FROM course_stories cs
              JOIN course_admins ca ON ca.course_id = cs.course_id AND ca.user_id = $1
              WHERE cs.story_id = $2
          )
          OR EXISTS (
              SELECT 1 FROM course_stories cs
              JOIN course_users cu ON cu.course_id = cs.course_id AND cu.user_id = $1
              WHERE cs.story_id = $2
          )
      )
) AS can_access;

-- name: IsUserAdminOfLinkedStory :one
SELECT EXISTS(
    SELECT 1 FROM users u
    WHERE u.user_id = $1
      AND (
          u.is_super_admin = true
          OR EXISTS (
              SELECT 1 FROM course_stories cs
              JOIN course_admins ca ON ca.course_id = cs.course_id AND ca.user_id = $1
              WHERE cs.story_id = $2
          )
      )
) AS is_admin;
