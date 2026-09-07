-- +goose Up
-- Per-sentence audio override for Recall. Empty means the student hears the
-- story-line narration that makes up the instructor sentence (possibly several
-- files chained). A path here replaces that default, the same way edited
-- hebrew_text replaces the picked story text.

ALTER TABLE recall_sentences
    ADD COLUMN IF NOT EXISTS audio_path TEXT,
    ADD COLUMN IF NOT EXISTS audio_bucket TEXT;

-- +goose Down
ALTER TABLE recall_sentences DROP COLUMN IF EXISTS audio_path;
ALTER TABLE recall_sentences DROP COLUMN IF EXISTS audio_bucket;
