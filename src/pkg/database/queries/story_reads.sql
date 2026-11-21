-- GetStoryFootnotesWithRefs returns every footnote of a story with its
-- references sorted; a footnote without references gets an empty array.
-- name: GetStoryFootnotesWithRefs :many
SELECT f.id, f.line_number, f.footnote_text,
       COALESCE(array_agg(fr.reference ORDER BY fr.reference) FILTER (WHERE fr.reference IS NOT NULL), '{}')::text[] AS refs
FROM footnotes f
LEFT JOIN footnote_references fr ON fr.footnote_id = f.id
WHERE f.story_id = sqlc.arg(story_id)::int
GROUP BY f.id, f.line_number, f.footnote_text
ORDER BY f.line_number, f.id;

-- GetLineFootnotesWithRefs is GetStoryFootnotesWithRefs for one line.
-- name: GetLineFootnotesWithRefs :many
SELECT f.id, f.footnote_text,
       COALESCE(array_agg(fr.reference ORDER BY fr.reference) FILTER (WHERE fr.reference IS NOT NULL), '{}')::text[] AS refs
FROM footnotes f
LEFT JOIN footnote_references fr ON fr.footnote_id = f.id
WHERE f.story_id = sqlc.arg(story_id)::int AND f.line_number = sqlc.arg(line_number)::int
GROUP BY f.id, f.footnote_text
ORDER BY f.id;

-- MergeTranslationRequestLines records revealed translation lines (1-based)
-- in one statement: it creates the user's request for the story or merges
-- the lines into it (sorted, no duplicates), and marks it complete when
-- complete is set. Completion is never cleared.
-- name: MergeTranslationRequestLines :exec
INSERT INTO translation_requests AS tr (user_id, story_id, requested_lines, completed_at)
VALUES (
    sqlc.arg(user_id)::text,
    sqlc.arg(story_id)::int,
    ARRAY(SELECT DISTINCT nl.n FROM unnest(sqlc.arg(lines)::int[]) AS nl(n) ORDER BY nl.n),
    CASE WHEN sqlc.arg(complete)::bool THEN CURRENT_TIMESTAMP END
)
ON CONFLICT (user_id, story_id) DO UPDATE
SET requested_lines = ARRAY(
        SELECT DISTINCT ml.n FROM unnest(tr.requested_lines || EXCLUDED.requested_lines) AS ml(n) ORDER BY ml.n
    ),
    completed_at = CASE WHEN sqlc.arg(complete)::bool THEN COALESCE(tr.completed_at, CURRENT_TIMESTAMP) ELSE tr.completed_at END;

-- SaveRecallAnswers logs one recall ordering attempt in one statement:
-- sentence_ids[i] was placed at position i (1-based); correct[i] picks the
-- table. Rows land in submitted order within each table.
-- name: SaveRecallAnswers :exec
WITH answers AS (
    SELECT a.sentence_id, a.ord::int AS position, (sqlc.arg(correct)::bool[])[a.ord] AS correct, a.ord
    FROM unnest(sqlc.arg(sentence_ids)::int[]) WITH ORDINALITY AS a(sentence_id, ord)
), correct_rows AS (
    INSERT INTO recall_correct_answers AS rca (user_id, story_id, recall_sentence_id)
    SELECT sqlc.arg(user_id)::text, sqlc.arg(story_id)::int, ac.sentence_id
    FROM answers ac
    WHERE ac.correct
    ORDER BY ac.ord
    RETURNING rca.id
)
INSERT INTO recall_incorrect_answers AS ria (user_id, story_id, recall_sentence_id, selected_position)
SELECT sqlc.arg(user_id)::text, sqlc.arg(story_id)::int, ai.sentence_id, ai.position
FROM answers ai
WHERE NOT ai.correct
ORDER BY ai.ord;
