-- +goose Up
-- Pause switch for income rules, matching recurring_expense_rules.
-- Existing rows stay active via the column default; no backfill needed.

ALTER TABLE recurring_income_rules
    ADD COLUMN IF NOT EXISTS is_active boolean NOT NULL DEFAULT true;

-- +goose Down
ALTER TABLE recurring_income_rules
    DROP COLUMN IF EXISTS is_active;
