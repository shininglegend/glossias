-- Core story operations

-- name: GetStory :one
SELECT s.story_id, s.week_number, s.day_letter, s.grammar_point, s.last_revision, s.author_id, s.author_name, s.course_id
FROM stories s
WHERE s.story_id = $1;

-- name: GetAllStories :many
SELECT s.story_id, s.week_number, s.day_letter, s.grammar_point, s.last_revision, s.author_id, s.author_name, s.course_id
FROM stories s
ORDER BY s.week_number, s.day_letter;

-- name: CreateStory :one
INSERT INTO stories (week_number, day_letter, grammar_point, author_id, author_name, course_id)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING story_id, last_revision;

-- name: UpdateStory :exec
UPDATE stories
SET week_number = $2, day_letter = $3, grammar_point = $4, author_id = $5, author_name = $6, course_id = $7, last_revision = CURRENT_TIMESTAMP
WHERE story_id = $1;

-- name: DeleteStory :exec
DELETE FROM stories WHERE story_id = $1;



-- name: GetAllStoriesBasic :many
SELECT DISTINCT s.story_id, s.week_number, s.day_letter, st.title
FROM stories s
JOIN story_titles st ON s.story_id = st.story_id
WHERE st.language_code = $1 OR $1 = ''
ORDER BY s.week_number, s.day_letter;

-- name: GetAllStoriesWithTitles :many
SELECT DISTINCT s.story_id, s.week_number, s.day_letter, st.title, st.language_code
FROM stories s
JOIN story_titles st ON s.story_id = st.story_id
ORDER BY s.week_number, s.day_letter;

-- name: GetStoryWithDescription :one
SELECT s.story_id, s.week_number, s.day_letter, s.grammar_point, s.last_revision, s.author_id, s.author_name, s.course_id,
       sd.language_code, sd.description_text
FROM stories s
LEFT JOIN story_descriptions sd ON s.story_id = sd.story_id
WHERE s.story_id = $1;

-- name: GetStoriesByCourse :many
SELECT s.story_id, s.week_number, s.day_letter, s.grammar_point, s.last_revision, s.author_id, s.author_name, s.course_id
FROM stories s
WHERE s.course_id = $1
ORDER BY s.week_number, s.day_letter;

-- name: GetStoriesForUserCourses :many
SELECT s.story_id, s.week_number, s.day_letter, s.grammar_point, s.last_revision, s.author_id, s.author_name, s.course_id
FROM stories s
JOIN course_admins ca ON s.course_id = ca.course_id
WHERE ca.user_id = $1
ORDER BY s.week_number, s.day_letter;
