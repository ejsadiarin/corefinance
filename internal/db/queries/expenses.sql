-- name: CreateExpense :one
INSERT INTO expenses (user_id, category_id, amount, currency, description, notes, expense_date, recurring_type, priority, status, is_debt, start_date, end_date, source_rule_id)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
RETURNING *;

-- name: GetExpense :one
SELECT e.*, ec.name AS category_name, ec.color AS category_color, ec.icon AS category_icon
FROM expenses e
LEFT JOIN expense_categories ec ON e.category_id = ec.id
WHERE e.id = $1 AND e.user_id = $2;

-- name: ListExpenses :many
SELECT e.*, ec.name AS category_name, ec.color AS category_color, ec.icon AS category_icon
FROM expenses e
LEFT JOIN expense_categories ec ON e.category_id = ec.id
WHERE e.user_id = $1
  AND ($2::date IS NULL OR e.expense_date >= $2)
  AND ($3::date IS NULL OR e.expense_date <= $3)
  AND ($4::uuid IS NULL OR e.category_id = $4)
  AND ($5::varchar IS NULL OR e.priority = $5)
  AND ($6::varchar IS NULL OR e.status = $6)
  AND ($7::varchar IS NULL OR e.recurring_type = $7)
ORDER BY e.expense_date DESC, e.created_at DESC
LIMIT $8 OFFSET $9;

-- name: SearchExpenses :many
SELECT e.*, ec.name AS category_name, ec.color AS category_color, ec.icon AS category_icon
FROM expenses e
LEFT JOIN expense_categories ec ON e.category_id = ec.id
WHERE e.user_id = $1
  AND (e.description ILIKE '%' || $2 || '%' OR e.notes ILIKE '%' || $2 || '%')
ORDER BY e.expense_date DESC, e.created_at DESC
LIMIT $3 OFFSET $4;

-- name: UpdateExpense :one
UPDATE expenses
SET category_id = $3, amount = $4, currency = $5, description = $6, notes = $7,
    expense_date = $8, recurring_type = $9, priority = $10, status = $11,
    is_debt = $12, start_date = $13, end_date = $14, source_rule_id = $15,
    updated_at = now()
WHERE id = $1 AND user_id = $2
RETURNING *;

-- name: DeleteExpense :exec
DELETE FROM expenses
WHERE id = $1 AND user_id = $2;

-- name: SkipExpense :exec
UPDATE expenses
SET status = 'skipped', updated_at = now()
WHERE id = $1 AND user_id = $2;

-- name: CheckSkippedExpense :one
SELECT EXISTS (
    SELECT 1 FROM expenses
    WHERE user_id = $1
      AND expense_date >= $2
      AND expense_date <= $3
      AND status = 'skipped'
) AS is_skipped;

-- name: GetExpenseStatsByCategory :many
SELECT
    ec.id AS category_id,
    ec.name AS category_name,
    ec.color AS category_color,
    COUNT(e.id) AS expense_count,
    COALESCE(SUM(e.amount), 0) AS total_amount
FROM expense_categories ec
LEFT JOIN expenses e ON ec.id = e.category_id
    AND e.user_id = $1
    AND ($2::date IS NULL OR e.expense_date >= $2)
    AND ($3::date IS NULL OR e.expense_date <= $3)
    AND e.status != 'skipped'
WHERE ec.user_id = $1 AND ec.is_active = true
GROUP BY ec.id, ec.name, ec.color
ORDER BY total_amount DESC;

-- name: GetTotalExpensesByDateRange :one
SELECT COALESCE(SUM(amount), 0) AS total
FROM expenses
WHERE user_id = $1
  AND expense_date >= $2
  AND expense_date <= $3
  AND status != 'skipped';
