-- name: BulkCreateVocabularyItems :copyfrom
INSERT INTO vocabulary_items (story_id, line_number, word, lexical_form, position_start, position_end)
VALUES ($1, $2, $3, $4, $5, $6);

-- name: BulkCreateGrammarItems :copyfrom
INSERT INTO grammar_items (story_id, line_number, text, position_start, position_end)
VALUES ($1, $2, $3, $4, $5);

-- name: BulkCreateStoryLines :copyfrom
INSERT INTO story_lines (story_id, line_number, text, audio_file)
VALUES ($1, $2, $3, $4);
