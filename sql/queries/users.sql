-- name: CreateUserFull :one
INSERT INTO users (id, name, api_key, created_at, updated_at)
VALUES ($1, $2, $3, $3, $4)
RETURNING *;

-- name: CreateUser :one
INSERT INTO users (id, name, api_key)
VALUES ($1, $2, $3)
RETURNING *;

-- name: GetUserByApiKey :one
SELECT * FROM users WHERE api_key = $1;
