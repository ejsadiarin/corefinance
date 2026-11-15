-- +goose Up
-- Every query reads and writes `rules` directly now (#10, ADR-0001), so the
-- per-kind views kept for the move to one table (#11) go.
DROP VIEW recurring_expense_rules;
DROP VIEW recurring_income_rules;

-- +goose Down
CREATE VIEW recurring_expense_rules AS
SELECT id, user_id, description, amount, currency, category_id, notes, recurring_type, start_date,
       end_date, priority, is_active, created_at, updated_at, materialized_through, kind
FROM rules WHERE kind = 'expense'
WITH CHECK OPTION;
ALTER VIEW recurring_expense_rules ALTER COLUMN kind SET DEFAULT 'expense';

CREATE VIEW recurring_income_rules AS
SELECT id, user_id, amount, currency, description, recurring_type, start_date, end_date, created_at,
       updated_at, is_active, category_id, notes, priority, materialized_through, kind
FROM rules WHERE kind = 'income'
WITH CHECK OPTION;
ALTER VIEW recurring_income_rules ALTER COLUMN kind SET DEFAULT 'income';
