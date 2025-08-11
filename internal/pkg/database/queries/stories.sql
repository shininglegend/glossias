-- name: GetStory :one
SELECT s.story_id, s.week_number, s.day_letter, s.grammar_point, s.last_revision, s.author_id, s.author_name
FROM stories s
WHERE s.story_id = $1;

-- name: GetAllStories :many
SELECT s.story_id, s.week_number, s.day_letter, s.grammar_point, s.last_revision, s.author_id, s.author_name
FROM stories s
ORDER BY s.week_number, s.day_letter;

-- name: CreateStory :one
INSERT INTO stories (week_number, day_letter, grammar_point, author_id, author_name)
VALUES ($1, $2, $3, $4, $5)
RETURNING story_id, last_revision;

-- name: UpdateStory :exec
UPDATE stories
SET week_number = $2, day_letter = $3, grammar_point = $4, author_id = $5, author_name = $6, last_revision = CURRENT_TIMESTAMP
WHERE story_id = $1;

-- name: DeleteStory :exec
DELETE FROM stories WHERE story_id = $1;

-- name: GetStoryTitles :many
SELECT story_id, language_code, title
FROM story_titles
WHERE story_id = $1;

-- name: GetStoryTitle :one
SELECT title
FROM story_titles
WHERE story_id = $1 AND language_code = $2;

-- name: UpsertStoryTitle :exec
INSERT INTO story_titles (story_id, language_code, title)
VALUES ($1, $2, $3)
ON CONFLICT (story_id, language_code)
DO UPDATE SET title = EXCLUDED.title;

-- name: GetStoryDescription :one
SELECT description_text
FROM story_descriptions
WHERE story_id = $1 AND language_code = $2;

-- name: UpsertStoryDescription :exec
INSERT INTO story_descriptions (story_id, language_code, description_text)
VALUES ($1, $2, $3)
ON CONFLICT (story_id, language_code)
DO UPDATE SET description_text = EXCLUDED.description_text;

-- name: GetStoryLines :many
SELECT story_id, line_number, text, audio_file
FROM story_lines
WHERE story_id = $1
ORDER BY line_number;

-- name: GetStoryLine :one
SELECT story_id, line_number, text, audio_file
FROM story_lines
WHERE story_id = $1 AND line_number = $2;

-- name: UpsertStoryLine :exec
INSERT INTO story_lines (story_id, line_number, text, audio_file)
VALUES ($1, $2, $3, $4)
ON CONFLICT (story_id, line_number)
DO UPDATE SET text = EXCLUDED.text, audio_file = EXCLUDED.audio_file;

-- name: DeleteStoryLine :exec
DELETE FROM story_lines WHERE story_id = $1 AND line_number = $2;

-- name: GetVocabularyItems :many
SELECT id, story_id, line_number, word, lexical_form, position_start, position_end
FROM vocabulary_items
WHERE story_id = $1 AND line_number = $2
ORDER BY position_start;

-- name: CreateVocabularyItem :one
INSERT INTO vocabulary_items (story_id, line_number, word, lexical_form, position_start, position_end)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING id;

-- name: UpdateVocabularyItem :exec
UPDATE vocabulary_items
SET word = $2, lexical_form = $3, position_start = $4, position_end = $5
WHERE id = $1;

-- name: UpdateVocabularyByPosition :exec
UPDATE vocabulary_items
SET word = $5, lexical_form = $6, position_start = $7, position_end = $8
WHERE story_id = $1 AND line_number = $2 AND position_start = $3 AND position_end = $4;

-- name: UpdateVocabularyByWord :exec
UPDATE vocabulary_items
SET lexical_form = $4
WHERE story_id = $1 AND line_number = $2 AND word = $3;

-- name: DeleteVocabularyItems :exec
DELETE FROM vocabulary_items WHERE story_id = $1 AND line_number = $2;

-- name: GetGrammarItems :many
SELECT id, story_id, line_number, text, position_start, position_end
FROM grammar_items
WHERE story_id = $1 AND line_number = $2
ORDER BY position_start;

-- name: CreateGrammarItem :one
INSERT INTO grammar_items (story_id, line_number, text, position_start, position_end)
VALUES ($1, $2, $3, $4, $5)
RETURNING id;

-- name: UpdateGrammarByPosition :exec
UPDATE grammar_items
SET text = $5, position_start = $6, position_end = $7
WHERE story_id = $1 AND line_number = $2 AND position_start = $3 AND position_end = $4;

-- name: DeleteGrammarItems :exec
DELETE FROM grammar_items WHERE story_id = $1 AND line_number = $2;

-- name: GetFootnotes :many
SELECT f.id, f.story_id, f.line_number, f.footnote_text
FROM footnotes f
WHERE f.story_id = $1 AND f.line_number = $2;

-- name: CreateFootnote :one
INSERT INTO footnotes (story_id, line_number, footnote_text)
VALUES ($1, $2, $3)
RETURNING id;

-- name: UpdateFootnote :exec
UPDATE footnotes
SET footnote_text = $3
WHERE id = $1 AND story_id = $2;

-- name: GetFootnoteReferences :many
SELECT reference
FROM footnote_references
WHERE footnote_id = $1;

-- name: CreateFootnoteReference :exec
INSERT INTO footnote_references (footnote_id, reference)
VALUES ($1, $2);

-- name: DeleteFootnoteReferences :exec
DELETE FROM footnote_references WHERE footnote_id = $1;

-- name: DeleteVocabularyItem :exec
DELETE FROM vocabulary_items WHERE id = $1;

-- name: DeleteGrammarItem :exec
DELETE FROM grammar_items WHERE id = $1;

-- name: DeleteFootnote :exec
DELETE FROM footnotes WHERE id = $1;

-- name: DeleteAllStoryAnnotations :exec
DELETE FROM footnotes WHERE story_id = $1;

-- name: DeleteAllLineAnnotations :exec
DELETE FROM footnotes WHERE story_id = $1 AND line_number = $2;

-- name: DeleteStoryTitles :exec
DELETE FROM story_titles WHERE story_id = $1;

-- name: DeleteStoryDescriptions :exec
DELETE FROM story_descriptions WHERE story_id = $1;

-- name: DeleteAllStoryLines :exec
DELETE FROM story_lines WHERE story_id = $1;

-- name: StoryExists :one
SELECT EXISTS(SELECT 1 FROM stories WHERE story_id = $1);

-- name: LineExists :one
SELECT EXISTS(SELECT 1 FROM story_lines WHERE story_id = $1 AND line_number = $2);

-- name: GetAllVocabularyForStory :many
SELECT line_number, word, lexical_form, position_start, position_end
FROM vocabulary_items
WHERE story_id = $1
ORDER BY line_number, position_start;

-- name: GetAllGrammarForStory :many
SELECT line_number, text, position_start, position_end
FROM grammar_items
WHERE story_id = $1
ORDER BY line_number, position_start;

-- name: GetAllFootnotesForStory :many
SELECT f.line_number, f.id, f.footnote_text
FROM footnotes f
WHERE f.story_id = $1
ORDER BY f.line_number, f.id;

-- name: GetAllStoriesBasic :many
SELECT DISTINCT s.story_id, s.week_number, s.day_letter, st.title
FROM stories s
JOIN story_titles st ON s.story_id = st.story_id
WHERE st.language_code = $1 OR $1 = ''
ORDER BY s.week_number, s.day_letter;

-- name: BulkCreateVocabularyItems :copyfrom
INSERT INTO vocabulary_items (story_id, line_number, word, lexical_form, position_start, position_end)
VALUES ($1, $2, $3, $4, $5, $6);

-- name: BulkCreateGrammarItems :copyfrom
INSERT INTO grammar_items (story_id, line_number, text, position_start, position_end)
VALUES ($1, $2, $3, $4, $5);

-- name: BulkCreateStoryLines :copyfrom
INSERT INTO story_lines (story_id, line_number, text, audio_file)
VALUES ($1, $2, $3, $4);

-- name: CheckVocabularyExists :one
SELECT EXISTS(
    SELECT 1 FROM vocabulary_items
    WHERE story_id = $1 AND line_number = $2 AND word = $3 AND lexical_form = $4
    AND position_start = $5 AND position_end = $6
);

-- name: CheckGrammarExists :one
SELECT EXISTS(
    SELECT 1 FROM grammar_items
    WHERE story_id = $1 AND line_number = $2 AND text = $3
    AND position_start = $4 AND position_end = $5
);

-- name: CheckFootnoteExists :one
SELECT id FROM footnotes f
WHERE f.story_id = $1 AND f.line_number = $2 AND f.footnote_text = $3
LIMIT 1;

-- name: GetAllStoriesWithTitles :many
SELECT DISTINCT s.story_id, s.week_number, s.day_letter, st.title, st.language_code
FROM stories s
JOIN story_titles st ON s.story_id = st.story_id
ORDER BY s.week_number, s.day_letter;

-- name: GetStoryWithDescription :one
SELECT s.story_id, s.week_number, s.day_letter, s.grammar_point, s.last_revision, s.author_id, s.author_name,
       sd.language_code, sd.description_text
FROM stories s
LEFT JOIN story_descriptions sd ON s.story_id = sd.story_id
WHERE s.story_id = $1;

-- name: GetLineText :one
SELECT text FROM story_lines
WHERE story_id = $1 AND line_number = $2;

-- name: GetAllAnnotationsForStory :many
SELECT 'vocabulary' as type, v.line_number, v.word as text, v.lexical_form as extra, v.position_start, v.position_end, 0 as footnote_id
FROM vocabulary_items v
WHERE v.story_id = $1
UNION ALL
SELECT 'grammar' as type, g.line_number, g.text, '' as extra, g.position_start, g.position_end, 0 as footnote_id
FROM grammar_items g
WHERE g.story_id = $1
UNION ALL
SELECT 'footnote' as type, f.line_number, f.footnote_text as text, '' as extra, 0 as position_start, 0 as position_end, f.id as footnote_id
FROM footnotes f
WHERE f.story_id = $1
ORDER BY line_number, position_start;

-- name: UpdateStoryRevision :exec
UPDATE stories
SET last_revision = CURRENT_TIMESTAMP
WHERE story_id = $1;

-- name: DeleteAllVocabularyForStory :exec
DELETE FROM vocabulary_items WHERE story_id = $1;

-- name: DeleteAllGrammarForStory :exec
DELETE FROM grammar_items WHERE story_id = $1;

-- name: GetStoryFootnotesWithReferences :many
SELECT f.id, f.line_number, f.footnote_text, array_agg(fr.reference ORDER BY fr.reference) as references
FROM footnotes f
LEFT JOIN footnote_references fr ON f.id = fr.footnote_id
WHERE f.story_id = $1
GROUP BY f.id, f.line_number, f.footnote_text
ORDER BY f.line_number, f.id;
