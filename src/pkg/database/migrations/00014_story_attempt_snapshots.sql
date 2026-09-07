-- +goose Up
-- Student redo keeps the official (first) score by freezing it before the
-- live answer rows are wiped. Later attempts are practice; admin can still
-- open each frozen score. Live tables stay current-attempt only.

CREATE TABLE story_attempts (
    attempt_id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    user_id TEXT NOT NULL REFERENCES users (user_id) ON DELETE CASCADE,
    story_id INTEGER NOT NULL REFERENCES stories (story_id) ON DELETE CASCADE,
    attempt_number INTEGER NOT NULL CHECK (attempt_number >= 1),
    started_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    completed_at TIMESTAMPTZ,
    UNIQUE (user_id, story_id, attempt_number)
);

CREATE INDEX story_attempts_story_user_idx
    ON story_attempts (story_id, user_id);

CREATE TABLE story_attempt_score_snapshots (
    attempt_id BIGINT PRIMARY KEY REFERENCES story_attempts (attempt_id) ON DELETE CASCADE,
    snapshot JSONB NOT NULL,
    snapshot_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- +goose Down
DROP TABLE IF EXISTS story_attempt_score_snapshots;
DROP TABLE IF EXISTS story_attempts;
