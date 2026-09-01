-- +goose Up
-- The Produce phase runs Hebrew → English: the student is shown a Hebrew
-- segment drawn from the story and writes the English. The columns already
-- held exactly these two texts, but were named for the reverse direction
-- (English prompt, Hebrew reference). Rename them to say what they are now.
ALTER TABLE produce_segments RENAME COLUMN reference_hebrew TO hebrew_text;
ALTER TABLE produce_segments RENAME COLUMN english_text TO reference_english;

-- +goose Down
ALTER TABLE produce_segments RENAME COLUMN hebrew_text TO reference_hebrew;
ALTER TABLE produce_segments RENAME COLUMN reference_english TO english_text;
