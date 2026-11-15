-- +goose Up
-- Expenses and incomes are entries only; recurring schedules live in `rules`
-- (#17, ADR-0001). The old inline-rule columns go, with the checks and
-- indexes on them. Production had no old-format recurring rows left
-- (recurring_type other than one-time with no source_rule_id) when this
-- shipped; refuse to run if any appeared since, rather than lose a schedule.
-- +goose StatementBegin
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM expenses WHERE recurring_type <> 'one-time' AND source_rule_id IS NULL)
       OR EXISTS (SELECT 1 FROM incomes WHERE recurring_type <> 'one-time' AND source_rule_id IS NULL) THEN
        RAISE EXCEPTION 'old-format recurring rows remain; run the worker with --backfill from an older build first';
    END IF;
END $$;
-- +goose StatementEnd

ALTER TABLE expenses DROP COLUMN recurring_type, DROP COLUMN start_date, DROP COLUMN end_date;
ALTER TABLE incomes DROP COLUMN recurring_type, DROP COLUMN start_date, DROP COLUMN end_date;

-- +goose Down
-- Every row comes back as a one-time entry, which is what each row was.
ALTER TABLE expenses
    ADD COLUMN recurring_type character varying(10) DEFAULT 'one-time' NOT NULL,
    ADD COLUMN start_date date,
    ADD COLUMN end_date date,
    ADD CONSTRAINT expenses_recurring_type_check CHECK ((((recurring_type)::text = ANY ((ARRAY['daily', 'weekly', 'monthly', 'yearly', 'one-time'])::text[])) OR (recurring_type IS NULL))),
    ADD CONSTRAINT expenses_recurring_start_date_check CHECK ((recurring_type = 'one-time' OR recurring_type IS NULL) OR (start_date IS NOT NULL));
CREATE INDEX idx_expenses_recurring ON expenses USING btree (recurring_type);

ALTER TABLE incomes
    ADD COLUMN recurring_type character varying(10) DEFAULT 'one-time' NOT NULL,
    ADD COLUMN start_date date,
    ADD COLUMN end_date date,
    ADD CONSTRAINT incomes_recurring_type_check CHECK ((((recurring_type)::text = ANY ((ARRAY['daily', 'weekly', 'monthly', 'yearly', 'one-time'])::text[])) OR (recurring_type IS NULL))),
    ADD CONSTRAINT incomes_recurring_start_date_check CHECK ((recurring_type = 'one-time' OR recurring_type IS NULL) OR (start_date IS NOT NULL));
CREATE INDEX idx_incomes_recurring ON incomes USING btree (recurring_type);
