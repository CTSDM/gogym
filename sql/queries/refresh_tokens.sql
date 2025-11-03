-- name: CreateRefreshToken :one
INSERT INTO refresh_tokens (token, expires_at, user_id)
VALUES ($1, $2, $3)
RETURNING *;

-- name: GetRefreshTokenByUserID :one
SELECT *
FROM refresh_tokens
WHERE user_id = $1 AND expires_at > timezone('utc', now())
ORDER BY expires_at DESC
LIMIT 1;
