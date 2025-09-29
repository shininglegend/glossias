-- Score management queries

-- name: SaveVocabScore :exec
INSERT INTO vocab_scores (user_id, story_id, line_number, vocab_item_id, correct)
VALUES ($1, $2, $3, $4, $5);

-- name: SaveGrammarScore :exec
INSERT INTO grammar_scores (user_id, story_id, line_number, grammar_point_id, correct)
VALUES ($1, $2, $3, $4, $5);

-- name: SaveVocabIncorrectAnswer :exec
INSERT INTO vocab_incorrect_answers (user_id, story_id, line_number, vocab_item_id, incorrect_answer)
VALUES ($1, $2, $3, $4, $5);

-- name: SaveGrammarIncorrectAnswer :exec
INSERT INTO grammar_incorrect_answers (user_id, story_id, line_number, grammar_point_id, selected_line, selected_positions)
VALUES ($1, $2, $3, $4, $5, $6);

-- name: GetUserVocabScores :many
SELECT vs.line_number, vs.vocab_item_id, vs.correct, vs.attempted_at, vi.word, vi.lexical_form
FROM vocab_scores vs
JOIN vocabulary_items vi ON vs.vocab_item_id = vi.id
WHERE vs.user_id = $1 AND vs.story_id = $2
ORDER BY vs.line_number, vs.attempted_at DESC;

-- name: GetUserGrammarScores :many
SELECT gs.line_number, gs.grammar_point_id, gs.correct, gs.attempted_at, gi.text, gp.name as grammar_point_name
FROM grammar_scores gs
JOIN grammar_points gp ON gs.grammar_point_id = gp.grammar_point_id
LEFT JOIN grammar_items gi ON gs.grammar_point_id = gi.grammar_point_id AND gs.story_id = gi.story_id AND gs.line_number = gi.line_number
WHERE gs.user_id = $1 AND gs.story_id = $2
ORDER BY gs.line_number, gs.attempted_at DESC;

-- name: GetStoryVocabScores :many
SELECT vs.user_id, vs.line_number, vs.vocab_item_id, vs.correct, vs.attempted_at,
       vi.word, vi.lexical_form, u.name as user_name, u.email
FROM vocab_scores vs
JOIN vocabulary_items vi ON vs.vocab_item_id = vi.id
JOIN users u ON vs.user_id = u.user_id
WHERE vs.story_id = $1
ORDER BY vs.line_number, vs.attempted_at DESC;

-- name: GetStoryGrammarScores :many
SELECT gs.user_id, gs.line_number, gs.grammar_point_id, gs.correct, gs.attempted_at,
       gi.text, gp.name as grammar_point_name, u.name as user_name, u.email
FROM grammar_scores gs
JOIN grammar_points gp ON gs.grammar_point_id = gp.grammar_point_id
LEFT JOIN grammar_items gi ON gs.grammar_point_id = gi.grammar_point_id AND gs.story_id = gi.story_id AND gs.line_number = gi.line_number
JOIN users u ON gs.user_id = u.user_id
WHERE gs.story_id = $1
ORDER BY gs.line_number, gs.attempted_at DESC;

-- name: GetUserStoryVocabSummary :one
SELECT
    COUNT(*) as total_attempts,
    COUNT(CASE WHEN vs.correct = true THEN 1 END) as correct_answers,
    COUNT(CASE WHEN vs.correct = false THEN 1 END) as incorrect_answers
FROM vocab_scores vs
WHERE vs.user_id = $1 AND vs.story_id = $2;

-- name: GetUserStoryGrammarSummary :one
SELECT
    COUNT(*) as total_attempts,
    COUNT(CASE WHEN gs.correct = true THEN 1 END) as correct_answers,
    COUNT(CASE WHEN gs.correct = false THEN 1 END) as incorrect_answers
FROM grammar_scores gs
WHERE gs.user_id = $1 AND gs.story_id = $2;

-- name: GetAllUsersStoryVocabSummary :many
SELECT
    vs.user_id,
    u.name as user_name,
    u.email,
    COUNT(*) as total_attempts,
    COUNT(CASE WHEN vs.correct = true THEN 1 END) as correct_answers,
    COUNT(CASE WHEN vs.correct = false THEN 1 END) as incorrect_answers
FROM vocab_scores vs
JOIN users u ON vs.user_id = u.user_id
WHERE vs.story_id = $1
GROUP BY vs.user_id, u.name, u.email
ORDER BY u.name;

-- name: GetAllUsersStoryGrammarSummary :many
SELECT
    gs.user_id,
    u.name as user_name,
    u.email,
    COUNT(*) as total_attempts,
    COUNT(CASE WHEN gs.correct = true THEN 1 END) as correct_answers,
    COUNT(CASE WHEN gs.correct = false THEN 1 END) as incorrect_answers
FROM grammar_scores gs
JOIN users u ON gs.user_id = u.user_id
WHERE gs.story_id = $1
GROUP BY gs.user_id, u.name, u.email
ORDER BY u.name;

-- name: GetUserLatestVocabScoresByLine :many
SELECT DISTINCT ON (vs.line_number, vs.vocab_item_id)
    vs.line_number,
    vs.vocab_item_id,
    vs.correct,
    vs.attempted_at,
    vi.word,
    vi.lexical_form
FROM vocab_scores vs
JOIN vocabulary_items vi ON vs.vocab_item_id = vi.id
WHERE vs.user_id = $1 AND vs.story_id = $2
ORDER BY vs.line_number, vs.vocab_item_id, vs.attempted_at DESC;

-- name: GetUserLatestGrammarScoresByLine :many
SELECT DISTINCT ON (gs.line_number, gs.grammar_point_id)
    gs.line_number,
    gs.grammar_point_id,
    gs.correct,
    gs.attempted_at,
    gi.text,
    gp.name as grammar_point_name
FROM grammar_scores gs
JOIN grammar_points gp ON gs.grammar_point_id = gp.grammar_point_id
LEFT JOIN grammar_items gi ON gs.grammar_point_id = gi.grammar_point_id AND gs.story_id = gi.story_id AND gs.line_number = gi.line_number
WHERE gs.user_id = $1 AND gs.story_id = $2
ORDER BY gs.line_number, gs.grammar_point_id, gs.attempted_at DESC;
