-- name: CreateIncome :one
INSERT INTO incomes (user_id, category_id, amount, currency, description, notes, date, recurring_type, priority, status, start_date, end_date, source_rule_id)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
RETURNING *;

-- name: GetIncome :one
SELECT i.*, ic.name AS category_name, ic.color AS category_color, ic.icon AS category_icon
FROM incomes i
LEFT JOIN income_categories ic ON i.category_id = ic.id
WHERE i.id = $1 AND i.user_id = $2;

-- name: ListIncomes :many
SELECT i.*, ic.name AS category_name, ic.color AS category_color, ic.icon AS category_icon
FROM incomes i
LEFT JOIN income_categories ic ON i.category_id = ic.id
WHERE i.user_id = $1
  AND ($2::date IS NULL OR i.date >= $2)
  AND ($3::date IS NULL OR i.date <= $3)
  AND ($4::uuid IS NULL OR i.category_id = $4)
  AND ($5::varchar IS NULL OR i.status = $5)
  AND ($6::varchar IS NULL OR i.recurring_type = $6)
ORDER BY i.date DESC, i.created_at DESC
LIMIT $7 OFFSET $8;

-- name: UpdateIncome :one
UPDATE incomes
SET category_id = $3, amount = $4, currency = $5, description = $6, notes = $7,
    date = $8, recurring_type = $9, priority = $10, status = $11,
    start_date = $12, end_date = $13, source_rule_id = $14,
    updated_at = now()
WHERE id = $1 AND user_id = $2
RETURNING *;

-- name: DeleteIncome :exec
DELETE FROM incomes
WHERE id = $1 AND user_id = $2;

-- name: SkipIncome :exec
UPDATE incomes
SET status = 'skipped', updated_at = now()
WHERE id = $1 AND user_id = $2;

-- name: CheckSkippedIncome :one
SELECT EXISTS (
    SELECT 1 FROM incomes
    WHERE user_id = $1
      AND date >= $2
      AND date <= $3
      AND status = 'skipped'
) AS is_skipped;

-- name: GetIncomeOccurrences :many
SELECT * FROM incomes
WHERE user_id = $1
  AND recurring_type != 'one-time'
  AND ($2::date IS NULL OR date >= $2)
  AND ($3::date IS NULL OR date <= $3)
ORDER BY date DESC;

-- name: GetTotalIncomesByDateRange :one
SELECT COALESCE(SUM(amount), 0) AS total
FROM incomes
WHERE user_id = $1
  AND date >= $2
  AND date <= $3
  AND status != 'skipped';
