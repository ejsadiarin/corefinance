-- name: CreateTag :one
INSERT INTO tags (user_id, name, color)
VALUES ($1, $2, $3)
RETURNING *;

-- name: ListTags :many
SELECT * FROM tags
WHERE user_id = $1
ORDER BY name;

-- name: GetTag :one
SELECT * FROM tags
WHERE id = $1 AND user_id = $2;

-- name: UpdateTag :one
UPDATE tags
SET name = $3, color = $4
WHERE id = $1 AND user_id = $2
RETURNING *;

-- name: DeleteTag :exec
DELETE FROM tags
WHERE id = $1 AND user_id = $2;
