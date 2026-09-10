-- +goose Up
-- Idempotency guard for worker-materialized rule instances (D4).
-- Partial indexes constrain only rows with source_rule_id set, leaving ad-hoc
-- one-time rows (source_rule_id IS NULL) unaffected.

CREATE UNIQUE INDEX IF NOT EXISTS ux_expenses_rule_instance
    ON expenses (user_id, source_rule_id, expense_date)
    WHERE source_rule_id IS NOT NULL;

CREATE UNIQUE INDEX IF NOT EXISTS ux_incomes_rule_instance
    ON incomes (user_id, source_rule_id, date)
    WHERE source_rule_id IS NOT NULL;

-- +goose Down
DROP INDEX IF EXISTS ux_expenses_rule_instance;
DROP INDEX IF EXISTS ux_incomes_rule_instance;
