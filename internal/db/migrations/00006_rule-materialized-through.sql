-- +goose Up
-- Per-rule watermark for the worker's catch-up pass (#7). materialized_through
-- is the last date the worker has filled for the rule; each pass inserts the
-- due dates after it up to min(end_date, today) and moves it forward. It never
-- moves backward, so an entry the user deleted is not recreated.
--
-- NULL means nothing has been filled yet: the next pass starts at start_date.
-- Existing rules are seeded with their latest rule-created entry, so history
-- before that entry (including deleted entries) is left alone, and any gap
-- after it (the worker was down) is filled on the first pass.

ALTER TABLE recurring_expense_rules ADD COLUMN IF NOT EXISTS materialized_through date;
ALTER TABLE recurring_income_rules ADD COLUMN IF NOT EXISTS materialized_through date;

UPDATE recurring_expense_rules r
SET materialized_through = (
    SELECT max(e.expense_date) FROM expenses e WHERE e.source_rule_id = r.id
);

UPDATE recurring_income_rules r
SET materialized_through = (
    SELECT max(i.date) FROM incomes i WHERE i.source_rule_id = r.id
);

-- +goose Down
ALTER TABLE recurring_income_rules DROP COLUMN IF EXISTS materialized_through;
ALTER TABLE recurring_expense_rules DROP COLUMN IF EXISTS materialized_through;
