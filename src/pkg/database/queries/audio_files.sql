-- Audio files management queries

-- name: CreateAudioFile :one
INSERT INTO line_audio_files (story_id, line_number, file_path, file_bucket, label)
VALUES ($1, $2, $3, $4, $5)
RETURNING audio_file_id, story_id, line_number, file_path, file_bucket, label, created_at;

-- name: GetAudioFile :one
SELECT audio_file_id, story_id, line_number, file_path, file_bucket, label, created_at
FROM line_audio_files
WHERE audio_file_id = $1;

-- name: GetLineAudioFiles :many
SELECT audio_file_id, story_id, line_number, file_path, file_bucket, label, created_at
FROM line_audio_files
WHERE story_id = $1 AND line_number = $2
ORDER BY label, created_at;

-- name: GetStoryAudioFilesByLabel :many
SELECT laf.audio_file_id, laf.story_id, laf.line_number, laf.file_path, laf.file_bucket, laf.label, laf.created_at
FROM line_audio_files laf
WHERE laf.story_id = $1 AND laf.label = $2
ORDER BY laf.line_number;

-- name: GetAllStoryAudioFiles :many
SELECT audio_file_id, story_id, line_number, file_path, file_bucket, label, created_at
FROM line_audio_files
WHERE story_id = $1
ORDER BY line_number, label, created_at;

-- name: UpdateAudioFile :one
UPDATE line_audio_files
SET file_path = $3, file_bucket = $4, label = $5
WHERE audio_file_id = $1 AND story_id = $2
RETURNING audio_file_id, story_id, line_number, file_path, file_bucket, label, created_at;

-- name: DeleteAudioFile :exec
DELETE FROM line_audio_files
WHERE audio_file_id = $1;

-- name: DeleteLineAudioFiles :exec
DELETE FROM line_audio_files
WHERE story_id = $1 AND line_number = $2;

-- name: DeleteStoryAudioFiles :exec
DELETE FROM line_audio_files
WHERE story_id = $1;

-- name: DeleteStoryAudioFilesByLabel :exec
DELETE FROM line_audio_files
WHERE story_id = $1 AND label = $2;

-- name: GetAudioFilesByLabel :many
SELECT audio_file_id, story_id, line_number, file_path, file_bucket, label, created_at
FROM line_audio_files
WHERE label = $1
ORDER BY story_id, line_number;
