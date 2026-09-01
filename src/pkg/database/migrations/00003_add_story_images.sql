-- +goose Up
-- Migration: Add story_images table for image storage
-- Date: 2026-08-27
-- Description: Creates the story_images table to track image files in Supabase bucket

CREATE TABLE IF NOT EXISTS story_images (
    image_id SERIAL PRIMARY KEY,
    story_id INTEGER REFERENCES stories (story_id) ON DELETE CASCADE,
    file_path TEXT NOT NULL, -- Supabase storage path
    file_bucket TEXT NOT NULL, -- Supabase bucket name
    label TEXT NOT NULL, -- e.g., "target_vocab", "recall"
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_story_images_label ON story_images (story_id, label);

-- +goose Down
DROP INDEX IF EXISTS idx_story_images_label;
DROP TABLE IF EXISTS story_images;
