-- +goose Up
-- story_attempts now holds completed attempts only: a story is archived the
-- moment the student finishes it, and the in-progress attempt is implicit
-- (the live answer tables). Open rows came from the earlier explicit-redo
-- flow and carry no snapshot.
DELETE FROM story_attempts WHERE completed_at IS NULL;

ALTER TABLE story_attempts
    ALTER COLUMN completed_at SET DEFAULT CURRENT_TIMESTAMP,
    ALTER COLUMN completed_at SET NOT NULL;

-- +goose Down
ALTER TABLE story_attempts
    ALTER COLUMN completed_at DROP NOT NULL,
    ALTER COLUMN completed_at DROP DEFAULT;
