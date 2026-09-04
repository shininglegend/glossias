-- Per-student per-story progress resets. Every table here carries user_id and
-- story_id directly, so a reset is a plain two-column delete. Phase completion
-- is derived from these rows (see SUMMER_2026.md), so deleting them reopens the
-- phase for the student.

-- name: DeleteUserStoryVocabCorrect :execrows
DELETE FROM vocab_correct_answers WHERE user_id = $1 AND story_id = $2;

-- name: DeleteUserStoryVocabIncorrect :execrows
DELETE FROM vocab_incorrect_answers WHERE user_id = $1 AND story_id = $2;

-- name: DeleteUserStoryGrammarCorrect :execrows
DELETE FROM grammar_correct_answers WHERE user_id = $1 AND story_id = $2;

-- name: DeleteUserStoryGrammarIncorrect :execrows
DELETE FROM grammar_incorrect_answers WHERE user_id = $1 AND story_id = $2;

-- name: DeleteUserStoryTranslationRequest :execrows
DELETE FROM translation_requests WHERE user_id = $1 AND story_id = $2;

-- name: DeleteUserStoryIdentifyCorrect :execrows
DELETE FROM identify_correct_answers WHERE user_id = $1 AND story_id = $2;

-- name: DeleteUserStoryIdentifyIncorrect :execrows
DELETE FROM identify_incorrect_answers WHERE user_id = $1 AND story_id = $2;

-- name: DeleteUserStoryProduceSubmissions :execrows
DELETE FROM produce_submissions WHERE user_id = $1 AND story_id = $2;

-- name: DeleteUserStoryProduceAttemptStarts :execrows
DELETE FROM produce_attempt_starts WHERE user_id = $1 AND story_id = $2;

-- name: DeleteUserStoryRecallCorrect :execrows
DELETE FROM recall_correct_answers WHERE user_id = $1 AND story_id = $2;

-- name: DeleteUserStoryRecallIncorrect :execrows
DELETE FROM recall_incorrect_answers WHERE user_id = $1 AND story_id = $2;

-- name: DeleteUserStoryTimeTracking :execrows
DELETE FROM user_time_tracking WHERE user_id = $1 AND story_id = $2;

-- phase is the same value GetStoryStudentPerformance (scores.sql) buckets time
-- under, so what the admin sees zeroed matches what was deleted.
-- name: DeleteUserStoryTimeTrackingByPhase :execrows
DELETE FROM user_time_tracking WHERE user_id = $1 AND story_id = $2 AND phase = $3;

-- Whole-story reset in one round trip: clears every answer/submission table.
-- Time rows are deleted separately by DeleteUserStoryTimeTracking.
-- name: ResetUserStoryAnswers :one
WITH vc AS (DELETE FROM vocab_correct_answers AS t_vocab_correct_answers WHERE t_vocab_correct_answers.user_id = $1 AND t_vocab_correct_answers.story_id = $2 RETURNING 1),
     vi AS (DELETE FROM vocab_incorrect_answers AS t_vocab_incorrect_answers WHERE t_vocab_incorrect_answers.user_id = $1 AND t_vocab_incorrect_answers.story_id = $2 RETURNING 1),
     gc AS (DELETE FROM grammar_correct_answers AS t_grammar_correct_answers WHERE t_grammar_correct_answers.user_id = $1 AND t_grammar_correct_answers.story_id = $2 RETURNING 1),
     gi AS (DELETE FROM grammar_incorrect_answers AS t_grammar_incorrect_answers WHERE t_grammar_incorrect_answers.user_id = $1 AND t_grammar_incorrect_answers.story_id = $2 RETURNING 1),
     tr AS (DELETE FROM translation_requests AS t_translation_requests WHERE t_translation_requests.user_id = $1 AND t_translation_requests.story_id = $2 RETURNING 1),
     ic AS (DELETE FROM identify_correct_answers AS t_identify_correct_answers WHERE t_identify_correct_answers.user_id = $1 AND t_identify_correct_answers.story_id = $2 RETURNING 1),
     ii AS (DELETE FROM identify_incorrect_answers AS t_identify_incorrect_answers WHERE t_identify_incorrect_answers.user_id = $1 AND t_identify_incorrect_answers.story_id = $2 RETURNING 1),
     ps AS (DELETE FROM produce_submissions AS t_produce_submissions WHERE t_produce_submissions.user_id = $1 AND t_produce_submissions.story_id = $2 RETURNING 1),
     pa AS (DELETE FROM produce_attempt_starts AS t_produce_attempt_starts WHERE t_produce_attempt_starts.user_id = $1 AND t_produce_attempt_starts.story_id = $2 RETURNING 1),
     rc AS (DELETE FROM recall_correct_answers AS t_recall_correct_answers WHERE t_recall_correct_answers.user_id = $1 AND t_recall_correct_answers.story_id = $2 RETURNING 1),
     ri AS (DELETE FROM recall_incorrect_answers AS t_recall_incorrect_answers WHERE t_recall_incorrect_answers.user_id = $1 AND t_recall_incorrect_answers.story_id = $2 RETURNING 1)
SELECT
    (SELECT COUNT(*) FROM vc)::bigint AS vocab_correct,
    (SELECT COUNT(*) FROM vi)::bigint AS vocab_incorrect,
    (SELECT COUNT(*) FROM gc)::bigint AS grammar_correct,
    (SELECT COUNT(*) FROM gi)::bigint AS grammar_incorrect,
    (SELECT COUNT(*) FROM tr)::bigint AS translation_requests,
    (SELECT COUNT(*) FROM ic)::bigint AS identify_correct,
    (SELECT COUNT(*) FROM ii)::bigint AS identify_incorrect,
    (SELECT COUNT(*) FROM ps)::bigint AS produce_submissions,
    (SELECT COUNT(*) FROM pa)::bigint AS produce_attempt_starts,
    (SELECT COUNT(*) FROM rc)::bigint AS recall_correct,
    (SELECT COUNT(*) FROM ri)::bigint AS recall_incorrect;
