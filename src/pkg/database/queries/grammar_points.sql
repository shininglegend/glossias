-- Grammar points management queries

-- name: CreateGrammarPoint :one
INSERT INTO grammar_points (name, description)
VALUES ($1, $2)
RETURNING grammar_point_id, name, description, created_at;

-- name: GetGrammarPoint :one
SELECT grammar_point_id, name, description, created_at
FROM grammar_points
WHERE grammar_point_id = $1;

-- name: GetGrammarPointByName :one
SELECT grammar_point_id, name, description, created_at
FROM grammar_points
WHERE name = $1;

-- name: ListGrammarPoints :many
SELECT grammar_point_id, name, description, created_at
FROM grammar_points
ORDER BY name;

-- name: UpdateGrammarPoint :one
UPDATE grammar_points
SET name = $2, description = $3
WHERE grammar_point_id = $1
RETURNING grammar_point_id, name, description, created_at;

-- name: DeleteGrammarPoint :exec
DELETE FROM grammar_points WHERE grammar_point_id = $1;

-- name: AddGrammarPointToStory :exec
INSERT INTO story_grammar_points (story_id, grammar_point_id)
VALUES ($1, $2)
ON CONFLICT (story_id, grammar_point_id) DO NOTHING;

-- name: RemoveGrammarPointFromStory :exec
DELETE FROM story_grammar_points
WHERE story_id = $1 AND grammar_point_id = $2;

-- name: GetStoryGrammarPoints :many
SELECT gp.grammar_point_id, gp.name, gp.description
FROM grammar_points gp
JOIN story_grammar_points sgp ON gp.grammar_point_id = sgp.grammar_point_id
WHERE sgp.story_id = $1
ORDER BY gp.name;

-- name: GetStoriesWithGrammarPoint :many
SELECT s.story_id, s.week_number, s.day_letter, s.video_url, s.last_revision, s.author_id, s.author_name, s.course_id
FROM stories s
JOIN story_grammar_points sgp ON s.story_id = sgp.story_id
WHERE sgp.grammar_point_id = $1
ORDER BY s.week_number, s.day_letter;

-- name: ClearStoryGrammarPoints :exec
DELETE FROM story_grammar_points WHERE story_id = $1;
