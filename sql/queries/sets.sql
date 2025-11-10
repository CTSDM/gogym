-- name: CreateSet :one
INSERT INTO sets (set_order, rest_time, session_id, exercise_id)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: UpdateSet :one
UPDATE sets
SET
    set_order = $1,
    rest_time = $2,
    exercise_id = $3
WHERE id = $4
RETURNING *;

-- name: GetSet :one
SELECT * FROM sets
WHERE id = $1;

-- name: GetSetsBySessionIDs :many
SELECT * FROM sets
WHERE session_id = ANY($1::uuid[])
ORDER BY session_id, set_order;

-- name: DeleteSet :one
DELETE FROM sets
WHERE id = $1
RETURNING *;

-- name: GetSetOwnerID :one
SELECT sessions.user_id FROM sets
JOIN sessions
ON sets.session_id = sessions.id
WHERE sets.id = $1;
