-- Grammar points management queries

-- name: CreateGrammarPoint :one
INSERT INTO grammar_points (story_id, name, description)
VALUES ($1, $2, $3)
RETURNING grammar_point_id, story_id, name, description, created_at;

-- name: GetGrammarPoint :one
SELECT grammar_point_id, story_id, name, description, created_at
FROM grammar_points
WHERE grammar_point_id = $1;

-- name: GetGrammarPointByName :one
SELECT grammar_point_id, story_id, name, description, created_at
FROM grammar_points
WHERE name = $1 AND story_id = $2;

-- name: ListGrammarPoints :many
SELECT grammar_point_id, story_id, name, description, created_at
FROM grammar_points
ORDER BY name;

-- name: UpdateGrammarPoint :one
UPDATE grammar_points
SET name = $2, description = $3
WHERE grammar_point_id = $1
RETURNING grammar_point_id, story_id, name, description, created_at;

-- name: DeleteGrammarPoint :exec
DELETE FROM grammar_points WHERE grammar_point_id = $1;

-- name: GetStoryGrammarPoints :many
SELECT grammar_point_id, name, description
FROM grammar_points
WHERE story_id = $1
ORDER BY name;

-- name: GetStoriesWithGrammarPoint :many
SELECT s.story_id, s.week_number, s.day_letter, s.video_url, s.last_revision, s.author_id, s.author_name, s.course_id
FROM stories s
JOIN grammar_points gp ON s.story_id = gp.story_id
WHERE gp.grammar_point_id = $1
ORDER BY s.week_number, s.day_letter;

-- name: ClearStoryGrammarPoints :exec
DELETE FROM grammar_points WHERE story_id = $1;
