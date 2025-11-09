-- name: CreateSession :one
INSERT INTO sessions (name, date, start_timestamp, duration_minutes, user_id)
VALUES (
    $1, $2, $3, $4, $5
)
RETURNING *;

-- name: GetSessionsByUserID :many
SELECT * FROM sessions
WHERE user_id = $1
ORDER BY date DESC;

-- name: GetSession :one
SELECT * FROM sessions
WHERE id = $1;

-- name: GetSessionsPaginated :many
SELECT * FROM sessions
WHERE user_id = $1
ORDER BY date DESC
OFFSET $2
LIMIT $3;

-- name: GetSessionOwnerID :one
SELECT user_id FROM sessions
WHERE id = $1;

-- name: UpdateSession :one
UPDATE sessions
SET name = $1,
    date = $2,
    start_timestamp = $3,
    duration_minutes = $4
WHERE id = $5
RETURNING *;
