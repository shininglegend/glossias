-- +goose Up
CREATE TABLE course_sections (
    parent_course_id INTEGER NOT NULL REFERENCES courses (course_id) ON DELETE CASCADE,
    section_course_id INTEGER NOT NULL REFERENCES courses (course_id) ON DELETE CASCADE,
    PRIMARY KEY (section_course_id),
    CHECK (parent_course_id <> section_course_id)
);

CREATE INDEX course_sections_parent_course_id_idx ON course_sections (parent_course_id);

-- +goose Down
DROP TABLE IF EXISTS course_sections;
