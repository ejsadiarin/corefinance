-- name: CreateRecurringIncomeRule :one
INSERT INTO recurring_income_rules (user_id, amount, currency, description, recurring_type, start_date, end_date)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING *;

-- name: ListRecurringIncomeRules :many
SELECT * FROM recurring_income_rules
WHERE user_id = $1
ORDER BY created_at DESC;

-- name: GetRecurringIncomeRule :one
SELECT * FROM recurring_income_rules
WHERE id = $1 AND user_id = $2;

-- name: UpdateRecurringIncomeRule :one
UPDATE recurring_income_rules
SET amount = $3, currency = $4, description = $5, recurring_type = $6,
    start_date = $7, end_date = $8, updated_at = now()
WHERE id = $1 AND user_id = $2
RETURNING *;

-- name: DeleteRecurringIncomeRule :exec
DELETE FROM recurring_income_rules
WHERE id = $1 AND user_id = $2;
