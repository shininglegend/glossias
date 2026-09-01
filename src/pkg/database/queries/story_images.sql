-- Story images management queries

-- name: CreateStoryImage :one
INSERT INTO story_images (story_id, file_path, file_bucket, label)
VALUES ($1, $2, $3, $4)
RETURNING image_id, story_id, file_path, file_bucket, label, created_at;

-- name: GetStoryImage :one
SELECT image_id, story_id, file_path, file_bucket, label, created_at
FROM story_images
WHERE image_id = $1;

-- name: GetStoryImagesByLabel :many
SELECT si.image_id, si.story_id, si.file_path, si.file_bucket, si.label, si.created_at
FROM story_images si
WHERE si.story_id = $1 AND si.label = $2
ORDER BY si.created_at;

-- name: GetAllStoryImages :many
SELECT image_id, story_id, file_path, file_bucket, label, created_at
FROM story_images
WHERE story_id = $1
ORDER BY label, created_at;

-- name: UpdateStoryImage :one
UPDATE story_images
SET file_path = $3, file_bucket = $4, label = $5
WHERE image_id = $1 AND story_id = $2
RETURNING image_id, story_id, file_path, file_bucket, label, created_at;

-- name: DeleteStoryImage :exec
DELETE FROM story_images
WHERE image_id = $1;

-- name: DeleteStoryImages :exec
DELETE FROM story_images
WHERE story_id = $1;

-- name: DeleteStoryImagesByLabel :exec
DELETE FROM story_images
WHERE story_id = $1 AND label = $2;

-- name: GetStoryImagesByLabelGlobally :many
SELECT image_id, story_id, file_path, file_bucket, label, created_at
FROM story_images
WHERE label = $1
ORDER BY story_id;
