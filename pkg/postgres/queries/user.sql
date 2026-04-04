-- name: CreateUser :one
INSERT INTO users (id, name, email, created_at)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: ListUsers :many
SELECT * FROM users ORDER BY created_at ASC;

-- name: GetUserByID :one
SELECT * FROM users WHERE id = $1 LIMIT 1;

-- name: UpdateUser :one
UPDATE users SET name = $1, email = $2 WHERE id = $3
RETURNING *;

-- name: DeleteUser :execresult
DELETE FROM users WHERE id = $1;
