-- +goose Up
-- Migration: Summer 2026 story-flow content tables
-- Date: 2026-08-27
-- Description: Adds the content and answer-log tables for the Identify, Produce
-- and Recall phases (SUMMER_2026.md F2/F3).

-- Five target vocabulary words per story, each with pronunciation audio and a
-- matching picture. Occurrences in the text are found by joining
-- vocabulary_items on lexical_form, so no per-occurrence rows are needed.
CREATE TABLE IF NOT EXISTS target_vocabulary (
    id SERIAL PRIMARY KEY,
    story_id INTEGER NOT NULL REFERENCES stories (story_id) ON DELETE CASCADE,
    lexical_form TEXT NOT NULL,
    audio_path TEXT, -- word pronunciation, audio-files bucket
    audio_bucket TEXT,
    correct_image_path TEXT, -- the matching picture, images bucket
    image_bucket TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (story_id, lexical_form)
);

CREATE INDEX IF NOT EXISTS idx_target_vocabulary_story ON target_vocabulary (story_id);

-- Two Produce segments per story: an English prompt the student renders into
-- Hebrew, plus the reference translation used for self-comparison and grading.
CREATE TABLE IF NOT EXISTS produce_segments (
    id SERIAL PRIMARY KEY,
    story_id INTEGER NOT NULL REFERENCES stories (story_id) ON DELETE CASCADE,
    segment_order INTEGER NOT NULL CHECK (segment_order IN (1, 2)),
    english_text TEXT NOT NULL,
    reference_hebrew TEXT NOT NULL,
    grammar_point_id INTEGER REFERENCES grammar_points (grammar_point_id) ON DELETE SET NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (story_id, segment_order)
);

CREATE INDEX IF NOT EXISTS idx_produce_segments_story ON produce_segments (story_id);

-- The contrastive grammar explanation shown after both Produce segments.
CREATE TABLE IF NOT EXISTS story_produce_explanations (
    story_id INTEGER PRIMARY KEY REFERENCES stories (story_id) ON DELETE CASCADE,
    explanation_text TEXT NOT NULL,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Student attempts at a Produce segment, graded asynchronously by the AI grader
-- (T13). ai_score stays NULL when grading fails so a grading outage never
-- blocks progression.
CREATE TABLE IF NOT EXISTS produce_submissions (
    id SERIAL PRIMARY KEY,
    user_id TEXT NOT NULL REFERENCES users (user_id) ON DELETE CASCADE,
    story_id INTEGER NOT NULL REFERENCES stories (story_id) ON DELETE CASCADE,
    segment_id INTEGER NOT NULL REFERENCES produce_segments (id) ON DELETE CASCADE,
    student_text TEXT NOT NULL,
    ai_score INTEGER CHECK (ai_score BETWEEN 0 AND 100),
    ai_feedback TEXT,
    graded_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_produce_submissions_user_story ON produce_submissions (user_id, story_id);
CREATE INDEX IF NOT EXISTS idx_produce_submissions_segment ON produce_submissions (segment_id);

-- Five Recall sentences per story, one per target word, shuffled for the
-- drag-and-drop sequencing exercise.
CREATE TABLE IF NOT EXISTS recall_sentences (
    id SERIAL PRIMARY KEY,
    story_id INTEGER NOT NULL REFERENCES stories (story_id) ON DELETE CASCADE,
    sequence_order INTEGER NOT NULL CHECK (sequence_order BETWEEN 1 AND 5),
    hebrew_text TEXT NOT NULL,
    target_vocab_id INTEGER REFERENCES target_vocabulary (id) ON DELETE SET NULL,
    image_path TEXT, -- images bucket
    image_bucket TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (story_id, sequence_order)
);

CREATE INDEX IF NOT EXISTS idx_recall_sentences_story ON recall_sentences (story_id);

-- Identify phase answer logs (append-only, mirroring vocab_correct_answers).
-- line_number is the story line whose narration triggered the picture quiz.
CREATE TABLE IF NOT EXISTS identify_correct_answers (
    id SERIAL PRIMARY KEY,
    user_id TEXT NOT NULL REFERENCES users (user_id) ON DELETE CASCADE,
    story_id INTEGER NOT NULL REFERENCES stories (story_id) ON DELETE CASCADE,
    line_number INTEGER NOT NULL,
    target_vocab_id INTEGER NOT NULL REFERENCES target_vocabulary (id) ON DELETE CASCADE,
    attempted_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (story_id, line_number) REFERENCES story_lines (story_id, line_number) ON DELETE CASCADE
);

-- selected_target_vocab_id is the target word whose picture the student wrongly
-- picked; the five options are always the story's own target words.
CREATE TABLE IF NOT EXISTS identify_incorrect_answers (
    id SERIAL PRIMARY KEY,
    user_id TEXT NOT NULL REFERENCES users (user_id) ON DELETE CASCADE,
    story_id INTEGER NOT NULL REFERENCES stories (story_id) ON DELETE CASCADE,
    line_number INTEGER NOT NULL,
    target_vocab_id INTEGER NOT NULL REFERENCES target_vocabulary (id) ON DELETE CASCADE,
    selected_target_vocab_id INTEGER NOT NULL REFERENCES target_vocabulary (id) ON DELETE CASCADE,
    attempted_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (story_id, line_number) REFERENCES story_lines (story_id, line_number) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_identify_correct_user_story ON identify_correct_answers (user_id, story_id);
CREATE INDEX IF NOT EXISTS idx_identify_correct_story ON identify_correct_answers (story_id);
CREATE INDEX IF NOT EXISTS idx_identify_incorrect_user_story ON identify_incorrect_answers (user_id, story_id);

-- Recall phase answer logs (append-only). One row per sentence per attempt:
-- correct when the student placed it at its sequence_order, incorrect
-- otherwise, with the position they chose.
CREATE TABLE IF NOT EXISTS recall_correct_answers (
    id SERIAL PRIMARY KEY,
    user_id TEXT NOT NULL REFERENCES users (user_id) ON DELETE CASCADE,
    story_id INTEGER NOT NULL REFERENCES stories (story_id) ON DELETE CASCADE,
    recall_sentence_id INTEGER NOT NULL REFERENCES recall_sentences (id) ON DELETE CASCADE,
    attempted_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS recall_incorrect_answers (
    id SERIAL PRIMARY KEY,
    user_id TEXT NOT NULL REFERENCES users (user_id) ON DELETE CASCADE,
    story_id INTEGER NOT NULL REFERENCES stories (story_id) ON DELETE CASCADE,
    recall_sentence_id INTEGER NOT NULL REFERENCES recall_sentences (id) ON DELETE CASCADE,
    selected_position INTEGER NOT NULL,
    attempted_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_recall_correct_user_story ON recall_correct_answers (user_id, story_id);
CREATE INDEX IF NOT EXISTS idx_recall_correct_story ON recall_correct_answers (story_id);
CREATE INDEX IF NOT EXISTS idx_recall_incorrect_user_story ON recall_incorrect_answers (user_id, story_id);

-- +goose Down
DROP TABLE IF EXISTS recall_incorrect_answers CASCADE;
DROP TABLE IF EXISTS recall_correct_answers CASCADE;
DROP TABLE IF EXISTS identify_incorrect_answers CASCADE;
DROP TABLE IF EXISTS identify_correct_answers CASCADE;
DROP TABLE IF EXISTS recall_sentences CASCADE;
DROP TABLE IF EXISTS produce_submissions CASCADE;
DROP TABLE IF EXISTS story_produce_explanations CASCADE;
DROP TABLE IF EXISTS produce_segments CASCADE;
DROP TABLE IF EXISTS target_vocabulary CASCADE;
