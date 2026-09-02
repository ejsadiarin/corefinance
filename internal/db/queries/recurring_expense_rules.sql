-- name: CreateRecurringExpenseRule :one
INSERT INTO recurring_expense_rules (user_id, description, amount, currency, category_id, notes, recurring_type, start_date, end_date, priority)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
RETURNING *;

-- name: ListRecurringExpenseRules :many
SELECT r.*, ec.name AS category_name, ec.color AS category_color, ec.icon AS category_icon
FROM recurring_expense_rules r
LEFT JOIN expense_categories ec ON r.category_id = ec.id
WHERE r.user_id = $1 AND r.is_active = true
ORDER BY r.created_at DESC;

-- name: GetRecurringExpenseRule :one
SELECT r.*, ec.name AS category_name, ec.color AS category_color, ec.icon AS category_icon
FROM recurring_expense_rules r
LEFT JOIN expense_categories ec ON r.category_id = ec.id
WHERE r.id = $1 AND r.user_id = $2;

-- name: UpdateRecurringExpenseRule :one
UPDATE recurring_expense_rules
SET description = $3, amount = $4, currency = $5, category_id = $6, notes = $7,
    recurring_type = $8, start_date = $9, end_date = $10, priority = $11,
    updated_at = now()
WHERE id = $1 AND user_id = $2
RETURNING *;

-- name: DeleteRecurringExpenseRule :exec
DELETE FROM recurring_expense_rules
WHERE id = $1 AND user_id = $2;
