-- +goose Up
-- One row per grading run of a Produce submission, for inspecting what the
-- grader was asked and what it answered. Failures are logged too (error set,
-- score NULL). Blank attempts graded locally have no model or prompts.
CREATE TABLE IF NOT EXISTS produce_grading_log (
    id SERIAL PRIMARY KEY,
    submission_id INTEGER NOT NULL REFERENCES produce_submissions (id) ON DELETE CASCADE,
    user_id TEXT NOT NULL,
    story_id INTEGER NOT NULL,
    segment_id INTEGER NOT NULL,
    hebrew_text TEXT NOT NULL,
    reference_english TEXT NOT NULL,
    student_text TEXT NOT NULL,
    grammar_point_name TEXT,
    model TEXT,
    system_prompt TEXT,
    user_prompt TEXT,
    raw_response TEXT,
    stop_reason TEXT,
    input_tokens INTEGER,
    output_tokens INTEGER,
    cache_read_input_tokens INTEGER,
    latency_ms INTEGER,
    score INTEGER,
    feedback TEXT,
    error TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_produce_grading_log_submission ON produce_grading_log (submission_id);
CREATE INDEX IF NOT EXISTS idx_produce_grading_log_story ON produce_grading_log (story_id, created_at DESC);

-- +goose Down
DROP TABLE IF EXISTS produce_grading_log;
