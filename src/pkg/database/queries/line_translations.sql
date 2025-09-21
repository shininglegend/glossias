-- Line translations management queries

-- name: GetLineTranslations :many
SELECT story_id, line_number, language_code, translation_text
FROM line_translations
WHERE story_id = $1 AND line_number = $2
ORDER BY language_code;

-- name: GetLineTranslation :one
SELECT translation_text
FROM line_translations
WHERE story_id = $1 AND line_number = $2 AND language_code = $3;

-- name: UpsertLineTranslation :exec
INSERT INTO line_translations (story_id, line_number, language_code, translation_text)
VALUES ($1, $2, $3, $4)
ON CONFLICT (story_id, line_number, language_code)
DO UPDATE SET translation_text = EXCLUDED.translation_text;

-- name: DeleteLineTranslation :exec
DELETE FROM line_translations
WHERE story_id = $1 AND line_number = $2 AND language_code = $3;

-- name: GetAllTranslationsForStory :many
SELECT story_id, line_number, language_code, translation_text
FROM line_translations
WHERE story_id = $1
ORDER BY line_number, language_code;

-- name: GetTranslationsByLanguage :many
SELECT story_id, line_number, translation_text
FROM line_translations
WHERE story_id = $1 AND language_code = $2
ORDER BY line_number;
