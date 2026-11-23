-- Set-based admin writes: each query replaces a per-row loop in models (save,
-- edit, translations, attempt renumbering). Payloads arrive as parallel,
-- equal-length arrays, unnested side by side in one SELECT list (set-returning
-- functions there advance in lockstep); generate_subscripts alongside gives
-- each element its 1-based payload position (ord), so results and child rows
-- can be tied back to it. sqlc only knows single-argument unnest, hence this
-- form rather than unnest(a, b) WITH ORDINALITY.

-- FindExistingVocabulary returns the 0-based payload indexes of vocabulary
-- items that already exist on the story, matching the same columns as the old
-- per-item CheckVocabularyExists.
-- name: FindExistingVocabulary :many
SELECT (p.ord - 1)::int AS idx
FROM (
    SELECT generate_subscripts(@words::text[], 1) AS ord,
           unnest(@line_numbers::int[]) AS line_number,
           unnest(@words::text[]) AS word,
           unnest(@lexical_forms::text[]) AS lexical_form,
           unnest(@position_starts::int[]) AS position_start,
           unnest(@position_ends::int[]) AS position_end
) AS p
WHERE EXISTS (
    SELECT 1 FROM vocabulary_items v
    WHERE v.story_id = @story_id::int
      AND v.line_number = p.line_number
      AND v.word = p.word
      AND v.lexical_form = p.lexical_form
      AND v.position_start = p.position_start
      AND v.position_end = p.position_end
)
ORDER BY p.ord;

-- FindExistingGrammar: as above for grammar items (grammar_point_id is not
-- part of the match, as before).
-- name: FindExistingGrammar :many
SELECT (p.ord - 1)::int AS idx
FROM (
    SELECT generate_subscripts(@texts::text[], 1) AS ord,
           unnest(@line_numbers::int[]) AS line_number,
           unnest(@texts::text[]) AS text,
           unnest(@position_starts::int[]) AS position_start,
           unnest(@position_ends::int[]) AS position_end
) AS p
WHERE EXISTS (
    SELECT 1 FROM grammar_items g
    WHERE g.story_id = @story_id::int
      AND g.line_number = p.line_number
      AND g.text = p.text
      AND g.position_start = p.position_start
      AND g.position_end = p.position_end
)
ORDER BY p.ord;

-- FindExistingFootnotes: footnotes match on line and text only.
-- name: FindExistingFootnotes :many
SELECT (p.ord - 1)::int AS idx
FROM (
    SELECT generate_subscripts(@footnote_texts::text[], 1) AS ord,
           unnest(@line_numbers::int[]) AS line_number,
           unnest(@footnote_texts::text[]) AS footnote_text
) AS p
WHERE EXISTS (
    SELECT 1 FROM footnotes f
    WHERE f.story_id = @story_id::int
      AND f.line_number = p.line_number
      AND f.footnote_text = p.footnote_text
)
ORDER BY p.ord;

-- InsertFootnotesWithRefs inserts a batch of footnotes and their references
-- in one statement. Ids are drawn from the footnotes sequence up front (the
-- MATERIALIZED CTE evaluates nextval exactly once per footnote, in payload
-- order), so references join to their footnote by payload position
-- (ref_footnote_ords holds the 1-based footnote position of each reference),
-- never by text: two footnotes with the same text stay distinct.
-- name: InsertFootnotesWithRefs :exec
WITH input AS MATERIALIZED (
    SELECT p.ord, p.line_number, p.footnote_text,
           nextval(pg_get_serial_sequence('footnotes', 'id'))::int AS id
    FROM (
        SELECT generate_subscripts(@footnote_texts::text[], 1) AS ord,
               unnest(@line_numbers::int[]) AS line_number,
               unnest(@footnote_texts::text[]) AS footnote_text
    ) AS p
    ORDER BY p.ord
), inserted AS (
    INSERT INTO footnotes (id, story_id, line_number, footnote_text)
    SELECT i.id, @story_id::int, i.line_number, i.footnote_text
    FROM input i
    ORDER BY i.ord
    RETURNING id
)
INSERT INTO footnote_references (footnote_id, reference)
SELECT i.id, r.reference
FROM (
    SELECT unnest(@ref_footnote_ords::int[]) AS footnote_ord,
           unnest(@reference_texts::text[]) AS reference
) AS r
JOIN input i ON i.ord = r.footnote_ord;

-- InsertFootnoteReferences attaches references to one existing footnote.
-- name: InsertFootnoteReferences :exec
INSERT INTO footnote_references (footnote_id, reference)
SELECT @footnote_id::int, unnest(@reference_texts::text[]);

-- name: UpsertStoryTitles :exec
INSERT INTO story_titles (story_id, language_code, title)
SELECT @story_id::int, t.language_code, t.title
FROM (
    SELECT unnest(@language_codes::text[]) AS language_code,
           unnest(@titles::text[]) AS title
) AS t
ON CONFLICT (story_id, language_code) DO UPDATE SET title = EXCLUDED.title;

-- UpsertLineTranslations writes one language's translations for many lines.
-- The caller must pass each line at most once: ON CONFLICT DO UPDATE cannot
-- touch the same row twice in one statement.
-- name: UpsertLineTranslations :exec
INSERT INTO line_translations (story_id, line_number, language_code, translation_text)
SELECT @story_id::int, t.line_number, @language_code::text, t.translation_text
FROM (
    SELECT unnest(@line_numbers::int[]) AS line_number,
           unnest(@translation_texts::text[]) AS translation_text
) AS t
ON CONFLICT (story_id, line_number, language_code)
DO UPDATE SET translation_text = EXCLUDED.translation_text;

-- InsertGrammarPoints inserts in payload order, so serial ids ascend with it.
-- An empty description is stored as NULL, as CreateGrammarPoint does.
-- name: InsertGrammarPoints :exec
INSERT INTO grammar_points (story_id, name, description)
SELECT @story_id::int, g.name, NULLIF(g.description, '')
FROM (
    SELECT generate_subscripts(@names::text[], 1) AS ord,
           unnest(@names::text[]) AS name,
           unnest(@descriptions::text[]) AS description
) AS g
ORDER BY g.ord;

-- TouchStoryRevision bumps last_revision; zero rows means no such story.
-- name: TouchStoryRevision :execrows
UPDATE stories SET last_revision = CURRENT_TIMESTAMP WHERE story_id = $1;

-- Attempt renumbering after deleting attempt N, in two set-based steps that
-- never trip the non-deferrable UNIQUE (user_id, story_id, attempt_number) or
-- CHECK (attempt_number >= 1):
--   1. offset: every attempt above N moves up by the current maximum M, so
--      the moved rows land above M, clear of every existing number;
--   2. settle: the moved rows take N, N+1, ... in their original order, all
--      below the still-offset values.
-- name: OffsetStoryAttemptsAfter :execrows
UPDATE story_attempts AS sa
SET attempt_number = sa.attempt_number + mx.max_number
FROM (
    SELECT MAX(s2.attempt_number)::int AS max_number
    FROM story_attempts s2
    WHERE s2.user_id = sqlc.arg(user_id)::text AND s2.story_id = @story_id::int
) AS mx
WHERE sa.user_id = sqlc.arg(user_id)::text AND sa.story_id = @story_id::int
  AND sa.attempt_number > sqlc.arg(deleted_number)::int;

-- name: SettleStoryAttemptsAfter :exec
UPDATE story_attempts AS sa
SET attempt_number = r.new_number
FROM (
    SELECT s2.attempt_id,
           (sqlc.arg(deleted_number)::int + ROW_NUMBER() OVER (ORDER BY s2.attempt_number) - 1)::int AS new_number
    FROM story_attempts s2
    WHERE s2.user_id = sqlc.arg(user_id)::text AND s2.story_id = @story_id::int
      AND s2.attempt_number > sqlc.arg(deleted_number)::int
) AS r
WHERE sa.attempt_id = r.attempt_id;

-- name: DeleteProduceSegmentByOrder :execrows
DELETE FROM produce_segments WHERE story_id = $1 AND segment_order = $2;
