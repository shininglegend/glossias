-- +goose Up
CREATE TABLE course_stories (
    course_id INTEGER NOT NULL REFERENCES courses (course_id) ON DELETE CASCADE,
    story_id INTEGER NOT NULL REFERENCES stories (story_id) ON DELETE CASCADE,
    PRIMARY KEY (course_id, story_id)
);

CREATE INDEX course_stories_story_id_idx ON course_stories (story_id);

INSERT INTO course_stories (course_id, story_id)
SELECT course_id, story_id
FROM stories
WHERE course_id IS NOT NULL;

-- +goose Down
DROP TABLE IF EXISTS course_stories;
