-- +goose Up
-- Append-only history of the Produce grader's system prompt. The newest row is
-- the active version; editing appends rather than updates so every grading
-- log row can point at the exact prompt it ran with. Seeded on startup from
-- the built-in default when empty.
CREATE TABLE IF NOT EXISTS produce_grading_prompts (
    id SERIAL PRIMARY KEY,
    prompt_text TEXT NOT NULL,
    note TEXT,
    created_by TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- The log references the prompt version instead of carrying the full text.
ALTER TABLE produce_grading_log DROP COLUMN IF EXISTS system_prompt;
ALTER TABLE produce_grading_log ADD COLUMN IF NOT EXISTS prompt_id INTEGER REFERENCES produce_grading_prompts (id);

-- +goose Down
ALTER TABLE produce_grading_log DROP COLUMN IF EXISTS prompt_id;
ALTER TABLE produce_grading_log ADD COLUMN IF NOT EXISTS system_prompt TEXT;
DROP TABLE IF EXISTS produce_grading_prompts;
