-- name: CreateSet :one
INSERT INTO sets (set_order, rest_time, session_id, exercise_id)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: GetSet :one
SELECT * FROM sets
WHERE id = $1;

-- name: GetSetsBySessionIDs :many
SELECT * FROM sets
WHERE session_id = ANY($1::uuid[])
ORDER BY session_id, set_order;
