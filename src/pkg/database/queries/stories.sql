-- Core story operations

-- name: GetStory :one
SELECT s.story_id, s.week_number, s.day_letter, s.video_url, s.last_revision, s.author_id, s.author_name, s.course_id
FROM stories s
WHERE s.story_id = $1;

-- GetStoriesVideoURLs is the video half of content readiness for many stories
-- at once (admin GET /api/stories).
-- name: GetStoriesVideoURLs :many
SELECT story_id, video_url
FROM stories
WHERE story_id = ANY(sqlc.arg(story_ids)::int[])
ORDER BY story_id;

-- name: GetAllStories :many
SELECT s.story_id, s.week_number, s.day_letter, s.video_url, s.last_revision, s.author_id, s.author_name, s.course_id
FROM stories s
ORDER BY s.week_number, s.day_letter;

-- name: CreateStory :one
INSERT INTO stories (week_number, day_letter, video_url, author_id, author_name, course_id)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING story_id, last_revision;

-- name: UpdateStory :exec
UPDATE stories
SET week_number = $2, day_letter = $3, video_url = $4, author_id = $5, author_name = $6, course_id = $7, last_revision = CURRENT_TIMESTAMP
WHERE story_id = $1;

-- name: DeleteStory :exec
DELETE FROM stories WHERE story_id = $1;

-- name: GetAllStoriesBasic :many
SELECT DISTINCT s.story_id, s.week_number, s.day_letter, st.title, s.course_id
FROM stories s
JOIN story_titles st ON s.story_id = st.story_id
WHERE st.language_code = $1 OR $1 = ''
ORDER BY s.week_number, s.day_letter;

-- name: GetAllStoriesForUser :many
SELECT s.story_id, s.week_number, s.day_letter, st.title, s.course_id,
       COALESCE((
           SELECT ARRAY_AGG(cs.course_id ORDER BY cs.course_id)
           FROM course_stories cs
           WHERE cs.story_id = s.story_id
       ), ARRAY[]::INTEGER[])::INTEGER[] AS course_ids
FROM stories s
JOIN story_titles st ON s.story_id = st.story_id
WHERE (st.language_code = $1 OR $1 = '')
  AND (
      NOT EXISTS (SELECT 1 FROM course_stories cs WHERE cs.story_id = s.story_id)
      OR EXISTS (
          SELECT 1 FROM course_stories cs
          JOIN course_users cu ON cu.course_id = cs.course_id
          WHERE cs.story_id = s.story_id AND cu.user_id = $2 AND cu.status = 'active'
      )
      OR EXISTS (
          SELECT 1 FROM course_stories cs
          JOIN course_admins ca ON ca.course_id = cs.course_id
          WHERE cs.story_id = s.story_id AND ca.user_id = $2
      )
  )
ORDER BY s.week_number, s.day_letter;

-- name: GetAllStoriesWithTitles :many
SELECT DISTINCT s.story_id, s.week_number, s.day_letter, st.title, st.language_code
FROM stories s
JOIN story_titles st ON s.story_id = st.story_id
ORDER BY s.week_number, s.day_letter;

-- name: GetStoryWithDescription :one
SELECT s.story_id, s.week_number, s.day_letter, s.video_url, s.last_revision, s.author_id, s.author_name, s.course_id,
       sd.language_code, sd.description_text
FROM stories s
LEFT JOIN story_descriptions sd ON s.story_id = sd.story_id
WHERE s.story_id = $1;

-- name: GetStoriesByCourse :many
SELECT s.story_id, s.week_number, s.day_letter, s.video_url, s.last_revision, s.author_id, s.author_name, s.course_id
FROM stories s
JOIN course_stories cs ON cs.story_id = s.story_id
WHERE cs.course_id = $1
ORDER BY s.week_number, s.day_letter;

-- name: GetCourseIdForStory :one
SELECT course_id FROM stories WHERE story_id = $1;

-- name: GetStoriesForUserCourses :many
SELECT DISTINCT s.story_id, s.week_number, s.day_letter, s.video_url, s.last_revision, s.author_id, s.author_name, s.course_id
FROM stories s
JOIN course_stories cs ON cs.story_id = s.story_id
JOIN course_admins ca ON cs.course_id = ca.course_id
WHERE ca.user_id = $1
ORDER BY s.week_number, s.day_letter;

-- name: GetCourseStoriesWithTitles :many
SELECT s.story_id, s.week_number, s.day_letter, st.title
FROM stories s
JOIN course_stories cs ON cs.story_id = s.story_id
JOIN story_titles st ON s.story_id = st.story_id AND st.language_code = $2
WHERE cs.course_id = $1
ORDER BY s.week_number, s.day_letter;
