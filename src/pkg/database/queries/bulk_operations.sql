-- name: BulkCreateVocabularyItems :copyfrom
INSERT INTO vocabulary_items (story_id, line_number, word, lexical_form, position_start, position_end)
VALUES ($1, $2, $3, $4, $5, $6);

-- name: BulkCreateGrammarItems :copyfrom
INSERT INTO grammar_items (story_id, line_number, grammar_point_id, text, position_start, position_end)
VALUES ($1, $2, $3, $4, $5, $6);

-- name: BulkCreateStoryLines :copyfrom
INSERT INTO story_lines (story_id, line_number, text)
VALUES ($1, $2, $3);

-- name: BulkCreateAudioFiles :copyfrom
INSERT INTO line_audio_files (story_id, line_number, file_path, file_bucket, label)
VALUES ($1, $2, $3, $4, $5);



-- name: BulkCreateLineTranslations :copyfrom
INSERT INTO line_translations (story_id, line_number, language_code, translation_text)
VALUES ($1, $2, $3, $4);
