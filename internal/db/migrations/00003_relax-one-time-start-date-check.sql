-- +goose Up
-- Allow NULL start_date for one-time rows (worker-materialized instances).
--
-- The baseline CHECK (((recurring_type IS NULL) OR (start_date IS NOT NULL)))
-- effectively requires start_date on EVERY row because recurring_type is
-- NOT NULL, so instance rows with recurring_type='one-time' and NULL
-- start_date (D4) would violate it. The corrected CHECK keeps the original
-- intent (recurring cadences must have a start_date) while exempting
-- one-time rows. Constraint names are unchanged.

ALTER TABLE expenses DROP CONSTRAINT IF EXISTS expenses_recurring_start_date_check;
ALTER TABLE expenses ADD CONSTRAINT expenses_recurring_start_date_check
    CHECK ((recurring_type = 'one-time' OR recurring_type IS NULL) OR (start_date IS NOT NULL));

ALTER TABLE incomes DROP CONSTRAINT IF EXISTS incomes_recurring_start_date_check;
ALTER TABLE incomes ADD CONSTRAINT incomes_recurring_start_date_check
    CHECK ((recurring_type = 'one-time' OR recurring_type IS NULL) OR (start_date IS NOT NULL));

-- +goose Down
-- Staging-only: restores the baseline CHECKs. Fails while any one-time row
-- with NULL start_date exists (e.g. worker-materialized instances).
ALTER TABLE expenses DROP CONSTRAINT IF EXISTS expenses_recurring_start_date_check;
ALTER TABLE expenses ADD CONSTRAINT expenses_recurring_start_date_check
    CHECK (((recurring_type IS NULL) OR (start_date IS NOT NULL)));

ALTER TABLE incomes DROP CONSTRAINT IF EXISTS incomes_recurring_start_date_check;
ALTER TABLE incomes ADD CONSTRAINT incomes_recurring_start_date_check
    CHECK (((recurring_type IS NULL) OR (start_date IS NOT NULL)));
