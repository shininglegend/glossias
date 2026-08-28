-- Versioned system prompt for the Produce AI grader (append-only).

-- The newest row is the active prompt.
-- name: GetActiveProduceGradingPrompt :one
SELECT id, prompt_text, note, created_by, created_at
FROM produce_grading_prompts
ORDER BY id DESC
LIMIT 1;

-- name: ListProduceGradingPrompts :many
SELECT id, prompt_text, note, created_by, created_at
FROM produce_grading_prompts
ORDER BY id DESC;

-- name: InsertProduceGradingPrompt :one
INSERT INTO produce_grading_prompts (prompt_text, note, created_by)
VALUES ($1, $2, $3)
RETURNING id, prompt_text, note, created_by, created_at;

-- name: CountProduceGradingPrompts :one
SELECT COUNT(*) FROM produce_grading_prompts;
