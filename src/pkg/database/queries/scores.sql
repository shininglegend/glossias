-- Score management queries

-- name: SaveVocabScore :exec
INSERT INTO vocab_correct_answers (user_id, story_id, line_number, vocab_item_id)
VALUES ($1, $2, $3, $4);

-- name: SaveGrammarScore :exec
INSERT INTO grammar_correct_answers (user_id, story_id, line_number, grammar_point_id)
VALUES ($1, $2, $3, $4);

-- name: SaveVocabIncorrectAnswer :exec
INSERT INTO vocab_incorrect_answers (user_id, story_id, line_number, vocab_item_id, incorrect_answer)
VALUES ($1, $2, $3, $4, $5);

-- name: SaveGrammarIncorrectAnswer :exec
INSERT INTO grammar_incorrect_answers (user_id, story_id, line_number, grammar_point_id, selected_line, selected_positions)
VALUES ($1, $2, $3, $4, $5, $6);

-- name: GetUserVocabScores :many
SELECT vs.line_number, vs.vocab_item_id, vs.attempted_at, vi.word, vi.lexical_form
FROM vocab_correct_answers vs
JOIN vocabulary_items vi ON vs.vocab_item_id = vi.id
WHERE vs.user_id = $1 AND vs.story_id = $2
ORDER BY vs.line_number, vs.attempted_at DESC;

-- name: GetUserGrammarScores :many
SELECT gs.line_number, gs.grammar_point_id, gs.attempted_at, gi.text, gp.name as grammar_point_name
FROM grammar_correct_answers gs
JOIN grammar_points gp ON gs.grammar_point_id = gp.grammar_point_id
LEFT JOIN grammar_items gi ON gs.grammar_point_id = gi.grammar_point_id AND gs.story_id = gi.story_id AND gs.line_number = gi.line_number
WHERE gs.user_id = $1 AND gs.story_id = $2
ORDER BY gs.line_number, gs.attempted_at DESC;

-- name: GetStoryVocabScores :many
SELECT vs.user_id, vs.line_number, vs.vocab_item_id, vs.attempted_at,
       vi.word, vi.lexical_form, u.name as user_name, u.email
FROM vocab_correct_answers vs
JOIN vocabulary_items vi ON vs.vocab_item_id = vi.id
JOIN users u ON vs.user_id = u.user_id
WHERE vs.story_id = $1
ORDER BY vs.line_number, vs.attempted_at DESC;

-- name: GetStoryGrammarScores :many
SELECT gs.user_id, gs.line_number, gs.grammar_point_id, gs.attempted_at,
       gi.text, gp.name as grammar_point_name, u.name as user_name, u.email
FROM grammar_correct_answers gs
JOIN grammar_points gp ON gs.grammar_point_id = gp.grammar_point_id
LEFT JOIN grammar_items gi ON gs.grammar_point_id = gi.grammar_point_id AND gs.story_id = gi.story_id AND gs.line_number = gi.line_number
JOIN users u ON gs.user_id = u.user_id
WHERE gs.story_id = $1
ORDER BY gs.line_number, gs.attempted_at DESC;

-- name: GetUserStoryVocabSummary :one
SELECT
    COUNT(vca.vocab_item_id) as correct_count,
    COUNT(via.vocab_item_id) as incorrect_count
FROM vocab_correct_answers vca
FULL OUTER JOIN vocab_incorrect_answers via ON vca.user_id = via.user_id AND vca.story_id = via.story_id AND vca.vocab_item_id = via.vocab_item_id
WHERE COALESCE(vca.user_id, via.user_id) = $1 AND COALESCE(vca.story_id, via.story_id) = $2;

-- name: GetUserStoryGrammarSummary :one
SELECT
    COUNT(gca.grammar_point_id) as correct_count,
    COUNT(gia.grammar_point_id) as incorrect_count
FROM grammar_correct_answers gca
FULL OUTER JOIN grammar_incorrect_answers gia ON gca.user_id = gia.user_id AND gca.story_id = gia.story_id AND gca.grammar_point_id = gia.grammar_point_id
WHERE COALESCE(gca.user_id, gia.user_id) = $1 AND COALESCE(gca.story_id, gia.story_id) = $2;

-- name: GetAllUsersStoryVocabSummary :many
SELECT
    COALESCE(vca.user_id, via.user_id) as user_id,
    u.name as user_name,
    u.email,
    COUNT(vca.vocab_item_id) as correct_answers,
    COUNT(via.vocab_item_id) as incorrect_answers
FROM vocab_correct_answers vca
FULL OUTER JOIN vocab_incorrect_answers via ON vca.user_id = via.user_id AND vca.story_id = via.story_id
JOIN users u ON COALESCE(vca.user_id, via.user_id) = u.user_id
WHERE COALESCE(vca.story_id, via.story_id) = $1
GROUP BY COALESCE(vca.user_id, via.user_id), u.name, u.email
ORDER BY u.name;

-- name: GetAllUsersStoryGrammarSummary :many
SELECT
    COALESCE(gca.user_id, gia.user_id) as user_id,
    u.name as user_name,
    u.email,
    COUNT(gca.grammar_point_id) as correct_answers,
    COUNT(gia.grammar_point_id) as incorrect_answers
FROM grammar_correct_answers gca
FULL OUTER JOIN grammar_incorrect_answers gia ON gca.user_id = gia.user_id AND gca.story_id = gia.story_id
JOIN users u ON COALESCE(gca.user_id, gia.user_id) = u.user_id
WHERE COALESCE(gca.story_id, gia.story_id) = $1
GROUP BY COALESCE(gca.user_id, gia.user_id), u.name, u.email
ORDER BY u.name;

-- name: GetUserLatestVocabScoresByLine :many
SELECT DISTINCT ON (vs.line_number, vs.vocab_item_id)
    vs.line_number,
    vs.vocab_item_id,
    vs.attempted_at,
    vi.word,
    vi.lexical_form
FROM vocab_correct_answers vs
JOIN vocabulary_items vi ON vs.vocab_item_id = vi.id
WHERE vs.user_id = $1 AND vs.story_id = $2
ORDER BY vs.line_number, vs.vocab_item_id, vs.attempted_at DESC;

-- name: GetUserLatestGrammarScoresByLine :many
SELECT DISTINCT ON (gs.line_number, gs.grammar_point_id)
    gs.line_number,
    gs.grammar_point_id,
    gs.attempted_at,
    gi.text,
    gp.name as grammar_point_name
FROM grammar_correct_answers gs
JOIN grammar_points gp ON gs.grammar_point_id = gp.grammar_point_id
LEFT JOIN grammar_items gi ON gs.grammar_point_id = gi.grammar_point_id AND gs.story_id = gi.story_id AND gs.line_number = gi.line_number
WHERE gs.user_id = $1 AND gs.story_id = $2
ORDER BY gs.line_number, gs.grammar_point_id, gs.attempted_at DESC;
