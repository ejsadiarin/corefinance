-- +goose Up
-- When the user last edited or skipped an entry through the API (#12, ADR-0001).
-- NULL means nobody touched it: a rule entry still as the writer created it.
-- Reconciling a rule's entries after a schedule change skips untouched ones
-- and reports edited ones instead.

ALTER TABLE expenses ADD COLUMN IF NOT EXISTS user_edited_at timestamptz;
ALTER TABLE incomes ADD COLUMN IF NOT EXISTS user_edited_at timestamptz;

-- +goose Down
ALTER TABLE incomes DROP COLUMN IF EXISTS user_edited_at;
ALTER TABLE expenses DROP COLUMN IF EXISTS user_edited_at;
