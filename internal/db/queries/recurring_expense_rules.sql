-- Expense rules: the rules of kind 'expense' (ADR-0001). Every query names
-- the kind, so an income rule's id never matches here.

-- name: CreateRecurringExpenseRule :one
INSERT INTO rules (kind, user_id, description, amount, currency, category_id, notes, recurring_type, start_date, end_date, priority)
VALUES ('expense', $1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
RETURNING *;

-- name: ListRecurringExpenseRules :many
SELECT r.*, ec.name AS category_name, ec.color AS category_color, ec.icon AS category_icon
FROM rules r
LEFT JOIN expense_categories ec ON r.category_id = ec.id
WHERE r.user_id = $1 AND r.kind = 'expense' AND r.is_active = true
ORDER BY r.created_at DESC;

-- name: GetRecurringExpenseRule :one
SELECT r.*, ec.name AS category_name, ec.color AS category_color, ec.icon AS category_icon
FROM rules r
LEFT JOIN expense_categories ec ON r.category_id = ec.id
WHERE r.id = $1 AND r.user_id = $2 AND r.kind = 'expense';

-- name: UpdateRecurringExpenseRule :one
UPDATE rules
SET description = $3, amount = $4, currency = $5, category_id = $6, notes = $7,
    recurring_type = $8, start_date = $9, end_date = $10, priority = $11,
    -- Resuming a paused rule (#7): skip the paused stretch so the materialize
    -- pass doesn't fill it. RHS columns are the pre-update values.
    materialized_through = CASE
        WHEN NOT is_active AND sqlc.narg(is_active)::bool
        THEN GREATEST(COALESCE(materialized_through, start_date - 1), sqlc.narg(resume_through)::date)
        ELSE materialized_through
    END,
    is_active = COALESCE(sqlc.narg(is_active), is_active),
    updated_at = now()
WHERE id = $1 AND user_id = $2 AND kind = 'expense'
RETURNING *;

-- name: DeleteRecurringExpenseRule :exec
DELETE FROM rules
WHERE id = $1 AND user_id = $2 AND kind = 'expense';

-- ListRecurringExpenseRulesWithPaused lists every rule, paused ones too, so
-- the web can show a paused rule and resume it.
-- name: ListRecurringExpenseRulesWithPaused :many
SELECT r.*, ec.name AS category_name, ec.color AS category_color, ec.icon AS category_icon
FROM rules r
LEFT JOIN expense_categories ec ON r.category_id = ec.id
WHERE r.user_id = $1 AND r.kind = 'expense'
ORDER BY r.created_at DESC;
