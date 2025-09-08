-- Story titles
-- name: GetStoryTitles :many
SELECT story_id, language_code, title
FROM story_titles
WHERE story_id = $1;

-- name: GetStoryTitle :one
SELECT title
FROM story_titles
WHERE story_id = $1 AND language_code = $2;

-- name: UpsertStoryTitle :exec
INSERT INTO story_titles (story_id, language_code, title)
VALUES ($1, $2, $3)
ON CONFLICT (story_id, language_code)
DO UPDATE SET title = EXCLUDED.title;

-- name: DeleteStoryTitles :exec
DELETE FROM story_titles WHERE story_id = $1;

-- Story descriptions
-- name: GetStoryDescription :one
SELECT description_text
FROM story_descriptions
WHERE story_id = $1 AND language_code = $2;

-- name: UpsertStoryDescription :exec
INSERT INTO story_descriptions (story_id, language_code, description_text)
VALUES ($1, $2, $3)
ON CONFLICT (story_id, language_code)
DO UPDATE SET description_text = EXCLUDED.description_text;

-- name: DeleteStoryDescriptions :exec
DELETE FROM story_descriptions WHERE story_id = $1;

-- Story lines
-- name: GetStoryLines :many
SELECT story_id, line_number, text, english_translation
FROM story_lines
WHERE story_id = $1
ORDER BY line_number;

-- name: GetStoryLine :one
SELECT story_id, line_number, text, english_translation
FROM story_lines
WHERE story_id = $1 AND line_number = $2;

-- name: GetLineText :one
SELECT text FROM story_lines
WHERE story_id = $1 AND line_number = $2;

-- name: UpsertStoryLine :exec
INSERT INTO story_lines (story_id, line_number, text, english_translation)
VALUES ($1, $2, $3, $4)
ON CONFLICT (story_id, line_number)
DO UPDATE SET text = EXCLUDED.text, english_translation = EXCLUDED.english_translation;

-- name: DeleteStoryLine :exec
DELETE FROM story_lines WHERE story_id = $1 AND line_number = $2;

-- name: DeleteAllStoryLines :exec
DELETE FROM story_lines WHERE story_id = $1;
