-- name: CreateSet :one
INSERT INTO sets (set_order, rest_time, session_id, exercise_id)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: GetSet :one
SELECT * FROM sets
WHERE id = $1;
