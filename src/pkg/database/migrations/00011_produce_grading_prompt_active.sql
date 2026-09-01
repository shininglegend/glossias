-- +goose Up
-- Which prompt version is active is a pointer, not "the newest row", so an
-- earlier version can be made active again without duplicating it. Single
-- row table; seeded to the newest version.
CREATE TABLE IF NOT EXISTS produce_grading_active_prompt (
    singleton BOOLEAN PRIMARY KEY DEFAULT TRUE CHECK (singleton),
    prompt_id INTEGER NOT NULL REFERENCES produce_grading_prompts (id),
    activated_by TEXT,
    activated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

INSERT INTO produce_grading_active_prompt (prompt_id)
SELECT id FROM produce_grading_prompts ORDER BY id DESC LIMIT 1
ON CONFLICT (singleton) DO NOTHING;

-- +goose Down
DROP TABLE IF EXISTS produce_grading_active_prompt;
