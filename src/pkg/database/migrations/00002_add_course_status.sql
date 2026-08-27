-- +goose Up
-- Migration: Add status column to course_users table
-- Date: 2026-01-17
-- Description: Adds status tracking for student-course enrollment (active, past, future)

-- Add status column with default 'active' and constraint
ALTER TABLE course_users 
ADD COLUMN IF NOT EXISTS status TEXT DEFAULT 'active' 
CHECK (status IN ('active', 'past', 'future'));

-- Create index for efficient filtering by status
CREATE INDEX IF NOT EXISTS idx_course_users_status ON course_users(status);

-- Update any existing NULL values to 'active' (backward compatibility)
UPDATE course_users SET status = 'active' WHERE status IS NULL;

-- +goose Down
DROP INDEX IF EXISTS idx_course_users_status;
ALTER TABLE course_users DROP COLUMN IF EXISTS status;
