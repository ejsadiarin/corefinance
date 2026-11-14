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

-- name: ListActiveExpenseRulesByUser :many
SELECT * FROM recurring_expense_rules
WHERE user_id = $1
  AND is_active = true
  AND start_date <= $2
  AND (end_date IS NULL OR end_date >= $2)
ORDER BY start_date ASC;

-- name: ListActiveIncomeRulesByUser :many
SELECT * FROM recurring_income_rules
WHERE user_id = $1
  AND is_active = true
  AND start_date <= $2
  AND (end_date IS NULL OR end_date >= $2)
ORDER BY start_date ASC;

-- name: ListLegacyExpensesByUser :many
SELECT * FROM expenses
WHERE user_id = $1 AND recurring_type != 'one-time' AND source_rule_id IS NULL
ORDER BY start_date ASC NULLS LAST, expense_date ASC;

-- name: ListLegacyIncomesByUser :many
SELECT * FROM incomes
WHERE user_id = $1 AND recurring_type != 'one-time' AND source_rule_id IS NULL
ORDER BY start_date ASC NULLS LAST, date ASC;

-- name: CountLegacyExpensesByUser :one
SELECT COUNT(*)::bigint AS legacy_count FROM expenses
WHERE user_id = $1 AND recurring_type != 'one-time' AND source_rule_id IS NULL;

-- name: CountLegacyIncomesByUser :one
SELECT COUNT(*)::bigint AS legacy_count FROM incomes
WHERE user_id = $1 AND recurring_type != 'one-time' AND source_rule_id IS NULL;

-- Catch-up pass (#7), one user at a time under a per-user lock (#14,
-- ADR-0001). A rule is behind when its watermark (materialized_through, or
-- the day before start_date when nothing has been filled) is earlier than
-- min(end_date, today). Rules that ended while the worker was down are still
-- included. Paused rules are not.

-- Serializes every writer of one user's rule entries (the worker and rule
-- saves) for the rest of the transaction. Take it before touching any rule
-- row, so writers always lock in the same order.
-- name: LockUserMaterialize :exec
SELECT pg_advisory_xact_lock(hashtext('corefinance-materialize'), hashtext(sqlc.arg(user_id)::uuid::text));

-- Users with any rule behind as of `horizon`, the latest date anywhere; each
-- user's own date is applied by ListUserRulesBehind.
-- name: ListUsersWithRulesBehind :many
SELECT DISTINCT user_id FROM rules
WHERE is_active = true
  AND start_date <= sqlc.arg(horizon)::date
  AND COALESCE(materialized_through, start_date - 1)
      < LEAST(COALESCE(end_date, sqlc.arg(horizon)::date), sqlc.arg(horizon)::date);

-- name: ListUserRulesBehind :many
SELECT * FROM rules
WHERE user_id = sqlc.arg(user_id)
  AND is_active = true
  AND start_date <= sqlc.arg(today)::date
  AND COALESCE(materialized_through, start_date - 1)
      < LEAST(COALESCE(end_date, sqlc.arg(today)::date), sqlc.arg(today)::date)
ORDER BY start_date ASC, id ASC;

-- name: AdvanceRuleWatermark :exec
UPDATE rules
SET materialized_through = GREATEST(materialized_through, sqlc.arg(through)::date)
WHERE id = sqlc.arg(id);

-- The watermark only moves forward: GREATEST ignores a NULL current value.
-- name: AdvanceExpenseRuleWatermark :exec
UPDATE recurring_expense_rules
SET materialized_through = GREATEST(materialized_through, sqlc.arg(through)::date)
WHERE id = sqlc.arg(id);

-- name: AdvanceIncomeRuleWatermark :exec
UPDATE recurring_income_rules
SET materialized_through = GREATEST(materialized_through, sqlc.arg(through)::date)
WHERE id = sqlc.arg(id);

-- Ending a rule in the past (#5): the rule's entries dated after the new end
-- date and up to the old one (NULL: no old end), not already skipped.

-- name: ListExpenseRuleEntriesAfterEnd :many
SELECT id, expense_date AS date, status FROM expenses
WHERE user_id = sqlc.arg(user_id) AND source_rule_id = sqlc.arg(rule_id)
  AND expense_date > sqlc.arg(after_end)::date
  AND (sqlc.narg(old_end)::date IS NULL OR expense_date <= sqlc.narg(old_end)::date)
  AND status != 'skipped'
ORDER BY expense_date;

-- name: SkipExpenseRuleEntriesAfterEnd :many
UPDATE expenses SET status = 'skipped', updated_at = now()
WHERE user_id = sqlc.arg(user_id) AND source_rule_id = sqlc.arg(rule_id)
  AND expense_date > sqlc.arg(after_end)::date
  AND (sqlc.narg(old_end)::date IS NULL OR expense_date <= sqlc.narg(old_end)::date)
  AND status != 'skipped'
RETURNING id, expense_date AS date, status;

-- name: ListIncomeRuleEntriesAfterEnd :many
SELECT id, date, status FROM incomes
WHERE user_id = sqlc.arg(user_id) AND source_rule_id = sqlc.arg(rule_id)
  AND date > sqlc.arg(after_end)::date
  AND (sqlc.narg(old_end)::date IS NULL OR date <= sqlc.narg(old_end)::date)
  AND status != 'skipped'
ORDER BY date;

-- name: SkipIncomeRuleEntriesAfterEnd :many
UPDATE incomes SET status = 'skipped', updated_at = now()
WHERE user_id = sqlc.arg(user_id) AND source_rule_id = sqlc.arg(rule_id)
  AND date > sqlc.arg(after_end)::date
  AND (sqlc.narg(old_end)::date IS NULL OR date <= sqlc.narg(old_end)::date)
  AND status != 'skipped'
RETURNING id, date, status;
