-- schema.sql
-- Remove SQLite-specific PRAGMA
-- Replace AUTOINCREMENT with SERIAL

-- Users table for authentication and authorization
CREATE TABLE IF NOT EXISTS users (
    user_id TEXT PRIMARY KEY, -- Clerk user ID
    email TEXT UNIQUE NOT NULL,
    name TEXT NOT NULL,
    is_super_admin BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Courses table
CREATE TABLE IF NOT EXISTS courses (
    course_id SERIAL PRIMARY KEY,
    course_number TEXT UNIQUE NOT NULL,
    name TEXT NOT NULL,
    description TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Course administrators junction table
CREATE TABLE IF NOT EXISTS course_admins (
    course_id INTEGER REFERENCES courses (course_id) ON DELETE CASCADE,
    user_id TEXT REFERENCES users (user_id) ON DELETE CASCADE,
    assigned_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (course_id, user_id)
);

-- Course users junction table - assigns users to courses without admin privileges
CREATE TABLE IF NOT EXISTS course_users (
    course_id INTEGER REFERENCES courses (course_id) ON DELETE CASCADE,
    user_id TEXT REFERENCES users (user_id) ON DELETE CASCADE,
    enrolled_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (course_id, user_id)
);

-- Grammar points table - each point belongs to a specific story
CREATE TABLE IF NOT EXISTS grammar_points (
    grammar_point_id SERIAL PRIMARY KEY,
    story_id INTEGER NOT NULL REFERENCES stories (story_id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    description TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS stories (
    story_id SERIAL PRIMARY KEY,
    week_number INTEGER NOT NULL,
    day_letter TEXT NOT NULL,
    video_url TEXT, -- Added for video metadata
    last_revision TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    author_id TEXT NOT NULL,
    author_name TEXT NOT NULL,
    course_id INTEGER REFERENCES courses (course_id) ON DELETE SET NULL
);



CREATE TABLE IF NOT EXISTS story_titles (
    story_id INTEGER REFERENCES stories (story_id) ON DELETE CASCADE,
    language_code TEXT,
    title TEXT NOT NULL,
    PRIMARY KEY (story_id, language_code)
);

CREATE TABLE IF NOT EXISTS story_descriptions (
    story_id INTEGER REFERENCES stories (story_id) ON DELETE CASCADE,
    language_code TEXT,
    description_text TEXT NOT NULL,
    PRIMARY KEY (story_id, language_code)
);

CREATE TABLE IF NOT EXISTS story_lines (
    story_id INTEGER REFERENCES stories (story_id) ON DELETE CASCADE,
    line_number INTEGER,
    text TEXT NOT NULL,
    PRIMARY KEY (story_id, line_number)
);

CREATE TABLE IF NOT EXISTS line_translations (
    story_id INTEGER,
    line_number INTEGER,
    language_code TEXT,
    translation_text TEXT NOT NULL,
    PRIMARY KEY (story_id, line_number, language_code),
    FOREIGN KEY (story_id, line_number) REFERENCES story_lines (story_id, line_number) ON DELETE CASCADE
);

-- Audio files table for multiple audio files per line
CREATE TABLE IF NOT EXISTS line_audio_files (
    audio_file_id SERIAL PRIMARY KEY,
    story_id INTEGER,
    line_number INTEGER,
    file_path TEXT NOT NULL, -- Supabase storage path
    file_bucket TEXT NOT NULL, -- Supabase bucket name
    label TEXT NOT NULL, -- e.g., "complete", "vocab_missing"
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (story_id, line_number) REFERENCES story_lines (story_id, line_number) ON DELETE CASCADE
);

-- Index for efficient querying by label
CREATE INDEX IF NOT EXISTS idx_audio_files_label ON line_audio_files (story_id, label);

CREATE TABLE IF NOT EXISTS vocabulary_items (
    id SERIAL PRIMARY KEY,
    story_id INTEGER,
    line_number INTEGER,
    word TEXT NOT NULL,
    lexical_form TEXT NOT NULL,
    position_start INTEGER NOT NULL,
    position_end INTEGER NOT NULL,
    FOREIGN KEY (story_id, line_number) REFERENCES story_lines (story_id, line_number) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS grammar_items (
    id SERIAL PRIMARY KEY,
    story_id INTEGER,
    line_number INTEGER,
    grammar_point_id INTEGER REFERENCES grammar_points (grammar_point_id) ON DELETE SET NULL,
    text TEXT NOT NULL,
    position_start INTEGER NOT NULL,
    position_end INTEGER NOT NULL,
    FOREIGN KEY (story_id, line_number) REFERENCES story_lines (story_id, line_number) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS footnotes (
    id SERIAL PRIMARY KEY,
    story_id INTEGER,
    line_number INTEGER,
    footnote_text TEXT NOT NULL,
    FOREIGN KEY (story_id, line_number) REFERENCES story_lines (story_id, line_number) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS footnote_references (
    footnote_id INTEGER REFERENCES footnotes (id) ON DELETE CASCADE,
    reference TEXT NOT NULL,
    PRIMARY KEY (footnote_id, reference)
);

-- User time tracking table
CREATE TABLE IF NOT EXISTS user_time_tracking (
    tracking_id SERIAL PRIMARY KEY,
    user_id TEXT NOT NULL REFERENCES users (user_id) ON DELETE CASCADE,
    route TEXT NOT NULL,
    story_id INTEGER REFERENCES stories (story_id) ON DELETE SET NULL,
    started_at TIMESTAMP NOT NULL,
    ended_at TIMESTAMP,
    total_time_seconds INTEGER,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Index for efficient querying by user and date
CREATE INDEX IF NOT EXISTS idx_time_tracking_user_date ON user_time_tracking (user_id, started_at);

-- Anonymous user time tracking table (separate to prevent database bloat)
CREATE TABLE IF NOT EXISTS anonymous_time_tracking (
    tracking_id SERIAL PRIMARY KEY,
    session_id TEXT NOT NULL, -- Browser fingerprint or session identifier
    route TEXT NOT NULL,
    story_id INTEGER REFERENCES stories (story_id) ON DELETE SET NULL,
    started_at TIMESTAMP NOT NULL,
    ended_at TIMESTAMP,
    total_time_seconds INTEGER,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Index for efficient querying and cleanup
CREATE INDEX IF NOT EXISTS idx_anonymous_time_tracking_date ON anonymous_time_tracking (created_at);
CREATE INDEX IF NOT EXISTS idx_anonymous_time_tracking_session ON anonymous_time_tracking (session_id, started_at);

-- Vocabulary scores table
CREATE TABLE IF NOT EXISTS vocab_scores (
    score_id SERIAL PRIMARY KEY,
    user_id TEXT NOT NULL REFERENCES users (user_id) ON DELETE CASCADE,
    story_id INTEGER NOT NULL REFERENCES stories (story_id) ON DELETE CASCADE,
    line_number INTEGER NOT NULL,
    vocab_item_id INTEGER NOT NULL REFERENCES vocabulary_items (id) ON DELETE CASCADE,
    correct BOOLEAN NOT NULL,
    attempted_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (story_id, line_number) REFERENCES story_lines (story_id, line_number) ON DELETE CASCADE
);

-- Grammar scores table
CREATE TABLE IF NOT EXISTS grammar_scores (
    score_id SERIAL PRIMARY KEY,
    user_id TEXT NOT NULL REFERENCES users (user_id) ON DELETE CASCADE,
    story_id INTEGER NOT NULL REFERENCES stories (story_id) ON DELETE CASCADE,
    line_number INTEGER NOT NULL,
    grammar_item_id INTEGER NOT NULL REFERENCES grammar_items (id) ON DELETE CASCADE,
    correct BOOLEAN NOT NULL,
    attempted_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (story_id, line_number) REFERENCES story_lines (story_id, line_number) ON DELETE CASCADE
);

-- Indexes for efficient score querying
CREATE INDEX IF NOT EXISTS idx_vocab_scores_user_story ON vocab_scores (user_id, story_id);
CREATE INDEX IF NOT EXISTS idx_vocab_scores_story ON vocab_scores (story_id);
CREATE INDEX IF NOT EXISTS idx_vocab_scores_user ON vocab_scores (user_id);
CREATE INDEX IF NOT EXISTS idx_grammar_scores_user_story ON grammar_scores (user_id, story_id);
CREATE INDEX IF NOT EXISTS idx_grammar_scores_story ON grammar_scores (story_id);
CREATE INDEX IF NOT EXISTS idx_grammar_scores_user ON grammar_scores (user_id);
