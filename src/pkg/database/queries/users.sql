-- User management queries

-- name: CreateUser :one
INSERT INTO users (user_id, email, name, is_super_admin)
VALUES ($1, $2, $3, $4)
RETURNING user_id, email, name, is_super_admin, created_at, updated_at;

-- name: GetUser :one
SELECT user_id, email, name, is_super_admin, created_at, updated_at
FROM users
WHERE user_id = $1;

-- name: GetUserByEmail :one
SELECT user_id, email, name, is_super_admin, created_at, updated_at
FROM users
WHERE email = $1;

-- name: GetUsersByEmails :many
SELECT user_id, email, name, is_super_admin, created_at, updated_at
FROM users
WHERE email = ANY($1::text[]);

-- name: UpdateUser :one
UPDATE users
SET email = $2, name = $3, is_super_admin = $4, updated_at = CURRENT_TIMESTAMP
WHERE user_id = $1
RETURNING user_id, email, name, is_super_admin, created_at, updated_at;

-- name: UpsertUser :one
INSERT INTO users (user_id, email, name, is_super_admin)
VALUES ($1, $2, $3, $4)
ON CONFLICT (user_id)
DO UPDATE SET
    email = EXCLUDED.email,
    name = EXCLUDED.name,
    updated_at = CURRENT_TIMESTAMP
RETURNING user_id, email, name, is_super_admin, created_at, updated_at;

-- name: DeleteUser :exec
DELETE FROM users WHERE user_id = $1;

-- name: ListUsers :many
SELECT user_id, email, name, is_super_admin, created_at, updated_at
FROM users
ORDER BY created_at DESC;

-- name: ListSuperAdmins :many
SELECT user_id, email, name, is_super_admin, created_at, updated_at
FROM users
WHERE is_super_admin = true
ORDER BY created_at DESC;
