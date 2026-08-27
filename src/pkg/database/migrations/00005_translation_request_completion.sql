-- +goose Up
-- Migration: Translation request completion marker
-- Date: 2026-08-27
-- Description: The Translate phase (SUMMER_2026.md T9) now saves requested
-- lines after every reveal instead of only at the end, so a row existing no
-- longer means the phase was finished. completed_at records completion.
-- Every pre-existing row was written on completion, so backfill from created_at.
ALTER TABLE translation_requests ADD COLUMN IF NOT EXISTS completed_at TIMESTAMP;
UPDATE translation_requests SET completed_at = created_at WHERE completed_at IS NULL;

-- +goose Down
ALTER TABLE translation_requests DROP COLUMN IF EXISTS completed_at;
