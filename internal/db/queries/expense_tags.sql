-- name: AddTagToExpense :exec
INSERT INTO expense_tags (expense_id, tag_id)
VALUES ($1, $2)
ON CONFLICT DO NOTHING;

-- name: RemoveTagFromExpense :exec
DELETE FROM expense_tags
WHERE expense_id = $1 AND tag_id = $2;

-- name: GetTagsByExpenseID :many
SELECT t.* FROM tags t
JOIN expense_tags et ON t.id = et.tag_id
WHERE et.expense_id = $1
ORDER BY t.name;

-- name: ListAllExpenseTagsByUser :many
SELECT et.expense_id, et.tag_id FROM expense_tags et
JOIN expenses e ON et.expense_id = e.id
WHERE e.user_id = $1;
