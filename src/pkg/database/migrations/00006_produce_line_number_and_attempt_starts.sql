-- +goose Up
-- Migration: Produce segment placement and attempt timers
-- Date: 2026-08-27
-- Description: The Produce phase (SUMMER_2026.md T12) shows the story text with
-- the segment's slot marked, which needs an authored line; and its 90-second
-- countdown must survive a reload, which needs the attempt's start time
-- recorded server-side.

-- Which story line the segment's reference sentence belongs to (1-based, like
-- story_lines.line_number). NULL for segments authored before this column
-- existed; the student page then falls back to searching the text.
ALTER TABLE produce_segments ADD COLUMN IF NOT EXISTS line_number INTEGER;

-- When a student started writing a segment. One row per (user, segment); the
-- remaining time is derived from it, so a reload resumes the same countdown.
-- A student's rows are only ever inserted, never updated — the first start
-- is the one that counts.
CREATE TABLE IF NOT EXISTS produce_attempt_starts (
    user_id TEXT NOT NULL REFERENCES users (user_id) ON DELETE CASCADE,
    story_id INTEGER NOT NULL REFERENCES stories (story_id) ON DELETE CASCADE,
    segment_id INTEGER NOT NULL REFERENCES produce_segments (id) ON DELETE CASCADE,
    started_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (user_id, segment_id)
);

CREATE INDEX IF NOT EXISTS idx_produce_attempt_starts_user_story ON produce_attempt_starts (user_id, story_id);

-- +goose Down
DROP TABLE IF EXISTS produce_attempt_starts CASCADE;
ALTER TABLE produce_segments DROP COLUMN IF EXISTS line_number;
