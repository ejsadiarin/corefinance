-- name: AddTagToIncome :exec
INSERT INTO income_tags (income_id, tag_id)
VALUES ($1, $2)
ON CONFLICT DO NOTHING;

-- name: RemoveTagFromIncome :exec
DELETE FROM income_tags
WHERE income_id = $1 AND tag_id = $2;

-- name: GetTagsByIncomeID :many
SELECT t.* FROM tags t
JOIN income_tags it ON t.id = it.tag_id
WHERE it.income_id = $1
ORDER BY t.name;

-- name: ListAllIncomeTagsByUser :many
SELECT it.income_id, it.tag_id FROM income_tags it
JOIN incomes i ON it.income_id = i.id
WHERE i.user_id = $1;
