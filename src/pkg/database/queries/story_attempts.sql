-- Story-level attempts: one row per *completed* run of a story, each with a
-- frozen score JSON. The live answer tables hold only the in-progress attempt,
-- so "current attempt number" is always (completed count + 1).

-- ArchiveStoryAttempt files the live score as the next completed attempt in
-- one statement. The snapshot is bound as text and cast, because the pool runs
-- the simple protocol, under which pgx sends []byte as a bytea literal that
-- jsonb rejects (SQLSTATE 22P02).
-- name: ArchiveStoryAttempt :one
WITH attempt AS (
    INSERT INTO story_attempts (user_id, story_id, attempt_number)
    SELECT @user_id::TEXT, @story_id::INT, COALESCE(MAX(sa.attempt_number), 0) + 1
    FROM story_attempts sa
    WHERE sa.user_id = @user_id::TEXT AND sa.story_id = @story_id::INT
    RETURNING attempt_id, attempt_number
), snap AS (
    INSERT INTO story_attempt_score_snapshots (attempt_id, snapshot)
    SELECT attempt_id, CAST(@snapshot::TEXT AS JSONB) FROM attempt
    RETURNING attempt_id
)
SELECT attempt.attempt_number FROM attempt;

-- name: ListUserStoryCompletedAttempts :many
SELECT sa.attempt_number, sa.completed_at
FROM story_attempts sa
JOIN story_attempt_score_snapshots snap ON snap.attempt_id = sa.attempt_id
WHERE sa.user_id = $1 AND sa.story_id = $2
ORDER BY sa.attempt_number;

-- name: GetUserStoryAttemptSnapshotByNumber :one
SELECT snap.snapshot, snap.snapshot_at, sa.attempt_id, sa.attempt_number
FROM story_attempts sa
JOIN story_attempt_score_snapshots snap ON snap.attempt_id = sa.attempt_id
WHERE sa.user_id = $1 AND sa.story_id = $2 AND sa.attempt_number = $3;

-- name: ListUserStoryAttemptSnapshots :many
SELECT sa.attempt_id, sa.attempt_number, sa.completed_at, snap.snapshot
FROM story_attempts sa
JOIN story_attempt_score_snapshots snap ON snap.attempt_id = sa.attempt_id
WHERE sa.user_id = $1 AND sa.story_id = $2
ORDER BY sa.attempt_number;

-- name: DeleteUserStoryAttempts :execrows
DELETE FROM story_attempts WHERE user_id = $1 AND story_id = $2;

-- name: DeleteUserStoryAttemptByNumber :execrows
DELETE FROM story_attempts WHERE user_id = $1 AND story_id = $2 AND attempt_number = $3;

-- Renumbering after a delete happens one row at a time in ascending order
-- (see models.DeleteArchivedAttempt): the unique (user, story, number) key is
-- not deferrable, so a single "SET attempt_number = attempt_number - 1" could
-- collide mid-statement depending on row order.
-- name: ListUserStoryAttemptsAfter :many
SELECT attempt_id, attempt_number FROM story_attempts
WHERE user_id = $1 AND story_id = $2 AND attempt_number > $3
ORDER BY attempt_number;

-- name: SetStoryAttemptNumber :exec
UPDATE story_attempts SET attempt_number = $2 WHERE attempt_id = $1;
