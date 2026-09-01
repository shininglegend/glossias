-- Target vocabulary queries (Identify / Recall phases)

-- name: GetStoryTargetVocabulary :many
SELECT id, story_id, lexical_form, audio_path, audio_bucket, correct_image_path, image_bucket
FROM target_vocabulary
WHERE story_id = $1
ORDER BY id;

-- name: GetTargetVocabulary :one
SELECT id, story_id, lexical_form, audio_path, audio_bucket, correct_image_path, image_bucket
FROM target_vocabulary
WHERE id = $1;

-- name: CreateTargetVocabulary :one
INSERT INTO target_vocabulary (story_id, lexical_form, audio_path, audio_bucket, correct_image_path, image_bucket)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING id, story_id, lexical_form, audio_path, audio_bucket, correct_image_path, image_bucket;

-- name: UpdateTargetVocabulary :one
UPDATE target_vocabulary
SET lexical_form = $2,
    audio_path = $3,
    audio_bucket = $4,
    correct_image_path = $5,
    image_bucket = $6
WHERE id = $1
RETURNING id, story_id, lexical_form, audio_path, audio_bucket, correct_image_path, image_bucket;

-- name: DeleteTargetVocabulary :exec
DELETE FROM target_vocabulary
WHERE id = $1;

-- name: DeleteStoryTargetVocabulary :exec
DELETE FROM target_vocabulary
WHERE story_id = $1;

-- name: CountStoryTargetVocabulary :one
SELECT COUNT(*) AS total_target_words
FROM target_vocabulary
WHERE story_id = $1;

-- GetTargetVocabularyOccurrences returns every place a target word appears in
-- the story text, by joining vocabulary_items on lexical_form. The Identify
-- phase uses this both to colour target words and to decide which lines to
-- pause on.
-- name: GetTargetVocabularyOccurrences :many
SELECT tv.id AS target_vocab_id,
       tv.lexical_form,
       vi.id AS vocab_item_id,
       vi.line_number,
       vi.word,
       vi.position_start,
       vi.position_end
FROM target_vocabulary tv
JOIN vocabulary_items vi
  ON vi.story_id = tv.story_id AND vi.lexical_form = tv.lexical_form
WHERE tv.story_id = $1
ORDER BY vi.line_number, vi.position_start;

-- GetStoryLexicalFormCounts lists every annotated lexical form in a story with
-- how many times it appears. The target-vocabulary editor uses it to show which
-- candidates meet the two-occurrence minimum before a word is chosen.
-- name: GetStoryLexicalFormCounts :many
SELECT lexical_form, COUNT(*) AS occurrences
FROM vocabulary_items
WHERE story_id = $1
GROUP BY lexical_form
ORDER BY lexical_form;
