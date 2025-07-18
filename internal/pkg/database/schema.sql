-- schema.sql
-- Remove SQLite-specific PRAGMA
-- Replace AUTOINCREMENT with SERIAL
CREATE TABLE IF NOT EXISTS stories (
    story_id SERIAL PRIMARY KEY,
    week_number INTEGER NOT NULL,
    day_letter TEXT NOT NULL,
    grammar_point TEXT,
    last_revision TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    author_id TEXT NOT NULL,
    author_name TEXT NOT NULL
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
    audio_file TEXT,
    PRIMARY KEY (story_id, line_number)
);

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
