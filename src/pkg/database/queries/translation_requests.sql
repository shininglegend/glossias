-- Translation requests management queries

-- name: CreateTranslationRequest :one
INSERT INTO translation_requests (user_id, story_id, requested_lines)
VALUES ($1, $2, $3)
RETURNING request_id, user_id, story_id, requested_lines, created_at;

-- name: GetTranslationRequest :one
SELECT request_id, user_id, story_id, requested_lines, created_at
FROM translation_requests
WHERE user_id = $1 AND story_id = $2;

-- name: GetTranslationRequestByID :one
SELECT request_id, user_id, story_id, requested_lines, created_at
FROM translation_requests
WHERE request_id = $1;

-- name: TranslationRequestExists :one
SELECT EXISTS(
    SELECT 1 FROM translation_requests
    WHERE user_id = $1 AND story_id = $2
) as exists;

-- name: DeleteTranslationRequest :exec
DELETE FROM translation_requests
WHERE user_id = $1 AND story_id = $2;

-- name: GetUserTranslationRequests :many
SELECT request_id, user_id, story_id, requested_lines, created_at
FROM translation_requests
WHERE user_id = $1
ORDER BY created_at DESC;

-- name: GetStoryTranslationRequests :many
SELECT request_id, user_id, story_id, requested_lines, created_at
FROM translation_requests
WHERE story_id = $1
ORDER BY created_at DESC;

-- name: GetUserTranslationStatusForStory :one
SELECT
    EXISTS(SELECT 1 FROM translation_requests tr WHERE tr.user_id = $1 AND tr.story_id = $2) as completed,
    COALESCE((SELECT tr2.requested_lines FROM translation_requests tr2 WHERE tr2.user_id = $1 AND tr2.story_id = $2), ARRAY[]::INTEGER[]) as requested_lines;
