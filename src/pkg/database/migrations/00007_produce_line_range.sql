-- +goose Up
-- Migration: Produce segments span a line range
-- Date: 2026-08-28
-- Description: The Produce editor now seeds a segment from a run of story
-- lines rather than a single one, so its Hebrew and English can be
-- reconstructed later ("Sync"). A single line is just a range where start
-- equals end.

ALTER TABLE produce_segments ADD COLUMN IF NOT EXISTS line_start INTEGER;
ALTER TABLE produce_segments ADD COLUMN IF NOT EXISTS line_end INTEGER;

UPDATE produce_segments
SET line_start = line_number, line_end = line_number
WHERE line_number IS NOT NULL;

ALTER TABLE produce_segments DROP COLUMN IF EXISTS line_number;

-- +goose Down
ALTER TABLE produce_segments ADD COLUMN IF NOT EXISTS line_number INTEGER;

UPDATE produce_segments
SET line_number = line_start
WHERE line_start IS NOT NULL;

ALTER TABLE produce_segments DROP COLUMN IF EXISTS line_end;
ALTER TABLE produce_segments DROP COLUMN IF EXISTS line_start;
