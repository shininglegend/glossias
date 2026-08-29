-- Score management queries

-- name: SaveGrammarScore :exec
INSERT INTO grammar_correct_answers (user_id, story_id, line_number, grammar_point_id)
VALUES ($1, $2, $3, $4);

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

-- name: GetUserGrammarScoresByGrammarPoint :many
SELECT gs.line_number, gs.grammar_point_id, gs.attempted_at
FROM grammar_correct_answers gs
WHERE gs.user_id = $1 AND gs.story_id = $2 AND gs.grammar_point_id = $3
ORDER BY gs.line_number, gs.attempted_at DESC;

-- name: GetUserGrammarIncorrectAnswers :many
SELECT gia.line_number, gia.grammar_point_id, gia.selected_line, gia.selected_positions, gia.attempted_at
FROM grammar_incorrect_answers gia
WHERE gia.user_id = $1 AND gia.story_id = $2 AND gia.grammar_point_id = $3
ORDER BY gia.line_number, gia.attempted_at DESC;

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
    (SELECT COUNT(*) FROM vocab_correct_answers vca WHERE vca.user_id = $1 AND vca.story_id = $2) as correct_count,
    (SELECT COUNT(*) FROM vocab_incorrect_answers via WHERE via.user_id = $1 AND via.story_id = $2) as incorrect_count;

-- name: GetUserStoryGrammarSummary :one
SELECT
    (SELECT COUNT(*) FROM grammar_correct_answers gca WHERE gca.user_id = $1 AND gca.story_id = $2) as correct_count,
    (SELECT COUNT(*) FROM grammar_incorrect_answers gia WHERE gia.user_id = $1 AND gia.story_id = $2) as incorrect_count;

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

-- name: CountStoryVocabItems :one
SELECT COUNT(*) as total_vocab_items
FROM vocabulary_items
WHERE story_id = $1;

-- name: CountStoryGrammarItems :one
SELECT COUNT(*) as total_grammar_items
FROM grammar_items
WHERE story_id = $1;

-- GetStoryPhaseTotals: how many scorable items each phase of a story has. The
-- denominators for CalculateScoreWithRetriesAllowed on the score page and the
-- admin report. identify_total counts target-word *occurrences* (one quiz
-- popup each), matching GetUserStoryPageCompletion.
-- name: GetStoryPhaseTotals :one
SELECT
    (SELECT COUNT(*) FROM vocabulary_items vi WHERE vi.story_id = @story_id::INT)::INT AS vocab_total,
    (SELECT COUNT(*) FROM grammar_items gi WHERE gi.story_id = @story_id)::INT AS grammar_total,
    (SELECT COUNT(*) FROM target_vocabulary tv
      JOIN vocabulary_items vi ON vi.story_id = tv.story_id AND vi.lexical_form = tv.lexical_form
      WHERE tv.story_id = @story_id)::INT AS identify_total,
    (SELECT COUNT(*) FROM produce_segments ps WHERE ps.story_id = @story_id)::INT AS produce_total,
    (SELECT COUNT(*) FROM recall_sentences rs WHERE rs.story_id = @story_id)::INT AS recall_total;

-- GetUserStoryScoreSummary: every per-user answer count the score page needs in
-- one round trip. Produce aggregates the latest submission per segment, like
-- GetUserStoryProduceSummary; ungraded segments are excluded from the average.
-- name: GetUserStoryScoreSummary :one
SELECT
    (SELECT COUNT(*) FROM vocab_correct_answers t WHERE t.user_id = @user_id AND t.story_id = @story_id::INT)::INT AS vocab_correct,
    (SELECT COUNT(*) FROM vocab_incorrect_answers t WHERE t.user_id = @user_id AND t.story_id = @story_id)::INT AS vocab_incorrect,
    (SELECT COUNT(*) FROM grammar_correct_answers t WHERE t.user_id = @user_id AND t.story_id = @story_id)::INT AS grammar_correct,
    (SELECT COUNT(*) FROM grammar_incorrect_answers t WHERE t.user_id = @user_id AND t.story_id = @story_id)::INT AS grammar_incorrect,
    (SELECT COUNT(*) FROM identify_correct_answers t WHERE t.user_id = @user_id AND t.story_id = @story_id)::INT AS identify_correct,
    (SELECT COUNT(*) FROM identify_incorrect_answers t WHERE t.user_id = @user_id AND t.story_id = @story_id)::INT AS identify_incorrect,
    (SELECT COUNT(*) FROM recall_correct_answers t WHERE t.user_id = @user_id AND t.story_id = @story_id)::INT AS recall_correct,
    (SELECT COUNT(*) FROM recall_incorrect_answers t WHERE t.user_id = @user_id AND t.story_id = @story_id)::INT AS recall_incorrect,
    COALESCE(latest.submitted, 0)::INT AS produce_submitted,
    COALESCE(latest.graded, 0)::INT AS produce_graded,
    COALESCE(latest.average_score, 0)::FLOAT8 AS produce_average_score
FROM (SELECT 1) AS one
LEFT JOIN (
    SELECT COUNT(*) AS submitted, COUNT(ai_score) AS graded, AVG(ai_score) AS average_score
    FROM (
        SELECT DISTINCT ON (segment_id) segment_id, ai_score
        FROM produce_submissions
        WHERE user_id = @user_id AND story_id = @story_id
        ORDER BY segment_id, created_at DESC
    ) l
) latest ON true;

-- GetStoryStudentPerformance: one row per enrolled student with their answer
-- counts, Produce grading state, and per-phase time for a story. Each answer
-- table is aggregated once with GROUP BY user_id and joined, instead of a
-- correlated scalar subquery per student per table.
-- name: GetStoryStudentPerformance :many
WITH vocab_c AS (
    SELECT user_id, COUNT(*) AS n FROM vocab_correct_answers WHERE story_id = @story_id::INT GROUP BY user_id
), vocab_i AS (
    SELECT user_id, COUNT(*) AS n FROM vocab_incorrect_answers WHERE story_id = @story_id GROUP BY user_id
), grammar_c AS (
    SELECT user_id, COUNT(*) AS n FROM grammar_correct_answers WHERE story_id = @story_id GROUP BY user_id
), grammar_i AS (
    SELECT user_id, COUNT(*) AS n FROM grammar_incorrect_answers WHERE story_id = @story_id GROUP BY user_id
), identify_c AS (
    SELECT user_id, COUNT(*) AS n FROM identify_correct_answers WHERE story_id = @story_id GROUP BY user_id
), identify_i AS (
    SELECT user_id, COUNT(*) AS n FROM identify_incorrect_answers WHERE story_id = @story_id GROUP BY user_id
), recall_c AS (
    SELECT user_id, COUNT(*) AS n FROM recall_correct_answers WHERE story_id = @story_id GROUP BY user_id
), recall_i AS (
    SELECT user_id, COUNT(*) AS n FROM recall_incorrect_answers WHERE story_id = @story_id GROUP BY user_id
), produce_latest AS (
    SELECT DISTINCT ON (user_id, segment_id) user_id, segment_id, ai_score
    FROM produce_submissions
    WHERE story_id = @story_id
    ORDER BY user_id, segment_id, created_at DESC
), produce_stats AS (
    SELECT user_id,
           COUNT(*) AS submitted,
           COUNT(ai_score) AS graded,
           AVG(ai_score) AS average_score
    FROM produce_latest
    GROUP BY user_id
), tr AS (
    SELECT user_id, requested_lines, (completed_at IS NOT NULL) AS completed
    FROM translation_requests
    WHERE story_id = @story_id
), time_stats AS (
    SELECT user_id,
        COALESCE(SUM(CASE WHEN route LIKE '%vocab%' THEN total_time_seconds END), 0)::INT AS vocab_time_seconds,
        COALESCE(SUM(CASE WHEN route LIKE '%grammar%' THEN total_time_seconds END), 0)::INT AS grammar_time_seconds,
        COALESCE(SUM(CASE WHEN route LIKE '%translate%' THEN total_time_seconds END), 0)::INT AS translation_time_seconds,
        COALESCE(SUM(CASE WHEN route LIKE '%audio%' OR route LIKE '%video%' THEN total_time_seconds END), 0)::INT AS video_time_seconds,
        COALESCE(SUM(CASE WHEN route LIKE '%identify%' THEN total_time_seconds END), 0)::INT AS identify_time_seconds,
        COALESCE(SUM(CASE WHEN route LIKE '%produce%' THEN total_time_seconds END), 0)::INT AS produce_time_seconds,
        COALESCE(SUM(CASE WHEN route LIKE '%recall%' THEN total_time_seconds END), 0)::INT AS recall_time_seconds,
        COALESCE(SUM(total_time_seconds), 0)::INT AS total_time_seconds
    FROM user_time_tracking
    WHERE story_id = @story_id AND ended_at IS NOT NULL
    GROUP BY user_id
)
SELECT
    u.user_id,
    u.name AS user_name,
    u.email,
    st.title AS story_title,
    COALESCE(vocab_c.n, 0)::INT AS vocab_correct,
    COALESCE(vocab_i.n, 0)::INT AS vocab_incorrect,
    COALESCE(grammar_c.n, 0)::INT AS grammar_correct,
    COALESCE(grammar_i.n, 0)::INT AS grammar_incorrect,
    COALESCE(identify_c.n, 0)::INT AS identify_correct,
    COALESCE(identify_i.n, 0)::INT AS identify_incorrect,
    COALESCE(recall_c.n, 0)::INT AS recall_correct,
    COALESCE(recall_i.n, 0)::INT AS recall_incorrect,
    COALESCE(produce_stats.submitted, 0)::INT AS produce_submitted,
    COALESCE(produce_stats.graded, 0)::INT AS produce_graded,
    COALESCE(produce_stats.average_score, 0)::FLOAT8 AS produce_average_score,
    COALESCE(tr.completed, false)::BOOLEAN AS translation_completed,
    COALESCE(tr.requested_lines, ARRAY[]::INTEGER[])::INTEGER[] AS requested_lines,
    COALESCE(time_stats.vocab_time_seconds, 0)::INT AS vocab_time_seconds,
    COALESCE(time_stats.grammar_time_seconds, 0)::INT AS grammar_time_seconds,
    COALESCE(time_stats.translation_time_seconds, 0)::INT AS translation_time_seconds,
    COALESCE(time_stats.video_time_seconds, 0)::INT AS video_time_seconds,
    COALESCE(time_stats.identify_time_seconds, 0)::INT AS identify_time_seconds,
    COALESCE(time_stats.produce_time_seconds, 0)::INT AS produce_time_seconds,
    COALESCE(time_stats.recall_time_seconds, 0)::INT AS recall_time_seconds,
    COALESCE(time_stats.total_time_seconds, 0)::INT AS total_time_seconds
FROM users u
JOIN course_users cu ON u.user_id = cu.user_id
JOIN stories s ON cu.course_id = s.course_id
LEFT JOIN story_titles st ON s.story_id = st.story_id AND st.language_code = 'en'
LEFT JOIN vocab_c ON vocab_c.user_id = u.user_id
LEFT JOIN vocab_i ON vocab_i.user_id = u.user_id
LEFT JOIN grammar_c ON grammar_c.user_id = u.user_id
LEFT JOIN grammar_i ON grammar_i.user_id = u.user_id
LEFT JOIN identify_c ON identify_c.user_id = u.user_id
LEFT JOIN identify_i ON identify_i.user_id = u.user_id
LEFT JOIN recall_c ON recall_c.user_id = u.user_id
LEFT JOIN recall_i ON recall_i.user_id = u.user_id
LEFT JOIN produce_stats ON produce_stats.user_id = u.user_id
LEFT JOIN tr ON tr.user_id = u.user_id
LEFT JOIN time_stats ON time_stats.user_id = u.user_id
WHERE s.story_id = @story_id
  AND (@status::TEXT = '' OR cu.status = @status)
ORDER BY u.name;
