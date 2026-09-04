-- +goose Up
-- Migration: phase column on user_time_tracking
-- Date: 2026-08-31
-- Description: Adds a phase column set from the route at write time
-- (models.PhaseFromRoute), so per-phase time bucketing and per-phase resets
-- match on an exact value instead of route LIKE substring patterns (T16).
-- The backfill mirrors the CASE buckets those queries used; rows whose route
-- matches no phase (story list, admin pages) stay NULL and are counted only
-- in per-story totals, exactly as before.

ALTER TABLE user_time_tracking ADD COLUMN IF NOT EXISTS phase TEXT;

UPDATE user_time_tracking SET phase = CASE
    WHEN route LIKE '%identify%' THEN 'identify'
    WHEN route LIKE '%translate%' THEN 'translate'
    WHEN route LIKE '%produce%' THEN 'produce'
    WHEN route LIKE '%recall%' THEN 'recall'
    WHEN route LIKE '%video%' OR route LIKE '%audio%' THEN 'video'
    WHEN route LIKE '%vocab%' THEN 'vocab'
    WHEN route LIKE '%grammar%' THEN 'grammar'
END
WHERE phase IS NULL;

-- +goose Down
ALTER TABLE user_time_tracking DROP COLUMN IF EXISTS phase;
