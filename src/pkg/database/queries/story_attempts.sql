-- Story-level attempts: one row per student redo cycle. The live answer
-- tables stay current-attempt only; completed attempts freeze a score JSON.

-- name: CreateStoryAttempt :one
INSERT INTO story_attempts (user_id, story_id, attempt_number)
VALUES ($1, $2, $3)
RETURNING attempt_id, user_id, story_id, attempt_number, started_at, completed_at;

-- name: GetUserStoryAttempts :many
SELECT attempt_id, user_id, story_id, attempt_number, started_at, completed_at
FROM story_attempts
WHERE user_id = $1 AND story_id = $2
ORDER BY attempt_number;

-- name: GetUserStoryAttemptsWithSnapshots :many
SELECT sa.attempt_id, sa.attempt_number, sa.started_at, sa.completed_at,
       (snap.attempt_id IS NOT NULL)::BOOLEAN AS has_snapshot
FROM story_attempts sa
LEFT JOIN story_attempt_score_snapshots snap ON snap.attempt_id = sa.attempt_id
WHERE sa.user_id = $1 AND sa.story_id = $2
ORDER BY sa.attempt_number;

-- name: GetLatestUserStoryAttempt :one
SELECT attempt_id, user_id, story_id, attempt_number, started_at, completed_at
FROM story_attempts
WHERE user_id = $1 AND story_id = $2
ORDER BY attempt_number DESC
LIMIT 1;

-- name: GetUserStoryAttemptByNumber :one
SELECT attempt_id, user_id, story_id, attempt_number, started_at, completed_at
FROM story_attempts
WHERE user_id = $1 AND story_id = $2 AND attempt_number = $3;

-- name: MarkStoryAttemptComplete :exec
UPDATE story_attempts
SET completed_at = CURRENT_TIMESTAMP
WHERE attempt_id = $1 AND completed_at IS NULL;

-- name: UpsertAttemptScoreSnapshot :exec
INSERT INTO story_attempt_score_snapshots (attempt_id, snapshot, snapshot_at)
VALUES ($1, $2, CURRENT_TIMESTAMP)
ON CONFLICT (attempt_id) DO UPDATE
SET snapshot = EXCLUDED.snapshot,
    snapshot_at = CURRENT_TIMESTAMP;

-- name: GetAttemptScoreSnapshot :one
SELECT snapshot, snapshot_at
FROM story_attempt_score_snapshots
WHERE attempt_id = $1;

-- name: GetUserStoryAttemptSnapshotByNumber :one
SELECT snap.snapshot, snap.snapshot_at, sa.attempt_id, sa.attempt_number
FROM story_attempts sa
JOIN story_attempt_score_snapshots snap ON snap.attempt_id = sa.attempt_id
WHERE sa.user_id = $1 AND sa.story_id = $2 AND sa.attempt_number = $3;

-- name: DeleteUserStoryAttempts :execrows
DELETE FROM story_attempts WHERE user_id = $1 AND story_id = $2;
