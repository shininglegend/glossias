-- Navigation queries: everything the "next page" decision needs in one round trip.

-- GetUserStoryPageCompletion returns, for one user and story, the authored total
-- and the user's progress for every skippable phase of the Summer 2026 flow:
-- Identify, Translate, Produce, Recall. Completion rules (e.g. "no segments
-- means produce is done") live in models.PageCompletion.
-- name: GetUserStoryPageCompletion :one
SELECT
    (SELECT COUNT(*) FROM target_vocabulary tv
      JOIN vocabulary_items vi ON vi.story_id = tv.story_id AND vi.lexical_form = tv.lexical_form
      WHERE tv.story_id = @story_id::INT)::INT AS identify_total,
    (SELECT COUNT(*) FROM target_vocabulary tv
      JOIN vocabulary_items vi ON vi.story_id = tv.story_id AND vi.lexical_form = tv.lexical_form
      WHERE tv.story_id = @story_id
        AND EXISTS (SELECT 1 FROM identify_correct_answers ica
                     WHERE ica.user_id = @user_id AND ica.story_id = tv.story_id
                       AND ica.line_number = vi.line_number AND ica.target_vocab_id = tv.id))::INT AS identify_correct,
    EXISTS (SELECT 1 FROM translation_requests tr
             WHERE tr.user_id = @user_id AND tr.story_id = @story_id AND tr.completed_at IS NOT NULL) AS translation_completed,
    (SELECT COUNT(*) FROM recall_sentences rs
      WHERE rs.story_id = @story_id)::INT AS recall_total,
    (SELECT COUNT(*) FROM recall_sentences rs
      WHERE rs.story_id = @story_id
        AND EXISTS (SELECT 1 FROM recall_correct_answers rca
                     WHERE rca.user_id = @user_id AND rca.story_id = rs.story_id
                       AND rca.recall_sentence_id = rs.id))::INT AS recall_correct,
    (SELECT COUNT(*) FROM produce_segments ps
      WHERE ps.story_id = @story_id)::INT AS produce_total,
    (SELECT COUNT(*) FROM produce_segments ps
      WHERE ps.story_id = @story_id
        AND EXISTS (SELECT 1 FROM produce_submissions psub
                     WHERE psub.user_id = @user_id AND psub.story_id = ps.story_id
                       AND psub.segment_id = ps.id))::INT AS produce_submitted;
