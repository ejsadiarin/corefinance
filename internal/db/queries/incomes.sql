-- name: CreateIncome :one
INSERT INTO incomes (user_id, category_id, amount, currency, exchange_rate, amount_php, rate_date, description, notes, date, priority, status, source_rule_id)
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
  AND ('00000000-0000-0000-0000-000000000000' = @category_id::uuid OR i.category_id = @category_id)
  AND ('' = @status OR i.status = @status)
ORDER BY i.date DESC, i.created_at DESC
LIMIT @page_limit OFFSET @page_offset;

-- name: UpdateIncome :one
UPDATE incomes
SET category_id = $3, amount = $4, currency = $5, exchange_rate = $6,
    amount_php = $7, rate_date = $8, description = $9, notes = $10,
    date = $11, priority = $12, status = $13, source_rule_id = $14,
    updated_at = now(), user_edited_at = now()
WHERE id = $1 AND user_id = $2
RETURNING *;

-- name: DeleteIncome :exec
DELETE FROM incomes
WHERE id = $1 AND user_id = $2;

-- name: SkipIncome :exec
UPDATE incomes
SET status = 'skipped', updated_at = now(), user_edited_at = now()
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
-- Incomes a recurring rule created. The worker stamps them one-time with
-- source_rule_id set, so the link, not the type, marks an occurrence.
SELECT * FROM incomes
WHERE user_id = @user_id
  AND source_rule_id IS NOT NULL
  AND (@start_date::date IS NULL OR date >= @start_date)
  AND (@end_date::date IS NULL OR date <= @end_date)
ORDER BY date DESC;

-- name: GetTotalIncomesByDateRange :one
SELECT COALESCE(SUM(amount_php), 0)::numeric AS total
FROM incomes
WHERE user_id = $1
  AND date >= $2
  AND date <= $3
  AND status != 'skipped';

-- name: ListAllIncomesByUser :many
SELECT * FROM incomes
WHERE user_id = $1
ORDER BY date DESC, created_at DESC;

-- name: ListIncomesByIDs :many
-- The caller's incomes among the given ids; ids that don't exist or
-- belong to someone else are simply absent.
SELECT i.*, ic.name AS category_name, ic.color AS category_color, ic.icon AS category_icon
FROM incomes i
LEFT JOIN income_categories ic ON i.category_id = ic.id
WHERE i.user_id = @user_id AND i.id = ANY(@ids::uuid[])
ORDER BY i.date DESC, i.created_at DESC;
