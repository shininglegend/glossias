-- Vocabulary-related queries

-- name: CheckAllVocabCompleteForLineForUser :one
SELECT NOT EXISTS (
    SELECT 1
    FROM vocabulary_items vi
    WHERE vi.story_id = $1
      AND vi.line_number = $2
      AND vi.id NOT IN (
          SELECT vca.vocab_item_id
          FROM vocab_correct_answers vca
          WHERE vca.user_id = $3 AND vca.story_id = $1
      )
) as all_complete;

-- name: SaveVocabIncorrectAnswer :exec
INSERT INTO vocab_incorrect_answers (user_id, story_id, line_number, vocab_item_id, incorrect_answer)
VALUES ($1, $2, $3, $4, $5);

-- name: SaveVocabScore :exec
INSERT INTO vocab_correct_answers (user_id, story_id, line_number, vocab_item_id)
VALUES ($1, $2, $3, $4);

-- name: GetIncompleteVocabForUser :many
SELECT vi.line_number, vi.position_start
FROM vocabulary_items vi
WHERE vi.story_id = $1
  AND vi.id NOT IN (
      SELECT vca.vocab_item_id
      FROM vocab_correct_answers vca
      WHERE vca.user_id = $2 AND vca.story_id = $1
  )
ORDER BY vi.line_number, vi.position_start;
