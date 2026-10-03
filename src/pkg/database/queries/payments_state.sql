-- name: GetPaymentsState :one
SELECT paused, reason, detail, changed_at
FROM payments_state
WHERE id = 1;

-- name: UpsertPaymentsState :exec
INSERT INTO payments_state (id, paused, reason, detail, changed_at)
VALUES (1, $1, $2, $3, CURRENT_TIMESTAMP)
ON CONFLICT (id) DO UPDATE
SET paused = EXCLUDED.paused,
    reason = EXCLUDED.reason,
    detail = EXCLUDED.detail,
    changed_at = CURRENT_TIMESTAMP;
