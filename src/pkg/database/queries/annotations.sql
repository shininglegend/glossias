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

-- name: DeleteVocabularyItem :exec
DELETE FROM vocabulary_items WHERE id = $1;

-- name: GetAllVocabularyForStory :many
SELECT line_number, word, lexical_form, position_start, position_end
FROM vocabulary_items
WHERE story_id = $1
ORDER BY line_number, position_start;

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

-- name: DeleteGrammarItem :exec
DELETE FROM grammar_items WHERE id = $1;

-- name: GetAllGrammarForStory :many
SELECT line_number, text, position_start, position_end
FROM grammar_items
WHERE story_id = $1
ORDER BY line_number, position_start;

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

-- name: DeleteFootnote :exec
DELETE FROM footnotes WHERE id = $1;

-- name: GetAllFootnotesForStory :many
SELECT f.line_number, f.id, f.footnote_text
FROM footnotes f
WHERE f.story_id = $1
ORDER BY f.line_number, f.id;

-- name: GetFootnoteReferences :many
SELECT reference
FROM footnote_references
WHERE footnote_id = $1;

-- name: CreateFootnoteReference :exec
INSERT INTO footnote_references (footnote_id, reference)
VALUES ($1, $2);

-- name: DeleteFootnoteReferences :exec
DELETE FROM footnote_references WHERE footnote_id = $1;

-- name: GetStoryFootnotesWithReferences :many
SELECT f.id, f.line_number, f.footnote_text, array_agg(fr.reference ORDER BY fr.reference) as references
FROM footnotes f
LEFT JOIN footnote_references fr ON f.id = fr.footnote_id
WHERE f.story_id = $1
GROUP BY f.id, f.line_number, f.footnote_text
ORDER BY f.line_number, f.id;

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
