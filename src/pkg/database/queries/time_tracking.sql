-- Time tracking queries

-- name: CreateTimeEntry :one
INSERT INTO user_time_tracking (user_id, route, story_id, started_at)
VALUES ($1, $2, $3, $4)
RETURNING tracking_id, user_id, route, story_id, started_at, ended_at, total_time_seconds, created_at;

-- name: CreateCompleteTimeEntry :one
INSERT INTO user_time_tracking (user_id, route, story_id, started_at, ended_at, total_time_seconds)
VALUES ($1, $2, $3, $4, $5, $6)
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

-- name: GetRecentTimeEntriesForUser :many
SELECT tracking_id, user_id, route, story_id, started_at, ended_at, total_time_seconds, created_at
FROM user_time_tracking
WHERE user_id = $1 AND created_at >= $2
ORDER BY created_at DESC;

-- name: GetUserStoryTimeTracking :one
SELECT
    COALESCE(SUM(CASE WHEN route LIKE '%vocab%' THEN total_time_seconds END), 0) as vocab_time_seconds,
    COALESCE(SUM(CASE WHEN route LIKE '%grammar%' THEN total_time_seconds END), 0) as grammar_time_seconds,
    COALESCE(SUM(CASE WHEN route LIKE '%translate%' THEN total_time_seconds END), 0) as translation_time_seconds,
    COALESCE(SUM(CASE WHEN route LIKE '%audio%' OR route LIKE '%video%' THEN total_time_seconds END), 0) as video_time_seconds
FROM user_time_tracking
WHERE user_id = $1 AND story_id = $2 AND ended_at IS NOT NULL;

-- name: FindRecentSimilarTimeEntry :one
SELECT tracking_id, total_time_seconds
FROM user_time_tracking
WHERE user_id = $1
  AND route = $2
  AND story_id IS NOT DISTINCT FROM $3
  AND created_at >= $4
ORDER BY created_at DESC
LIMIT 1;

-- name: UpsertTimeEntry :one
INSERT INTO user_time_tracking (user_id, route, story_id, started_at, ended_at, total_time_seconds)
VALUES ($1, $2, $3, $4, $5, $6)
ON CONFLICT (tracking_id)
DO UPDATE SET
    total_time_seconds = GREATEST(user_time_tracking.total_time_seconds, EXCLUDED.total_time_seconds),
    ended_at = EXCLUDED.ended_at
RETURNING tracking_id, user_id, route, story_id, started_at, ended_at, total_time_seconds, created_at;

-- name: UpdateTimeEntryIfBigger :exec
UPDATE user_time_tracking
SET total_time_seconds = GREATEST(total_time_seconds, $2),
    ended_at = $3
WHERE tracking_id = $1;

-- name: AccumulateTimeEntry :exec
UPDATE user_time_tracking
SET total_time_seconds = COALESCE(total_time_seconds, 0) + $2,
    ended_at = $3
WHERE tracking_id = $1;
