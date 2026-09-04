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
WHERE i.user_id = @user_id
  AND (@start_date::date IS NULL OR i.date >= @start_date)
  AND (@end_date::date IS NULL OR i.date <= @end_date)
  AND (@category_id::uuid IS NULL OR i.category_id = @category_id)
  AND (@status::varchar IS NULL OR i.status = @status)
  AND (@recurring_type::varchar IS NULL OR i.recurring_type = @recurring_type)
ORDER BY i.date DESC, i.created_at DESC
LIMIT @page_limit OFFSET @page_offset;

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
WHERE user_id = @user_id
  AND recurring_type != 'one-time'
  AND (@start_date::date IS NULL OR date >= @start_date)
  AND (@end_date::date IS NULL OR date <= @end_date)
ORDER BY date DESC;

-- name: GetTotalIncomesByDateRange :one
SELECT COALESCE(SUM(amount), 0)::numeric AS total
FROM incomes
WHERE user_id = $1
  AND date >= $2
  AND date <= $3
  AND status != 'skipped';

-- name: ListAllIncomesByUser :many
SELECT * FROM incomes
WHERE user_id = $1
ORDER BY date DESC, created_at DESC;
