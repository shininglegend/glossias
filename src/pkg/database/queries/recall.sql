-- Recall phase queries: sentences and answer logs

-- name: GetStoryRecallSentences :many
SELECT id, story_id, sequence_order, hebrew_text, target_vocab_id, image_path, image_bucket
FROM recall_sentences
WHERE story_id = $1
ORDER BY sequence_order;

-- name: GetRecallSentence :one
SELECT id, story_id, sequence_order, hebrew_text, target_vocab_id, image_path, image_bucket
FROM recall_sentences
WHERE id = $1;

-- name: UpsertRecallSentence :one
INSERT INTO recall_sentences (story_id, sequence_order, hebrew_text, target_vocab_id, image_path, image_bucket)
VALUES ($1, $2, $3, $4, $5, $6)
ON CONFLICT (story_id, sequence_order) DO UPDATE
SET hebrew_text = EXCLUDED.hebrew_text,
    target_vocab_id = EXCLUDED.target_vocab_id,
    image_path = EXCLUDED.image_path,
    image_bucket = EXCLUDED.image_bucket
RETURNING id, story_id, sequence_order, hebrew_text, target_vocab_id, image_path, image_bucket;

-- name: DeleteRecallSentence :exec
DELETE FROM recall_sentences
WHERE id = $1;

-- name: DeleteStoryRecallSentences :exec
DELETE FROM recall_sentences
WHERE story_id = $1;

-- name: CountStoryRecallSentences :one
SELECT COUNT(*) AS total_sentences
FROM recall_sentences
WHERE story_id = $1;

-- name: SaveRecallCorrectAnswer :exec
INSERT INTO recall_correct_answers (user_id, story_id, recall_sentence_id)
VALUES ($1, $2, $3);

-- name: SaveRecallIncorrectAnswer :exec
INSERT INTO recall_incorrect_answers (user_id, story_id, recall_sentence_id, selected_position)
VALUES ($1, $2, $3, $4);

-- name: GetUserStoryRecallSummary :one
SELECT
    (SELECT COUNT(*) FROM recall_correct_answers rca WHERE rca.user_id = $1 AND rca.story_id = $2) AS correct_count,
    (SELECT COUNT(*) FROM recall_incorrect_answers ria WHERE ria.user_id = $1 AND ria.story_id = $2) AS incorrect_count;

-- name: GetUserRecallCorrectAnswers :many
SELECT rca.recall_sentence_id, rca.attempted_at, rs.sequence_order, rs.hebrew_text
FROM recall_correct_answers rca
JOIN recall_sentences rs ON rs.id = rca.recall_sentence_id
WHERE rca.user_id = $1 AND rca.story_id = $2
ORDER BY rs.sequence_order, rca.attempted_at DESC;

-- name: GetAllUsersStoryRecallSummary :many
SELECT
    u.user_id,
    u.name AS user_name,
    u.email,
    COALESCE(stats.correct_count, 0) AS correct_answers,
    COALESCE(stats.incorrect_count, 0) AS incorrect_answers
FROM users u
JOIN LATERAL (
    SELECT
        (SELECT COUNT(*) FROM recall_correct_answers rca WHERE rca.user_id = u.user_id AND rca.story_id = $1) AS correct_count,
        (SELECT COUNT(*) FROM recall_incorrect_answers ria WHERE ria.user_id = u.user_id AND ria.story_id = $1) AS incorrect_count
) stats ON stats.correct_count > 0 OR stats.incorrect_count > 0
ORDER BY u.name;
