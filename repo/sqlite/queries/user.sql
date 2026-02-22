-- name: CreateUser :one
INSERT INTO users (id, name, email, created_at)
VALUES (?, ?, ?, ?)
RETURNING *;

-- name: ListUsers :many
SELECT * FROM users ORDER BY created_at ASC;

-- name: GetUserByID :one
SELECT * FROM users WHERE id = ? LIMIT 1;

-- name: UpdateUser :one
UPDATE users SET name = ?, email = ? WHERE id = ?
RETURNING *;

-- name: DeleteUser :execresult
DELETE FROM users WHERE id = ?;
