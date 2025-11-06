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

-- name: DeleteUser :one
DELETE FROM users
WHERE id = $1
RETURNING *;

-- name: GetUserByUsername :one
SELECT * FROM users
WHERE username = $1;

-- name: CreateAdmin :one
INSERT INTO users (id, username, is_admin, country, hashed_password, birthday)
VALUES (gen_random_uuid(), $1, TRUE, $2, $3, $4)
RETURNING *;
