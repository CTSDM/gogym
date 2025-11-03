-- name: CreateUser :one
INSERT INTO users (id, username, country, hashed_password, birthday)
VALUES (
    gen_random_uuid(), $1, $2, $3, $4
)
RETURNING *;

-- name: GetUser :one
SELECT * FROM users
WHERE id = $1;

-- name: GetUsers :many
SELECT * FROM users;

-- name: GetUserByUsername :one
SELECT * FROM users
WHERE username = $1;
