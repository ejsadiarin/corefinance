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

-- Next-due scheduling (#16): users with an active rule whose next occurrence
-- has started, or whose next_due_at hasn't been computed yet.
-- name: ListUsersDue :many
SELECT DISTINCT user_id FROM rules
WHERE is_active = true
  AND (next_due_at IS NULL OR next_due_at <= sqlc.arg(now)::timestamptz);

-- name: ListUserActiveRules :many
SELECT * FROM rules
WHERE user_id = sqlc.arg(user_id) AND is_active = true
ORDER BY start_date ASC, id ASC;

-- name: SetRuleNextDue :exec
UPDATE rules SET next_due_at = sqlc.arg(next_due_at)::timestamptz WHERE id = sqlc.arg(id);

-- The earliest next occurrence still to come, if any (NULL when none).
-- name: GetNextWake :one
SELECT min(next_due_at)::timestamptz AS next_wake FROM rules
WHERE is_active = true AND next_due_at > sqlc.arg(now)::timestamptz AND next_due_at < 'infinity';

-- A timezone change moves every next occurrence: the next pass recomputes.
-- name: ClearUserNextDue :exec
UPDATE rules SET next_due_at = NULL WHERE user_id = sqlc.arg(user_id);

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

-- Reconciling a rule's entries after a schedule change (#15, ADR-0001): the
-- rule's entries that aren't skipped, with whether the user edited them.
-- name: ListRuleEntriesToReconcile :many
SELECT id, date, status, user_edited_at FROM (
    SELECT id, expense_date AS date, status, user_edited_at FROM expenses
    WHERE user_id = sqlc.arg(owner)::uuid AND source_rule_id = sqlc.arg(rule_id)::uuid AND status != 'skipped'
    UNION ALL
    SELECT id, date, status, user_edited_at FROM incomes
    WHERE user_id = sqlc.arg(owner)::uuid AND source_rule_id = sqlc.arg(rule_id)::uuid AND status != 'skipped'
) entries
ORDER BY date;

-- A rule's ids live in one of the two tables; each statement touches only
-- its own. Skipping keeps user_edited_at: the rule skipped it, not the user.
-- name: SkipExpenseEntries :exec
UPDATE expenses SET status = 'skipped', updated_at = now()
WHERE user_id = sqlc.arg(user_id) AND id = ANY(sqlc.arg(ids)::uuid[]);

-- name: SkipIncomeEntries :exec
UPDATE incomes SET status = 'skipped', updated_at = now()
WHERE user_id = sqlc.arg(user_id) AND id = ANY(sqlc.arg(ids)::uuid[]);

-- Keeping a stale entry is the user's decision, so it counts as an edit:
-- a later schedule change reports it for review instead of skipping it.
-- name: MarkExpenseEntriesKept :exec
UPDATE expenses SET user_edited_at = now(), updated_at = now()
WHERE user_id = sqlc.arg(user_id) AND id = ANY(sqlc.arg(ids)::uuid[]);

-- name: MarkIncomeEntriesKept :exec
UPDATE incomes SET user_edited_at = now(), updated_at = now()
WHERE user_id = sqlc.arg(user_id) AND id = ANY(sqlc.arg(ids)::uuid[]);
