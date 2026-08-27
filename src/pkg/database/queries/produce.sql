-- Produce phase queries: segments, explanation, and student submissions

-- name: GetStoryProduceSegments :many
SELECT ps.id, ps.story_id, ps.segment_order, ps.english_text, ps.reference_hebrew,
       ps.grammar_point_id, gp.name AS grammar_point_name
FROM produce_segments ps
LEFT JOIN grammar_points gp ON gp.grammar_point_id = ps.grammar_point_id
WHERE ps.story_id = $1
ORDER BY ps.segment_order;

-- name: GetProduceSegment :one
SELECT ps.id, ps.story_id, ps.segment_order, ps.english_text, ps.reference_hebrew,
       ps.grammar_point_id, gp.name AS grammar_point_name
FROM produce_segments ps
LEFT JOIN grammar_points gp ON gp.grammar_point_id = ps.grammar_point_id
WHERE ps.id = $1;

-- name: UpsertProduceSegment :one
INSERT INTO produce_segments (story_id, segment_order, english_text, reference_hebrew, grammar_point_id)
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (story_id, segment_order) DO UPDATE
SET english_text = EXCLUDED.english_text,
    reference_hebrew = EXCLUDED.reference_hebrew,
    grammar_point_id = EXCLUDED.grammar_point_id
RETURNING id, story_id, segment_order, english_text, reference_hebrew, grammar_point_id;

-- name: DeleteProduceSegment :exec
DELETE FROM produce_segments
WHERE id = $1;

-- name: DeleteStoryProduceSegments :exec
DELETE FROM produce_segments
WHERE story_id = $1;

-- name: GetStoryProduceExplanation :one
SELECT story_id, explanation_text
FROM story_produce_explanations
WHERE story_id = $1;

-- name: UpsertStoryProduceExplanation :one
INSERT INTO story_produce_explanations (story_id, explanation_text)
VALUES ($1, $2)
ON CONFLICT (story_id) DO UPDATE
SET explanation_text = EXCLUDED.explanation_text,
    updated_at = CURRENT_TIMESTAMP
RETURNING story_id, explanation_text;

-- name: DeleteStoryProduceExplanation :exec
DELETE FROM story_produce_explanations
WHERE story_id = $1;

-- name: CreateProduceSubmission :one
INSERT INTO produce_submissions (user_id, story_id, segment_id, student_text)
VALUES ($1, $2, $3, $4)
RETURNING id, user_id, story_id, segment_id, student_text, ai_score, ai_feedback, graded_at, created_at;

-- name: GradeProduceSubmission :exec
UPDATE produce_submissions
SET ai_score = $2,
    ai_feedback = $3,
    graded_at = CURRENT_TIMESTAMP
WHERE id = $1;

-- GetUserStoryProduceSubmissions returns the latest submission per segment,
-- which is what the Score page reports on.
-- name: GetUserStoryProduceSubmissions :many
SELECT DISTINCT ON (psub.segment_id)
    psub.id, psub.user_id, psub.story_id, psub.segment_id, psub.student_text,
    psub.ai_score, psub.ai_feedback, psub.graded_at, psub.created_at,
    ps.segment_order
FROM produce_submissions psub
JOIN produce_segments ps ON ps.id = psub.segment_id
WHERE psub.user_id = $1 AND psub.story_id = $2
ORDER BY psub.segment_id, psub.created_at DESC;

-- name: GetUserStoryProduceSummary :one
SELECT COUNT(*) AS segments_submitted,
       COUNT(ai_score) AS segments_graded,
       COALESCE(AVG(ai_score), 0)::FLOAT8 AS average_score
FROM (
    SELECT DISTINCT ON (segment_id) segment_id, ai_score
    FROM produce_submissions
    WHERE user_id = $1 AND story_id = $2
    ORDER BY segment_id, created_at DESC
) latest;

-- name: CountStoryProduceSegments :one
SELECT COUNT(*) AS total_segments
FROM produce_segments
WHERE story_id = $1;
