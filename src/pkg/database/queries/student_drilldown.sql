-- Per-student drill-down (T16): the actual answers and submissions behind one
-- row of the admin performance table — one student on one story. Read-only,
-- served by GET /api/admin/stories/{id}/students/{userId}.

-- name: GetStudentStoryHeader :one
SELECT u.user_id, u.name AS user_name, u.email,
       COALESCE(st.title, '')::TEXT AS story_title
FROM users u
LEFT JOIN story_titles st ON st.story_id = @story_id::INT AND st.language_code = 'en'
WHERE u.user_id = @user_id;

-- Every Identify pick, correct and incorrect, in the order the student made
-- them. selected_word is empty on correct rows (the pick was the target).
-- name: GetUserStoryIdentifyAnswerLog :many
SELECT ica.line_number, TRUE::BOOLEAN AS correct,
       tv.lexical_form::TEXT AS target_word, ''::TEXT AS selected_word,
       ica.attempted_at
FROM identify_correct_answers ica
JOIN target_vocabulary tv ON tv.id = ica.target_vocab_id
WHERE ica.user_id = @user_id AND ica.story_id = @story_id::INT
UNION ALL
SELECT iia.line_number, FALSE::BOOLEAN,
       tv.lexical_form::TEXT, sel.lexical_form::TEXT,
       iia.attempted_at
FROM identify_incorrect_answers iia
JOIN target_vocabulary tv ON tv.id = iia.target_vocab_id
JOIN target_vocabulary sel ON sel.id = iia.selected_target_vocab_id
WHERE iia.user_id = @user_id AND iia.story_id = @story_id
ORDER BY attempted_at;

-- Every Produce submission (not just the latest per segment, which is what the
-- performance table scores on), oldest first within each segment.
-- name: GetUserStoryProduceSubmissionHistory :many
SELECT ps.segment_order, psub.student_text, psub.ai_score, psub.ai_feedback,
       psub.graded_at, psub.created_at
FROM produce_submissions psub
JOIN produce_segments ps ON ps.id = psub.segment_id
WHERE psub.user_id = @user_id AND psub.story_id = @story_id::INT
ORDER BY ps.segment_order, psub.created_at;

-- Every Recall placement across all attempts. Correct rows placed the sentence
-- at its own sequence_order; the model layer groups rows back into attempts.
-- name: GetUserStoryRecallAnswerLog :many
SELECT rs.hebrew_text::TEXT AS hebrew_text,
       rs.sequence_order AS correct_position,
       rs.sequence_order AS selected_position,
       TRUE::BOOLEAN AS correct, rca.attempted_at
FROM recall_correct_answers rca
JOIN recall_sentences rs ON rs.id = rca.recall_sentence_id
WHERE rca.user_id = @user_id AND rca.story_id = @story_id::INT
UNION ALL
SELECT rs.hebrew_text::TEXT, rs.sequence_order, ria.selected_position,
       FALSE::BOOLEAN, ria.attempted_at
FROM recall_incorrect_answers ria
JOIN recall_sentences rs ON rs.id = ria.recall_sentence_id
WHERE ria.user_id = @user_id AND ria.story_id = @story_id
ORDER BY attempted_at, selected_position;
