-- name: CreateLog :one
INSERT INTO logs (weight, reps, logs_order, exercise_id, set_id)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: GetLog :one
SELECT * FROM logs
WHERE id = $1;

-- name: UpdateLog :one
UPDATE logs
SET weight = $1, reps = $2, logs_order = $3
WHERE id = $4
RETURNING *;

-- name: GetLogOwnerID :one
SELECT sessions.user_id FROM logs
JOIN sets
ON logs.set_id = sets.id
JOIN sessions
ON sets.session_id = sessions.id
WHERE logs.id = $1;

-- name: GetLogsBySetIDs :many
SELECT * FROM logs
WHERE set_id = ANY($1::bigint[])
ORDER BY set_id, logs_order;
