-- Worker materialization queries (D3/D4/D5): active-rule selection, idempotent
-- instance inserts, and legacy inline-rule backfill support.

-- name: ListActiveExpenseRules :many
SELECT * FROM recurring_expense_rules
WHERE is_active = true
  AND start_date <= $1
  AND (end_date IS NULL OR end_date >= $1)
ORDER BY start_date ASC;

-- name: ListActiveIncomeRules :many
SELECT * FROM recurring_income_rules
WHERE is_active = true
  AND start_date <= $1
  AND (end_date IS NULL OR end_date >= $1)
ORDER BY start_date ASC;

-- name: CreateExpenseInstance :execrows
INSERT INTO expenses (user_id, category_id, amount, currency, description, notes, expense_date, recurring_type, priority, status, is_debt, start_date, end_date, source_rule_id)
VALUES ($1, $2, $3, $4, $5, $6, $7, 'one-time', $8, 'posted', $9, NULL, NULL, $10)
ON CONFLICT (user_id, source_rule_id, expense_date) WHERE source_rule_id IS NOT NULL DO NOTHING;

-- name: CreateIncomeInstance :execrows
INSERT INTO incomes (user_id, category_id, amount, currency, description, notes, date, recurring_type, priority, status, start_date, end_date, source_rule_id)
VALUES ($1, $2, $3, $4, $5, $6, $7, 'one-time', $8, 'posted', NULL, NULL, $9)
ON CONFLICT (user_id, source_rule_id, date) WHERE source_rule_id IS NOT NULL DO NOTHING;

-- name: ListLegacyExpenses :many
SELECT * FROM expenses
WHERE recurring_type != 'one-time' AND source_rule_id IS NULL
ORDER BY start_date ASC NULLS LAST, expense_date ASC;

-- name: ListLegacyIncomes :many
SELECT * FROM incomes
WHERE recurring_type != 'one-time' AND source_rule_id IS NULL
ORDER BY start_date ASC NULLS LAST, date ASC;

-- name: FlipLegacyExpenseToOneTime :exec
UPDATE expenses SET recurring_type = 'one-time', updated_at = now()
WHERE id = $1;

-- name: FlipLegacyIncomeToOneTime :exec
UPDATE incomes SET recurring_type = 'one-time', updated_at = now()
WHERE id = $1;

-- name: CountLegacyExpenses :one
SELECT COUNT(*)::bigint AS legacy_count FROM expenses
WHERE recurring_type != 'one-time' AND source_rule_id IS NULL;

-- name: CountLegacyIncomes :one
SELECT COUNT(*)::bigint AS legacy_count FROM incomes
WHERE recurring_type != 'one-time' AND source_rule_id IS NULL;
