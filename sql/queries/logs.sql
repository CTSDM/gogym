-- name: CreateLog :one
INSERT INTO logs (weight, reps, logs_order, exercise_id, set_id)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: GetLog :one
SELECT * FROM logs
WHERE id = $1;
