-- name: CreateExpenseCategory :one
INSERT INTO expense_categories (user_id, name, color, icon)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: ListExpenseCategories :many
SELECT * FROM expense_categories
WHERE user_id = $1 AND is_active = true
ORDER BY name;

-- name: GetExpenseCategory :one
SELECT * FROM expense_categories
WHERE id = $1 AND user_id = $2;

-- name: UpdateExpenseCategory :one
UPDATE expense_categories
SET name = $3, color = $4, icon = $5, updated_at = now()
WHERE id = $1 AND user_id = $2
RETURNING *;

-- name: DeleteExpenseCategory :exec
UPDATE expense_categories
SET is_active = false, updated_at = now()
WHERE id = $1 AND user_id = $2;

-- name: CreateIncomeCategory :one
INSERT INTO income_categories (user_id, name, color, icon)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: ListIncomeCategories :many
SELECT * FROM income_categories
WHERE user_id = $1 AND is_active = true
ORDER BY name;

-- name: GetIncomeCategory :one
SELECT * FROM income_categories
WHERE id = $1 AND user_id = $2;

-- name: UpdateIncomeCategory :one
UPDATE income_categories
SET name = $3, color = $4, icon = $5, updated_at = now()
WHERE id = $1 AND user_id = $2
RETURNING *;

-- name: DeleteIncomeCategory :exec
UPDATE income_categories
SET is_active = false, updated_at = now()
WHERE id = $1 AND user_id = $2;
