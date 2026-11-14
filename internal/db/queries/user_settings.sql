-- Per-user settings (#9).

-- name: GetUserSettings :one
SELECT * FROM user_settings WHERE user_id = $1;

-- name: PutUserSettings :one
INSERT INTO user_settings (user_id, timezone)
VALUES ($1, $2)
ON CONFLICT (user_id) DO UPDATE SET timezone = EXCLUDED.timezone, updated_at = now()
RETURNING *;
