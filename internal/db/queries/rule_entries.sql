-- Rule-created entries for review (#8): the expenses and incomes the worker
-- (or a rule write) created, newest-inserted first. Every filter is optional.
-- created_since compares the insert time, so entries back-filled with older
-- dates still show up as new.

-- name: ListRuleEntries :many
SELECT * FROM (
    SELECT 'expense'::text AS kind, e.id, e.source_rule_id::uuid AS rule_id, e.expense_date AS date,
           e.amount, e.currency, e.description, e.notes, e.category_id, ec.name AS category_name,
           e.priority, e.status, e.created_at, e.updated_at
    FROM expenses e
    LEFT JOIN expense_categories ec ON ec.id = e.category_id
    WHERE e.user_id = sqlc.arg(user_id)
      AND e.source_rule_id IS NOT NULL
      AND (sqlc.narg(rule_id)::uuid IS NULL OR e.source_rule_id = sqlc.narg(rule_id)::uuid)
      AND (sqlc.narg(start_date)::date IS NULL OR e.expense_date >= sqlc.narg(start_date)::date)
      AND (sqlc.narg(end_date)::date IS NULL OR e.expense_date <= sqlc.narg(end_date)::date)
      AND (sqlc.narg(created_since)::timestamptz IS NULL OR e.created_at > sqlc.narg(created_since)::timestamptz)
    UNION ALL
    SELECT 'income'::text AS kind, i.id, i.source_rule_id::uuid AS rule_id, i.date,
           i.amount, i.currency, i.description, i.notes, i.category_id, ic.name AS category_name,
           i.priority, i.status, i.created_at, i.updated_at
    FROM incomes i
    LEFT JOIN income_categories ic ON ic.id = i.category_id
    WHERE i.user_id = sqlc.arg(user_id)
      AND i.source_rule_id IS NOT NULL
      AND (sqlc.narg(rule_id)::uuid IS NULL OR i.source_rule_id = sqlc.narg(rule_id)::uuid)
      AND (sqlc.narg(start_date)::date IS NULL OR i.date >= sqlc.narg(start_date)::date)
      AND (sqlc.narg(end_date)::date IS NULL OR i.date <= sqlc.narg(end_date)::date)
      AND (sqlc.narg(created_since)::timestamptz IS NULL OR i.created_at > sqlc.narg(created_since)::timestamptz)
) entries
ORDER BY created_at DESC, date DESC, id DESC
LIMIT sqlc.arg(page_limit) OFFSET sqlc.arg(page_offset);
