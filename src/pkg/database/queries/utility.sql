-- name: StoryExists :one
SELECT EXISTS(SELECT 1 FROM stories WHERE story_id = $1);

-- name: LineExists :one
SELECT EXISTS(SELECT 1 FROM story_lines WHERE story_id = $1 AND line_number = $2);

-- name: UpdateStoryRevision :exec
UPDATE stories
SET last_revision = CURRENT_TIMESTAMP
WHERE story_id = $1;

-- name: DeleteAllStoryAnnotations :exec
DELETE FROM footnotes WHERE story_id = $1;

-- name: DeleteAllLineAnnotations :exec
DELETE FROM footnotes WHERE story_id = $1 AND line_number = $2;

-- name: DeleteLineVocabulary :exec
DELETE FROM vocabulary_items WHERE story_id = $1 AND line_number = $2;

-- name: DeleteLineGrammar :exec
DELETE FROM grammar_items WHERE story_id = $1 AND line_number = $2;

-- name: DeleteLineFootnoteReferences :exec
DELETE FROM footnote_references
WHERE footnote_id IN (
    SELECT id FROM footnotes WHERE story_id = $1 AND line_number = $2
);

-- name: DeleteFootnoteReferencesByStory :exec
DELETE FROM footnote_references
WHERE footnote_id IN (
    SELECT id FROM footnotes WHERE story_id = $1
);

-- name: DeleteAllVocabularyForStory :exec
DELETE FROM vocabulary_items WHERE story_id = $1;

-- name: DeleteAllGrammarForStory :exec
DELETE FROM grammar_items WHERE story_id = $1;

-- name: DeleteAllTranslationsForStory :exec
DELETE FROM line_translations WHERE story_id = $1;

-- name: DeleteLineTranslations :exec
DELETE FROM line_translations WHERE story_id = $1 AND line_number = $2;
