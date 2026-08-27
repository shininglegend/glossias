-- Identify phase answer-log queries

-- name: SaveIdentifyCorrectAnswer :exec
INSERT INTO identify_correct_answers (user_id, story_id, line_number, target_vocab_id)
VALUES ($1, $2, $3, $4);

-- name: SaveIdentifyIncorrectAnswer :exec
INSERT INTO identify_incorrect_answers (user_id, story_id, line_number, target_vocab_id, selected_target_vocab_id)
VALUES ($1, $2, $3, $4, $5);

-- name: GetUserStoryIdentifySummary :one
SELECT
    (SELECT COUNT(*) FROM identify_correct_answers ica WHERE ica.user_id = $1 AND ica.story_id = $2) AS correct_count,
    (SELECT COUNT(*) FROM identify_incorrect_answers iia WHERE iia.user_id = $1 AND iia.story_id = $2) AS incorrect_count;

-- name: GetUserIdentifyCorrectAnswers :many
SELECT ica.line_number, ica.target_vocab_id, ica.attempted_at, tv.lexical_form
FROM identify_correct_answers ica
JOIN target_vocabulary tv ON tv.id = ica.target_vocab_id
WHERE ica.user_id = $1 AND ica.story_id = $2
ORDER BY ica.line_number, ica.attempted_at DESC;

-- GetUserIncompleteIdentifyTargets returns the target words the user has not
-- yet identified correctly, used to resume a partially finished Identify phase.
-- name: GetUserIncompleteIdentifyTargets :many
SELECT tv.id, tv.lexical_form
FROM target_vocabulary tv
WHERE tv.story_id = $1
  AND tv.id NOT IN (
      SELECT ica.target_vocab_id
      FROM identify_correct_answers ica
      WHERE ica.user_id = $2 AND ica.story_id = $1
  )
ORDER BY tv.id;

-- name: GetAllUsersStoryIdentifySummary :many
SELECT
    u.user_id,
    u.name AS user_name,
    u.email,
    COALESCE(stats.correct_count, 0) AS correct_answers,
    COALESCE(stats.incorrect_count, 0) AS incorrect_answers
FROM users u
JOIN LATERAL (
    SELECT
        (SELECT COUNT(*) FROM identify_correct_answers ica WHERE ica.user_id = u.user_id AND ica.story_id = $1) AS correct_count,
        (SELECT COUNT(*) FROM identify_incorrect_answers iia WHERE iia.user_id = u.user_id AND iia.story_id = $1) AS incorrect_count
) stats ON stats.correct_count > 0 OR stats.incorrect_count > 0
ORDER BY u.name;
