-- The budget part of an export (coregateway ADR-0008). Import keeps each
-- record's id: a row the owner already has is skipped (ON CONFLICT DO
-- NOTHING), and ids another owner holds are found first and fail the import.

-- name: ListAllRulesByUser :many
SELECT * FROM rules
WHERE user_id = $1
ORDER BY kind, start_date, id;

-- Ids in the file that belong to someone other than the importing owner.
-- name: ListForeignBudgetIDs :many
SELECT id, 'tag'::text AS entity FROM tags
WHERE id = ANY(sqlc.arg(tag_ids)::uuid[]) AND user_id <> sqlc.arg(user_id)::uuid
UNION ALL
SELECT id, 'rule'::text FROM rules
WHERE id = ANY(sqlc.arg(rule_ids)::uuid[]) AND user_id <> sqlc.arg(user_id)::uuid
UNION ALL
SELECT id, 'expense'::text FROM expenses
WHERE id = ANY(sqlc.arg(expense_ids)::uuid[]) AND user_id <> sqlc.arg(user_id)::uuid
UNION ALL
SELECT id, 'income'::text FROM incomes
WHERE id = ANY(sqlc.arg(income_ids)::uuid[]) AND user_id <> sqlc.arg(user_id)::uuid;

-- name: ImportTag :execrows
INSERT INTO tags (id, user_id, name, color, created_at)
VALUES (sqlc.arg(id), sqlc.arg(user_id), sqlc.arg(name), sqlc.narg(color), COALESCE(sqlc.narg(created_at)::timestamptz, now()))
ON CONFLICT DO NOTHING;

-- next_due_at is left NULL: the worker computes it from materialized_through
-- on its next pass, as for a rule it has never scheduled (ADR-0001).
-- name: ImportRule :execrows
INSERT INTO rules (id, kind, user_id, description, amount, currency, category_id, notes,
                   recurring_type, start_date, end_date, priority, is_active,
                   materialized_through, created_at)
VALUES (sqlc.arg(id), sqlc.arg(kind), sqlc.arg(user_id), sqlc.arg(description), sqlc.arg(amount),
        sqlc.arg(currency), sqlc.narg(category_id), sqlc.narg(notes), sqlc.arg(recurring_type),
        sqlc.arg(start_date), sqlc.narg(end_date), sqlc.arg(priority), sqlc.arg(is_active),
        sqlc.narg(materialized_through), COALESCE(sqlc.narg(created_at)::timestamptz, now()))
ON CONFLICT DO NOTHING;

-- Also skips an entry whose rule already has one on that date for the owner.
-- name: ImportExpense :execrows
INSERT INTO expenses (id, user_id, category_id, amount, currency, exchange_rate, amount_php, rate_date,
                      description, notes, expense_date, priority, status, is_debt, source_rule_id,
                      user_edited_at, occurred_at, created_at)
VALUES (sqlc.arg(id), sqlc.arg(user_id), sqlc.narg(category_id), sqlc.arg(amount), sqlc.arg(currency),
        sqlc.arg(exchange_rate), sqlc.arg(amount_php), sqlc.arg(rate_date), sqlc.arg(description),
        sqlc.narg(notes), sqlc.arg(expense_date), sqlc.arg(priority), sqlc.arg(status), sqlc.arg(is_debt),
        sqlc.narg(source_rule_id), sqlc.narg(user_edited_at), sqlc.narg(occurred_at), COALESCE(sqlc.narg(created_at)::timestamptz, now()))
ON CONFLICT DO NOTHING;

-- name: ImportIncome :execrows
INSERT INTO incomes (id, user_id, category_id, amount, currency, exchange_rate, amount_php, rate_date,
                     description, notes, date, priority, status, source_rule_id,
                     user_edited_at, occurred_at, created_at)
VALUES (sqlc.arg(id), sqlc.arg(user_id), sqlc.narg(category_id), sqlc.arg(amount), sqlc.arg(currency),
        sqlc.arg(exchange_rate), sqlc.arg(amount_php), sqlc.arg(rate_date), sqlc.arg(description),
        sqlc.narg(notes), sqlc.arg(date), sqlc.arg(priority), sqlc.arg(status),
        sqlc.narg(source_rule_id), sqlc.narg(user_edited_at), sqlc.narg(occurred_at), COALESCE(sqlc.narg(created_at)::timestamptz, now()))
ON CONFLICT DO NOTHING;
