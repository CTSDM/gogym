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

-- name: UpdateLogsExerciseIDBySetID :exec
UPDATE logs
SET exercise_id = $1
WHERE set_id = $2;

-- name: GetLogsBySetID :many
SELECT * FROM logs
WHERE set_id = $1
ORDER BY logs_order ASC;

-- name: DeleteLog :one
DELETE FROM logs
WHERE id = $1
RETURNING *;

-- name: GetLogsByUserID :many
SELECT sessions.date, logs.*
FROM logs
LEFT JOIN sets ON sets.id = logs.set_id
LEFT JOIN sessions ON sessions.id = sets.session_id
WHERE sessions.user_id = $1
ORDER BY sessions.date DESC, logs.logs_order DESC
OFFSET $2
LIMIT $3;
