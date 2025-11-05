-- name: CreateExercise :one
INSERT INTO exercises (name, description)
VALUES ($1, $2)
RETURNING *;

-- name: GetExercise :one
SELECT * FROM exercises
WHERE id = $1;

-- name: GetExercises :many
SELECT * FROM exercises;
