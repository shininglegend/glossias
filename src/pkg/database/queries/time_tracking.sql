-- Time tracking queries

-- name: CreateTimeEntry :one
INSERT INTO user_time_tracking (user_id, route, story_id, started_at)
VALUES ($1, $2, $3, $4)
RETURNING tracking_id, user_id, route, story_id, started_at, ended_at, total_time_seconds, created_at;

-- name: UpdateTimeEntry :one
UPDATE user_time_tracking
SET ended_at = $2, total_time_seconds = $3
WHERE tracking_id = $1
RETURNING tracking_id, user_id, route, story_id, started_at, ended_at, total_time_seconds, created_at;

-- name: GetTimeEntriesForStory :many
SELECT tracking_id, user_id, route, story_id, started_at, ended_at, total_time_seconds, created_at
FROM user_time_tracking
WHERE story_id = $1
ORDER BY started_at DESC;

-- name: GetTimeEntriesForUser :many
SELECT tracking_id, user_id, route, story_id, started_at, ended_at, total_time_seconds, created_at
FROM user_time_tracking
WHERE user_id = $1
ORDER BY started_at DESC;

-- name: GetTimeEntryByID :one
SELECT tracking_id, user_id, route, story_id, started_at, ended_at, total_time_seconds, created_at
FROM user_time_tracking
WHERE tracking_id = $1;

-- name: GetActiveTimeEntry :one
SELECT tracking_id, user_id, route, story_id, started_at, ended_at, total_time_seconds, created_at
FROM user_time_tracking
WHERE user_id = $1 AND route = $2 AND story_id IS NOT DISTINCT FROM $3 AND ended_at IS NULL
ORDER BY started_at DESC
LIMIT 1;

-- name: CloseTimeEntry :exec
UPDATE user_time_tracking
SET ended_at = $2, total_time_seconds = $3
WHERE tracking_id = $1;

-- Anonymous time tracking queries

-- name: CreateAnonymousTimeEntry :one
INSERT INTO anonymous_time_tracking (session_id, route, story_id, started_at)
VALUES ($1, $2, $3, $4)
RETURNING tracking_id, session_id, route, story_id, started_at, ended_at, total_time_seconds, created_at;

-- name: UpdateAnonymousTimeEntry :one
UPDATE anonymous_time_tracking
SET ended_at = $2, total_time_seconds = $3
WHERE tracking_id = $1
RETURNING tracking_id, session_id, route, story_id, started_at, ended_at, total_time_seconds, created_at;

-- name: GetAnonymousTimeEntryByID :one
SELECT tracking_id, session_id, route, story_id, started_at, ended_at, total_time_seconds, created_at
FROM anonymous_time_tracking
WHERE tracking_id = $1;

-- name: GetActiveAnonymousTimeEntry :one
SELECT tracking_id, session_id, route, story_id, started_at, ended_at, total_time_seconds, created_at
FROM anonymous_time_tracking
WHERE session_id = $1 AND route = $2 AND story_id IS NOT DISTINCT FROM $3 AND ended_at IS NULL
ORDER BY started_at DESC
LIMIT 1;

-- name: CloseAnonymousTimeEntry :exec
UPDATE anonymous_time_tracking
SET ended_at = $2, total_time_seconds = $3
WHERE tracking_id = $1;

-- name: DeleteOldAnonymousEntries :exec
DELETE FROM anonymous_time_tracking
WHERE created_at < $1;
