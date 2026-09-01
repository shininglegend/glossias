-- Versioned system prompt for the Produce AI grader. Versions are append-only;
-- produce_grading_active_prompt points at the one in use.

-- name: GetActiveProduceGradingPrompt :one
SELECT p.id, p.prompt_text, p.note, p.created_by, p.created_at
FROM produce_grading_active_prompt a
JOIN produce_grading_prompts p ON p.id = a.prompt_id;

-- name: ListProduceGradingPrompts :many
SELECT id, prompt_text, note, created_by, created_at
FROM produce_grading_prompts
ORDER BY id DESC;

-- name: GetProduceGradingPrompt :one
SELECT id, prompt_text, note, created_by, created_at
FROM produce_grading_prompts
WHERE id = $1;

-- Finds an existing version with exactly this text, so re-saving an earlier
-- version re-activates it instead of duplicating it.
-- name: GetProduceGradingPromptByText :one
SELECT id, prompt_text, note, created_by, created_at
FROM produce_grading_prompts
WHERE prompt_text = $1
ORDER BY id DESC
LIMIT 1;

-- name: InsertProduceGradingPrompt :one
INSERT INTO produce_grading_prompts (prompt_text, note, created_by)
VALUES ($1, $2, $3)
RETURNING id, prompt_text, note, created_by, created_at;

-- name: SetActiveProduceGradingPrompt :exec
INSERT INTO produce_grading_active_prompt (prompt_id, activated_by)
VALUES ($1, $2)
ON CONFLICT (singleton) DO UPDATE
SET prompt_id = EXCLUDED.prompt_id,
    activated_by = EXCLUDED.activated_by,
    activated_at = CURRENT_TIMESTAMP;

-- name: CountProduceGradingPrompts :one
SELECT COUNT(*) FROM produce_grading_prompts;
