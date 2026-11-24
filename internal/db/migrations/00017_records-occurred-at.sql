-- +goose Up
-- When a budget record happened (#28), when the owner says. The record's
-- day (expense_date / date) stays what every budget figure counts; this is
-- the moment within it, stored as UTC. Null for records without a time:
-- everything before this, rule entries, and records entered by day only.

ALTER TABLE expenses ADD COLUMN occurred_at timestamptz;
ALTER TABLE incomes ADD COLUMN occurred_at timestamptz;

-- +goose Down
ALTER TABLE incomes DROP COLUMN occurred_at;
ALTER TABLE expenses DROP COLUMN occurred_at;
