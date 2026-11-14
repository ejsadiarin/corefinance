-- +goose Up
-- Income rules gain the category, notes and priority that expense rules
-- already have, so the instances they create are categorized like any other
-- income. category_id is unchecked, as on recurring_expense_rules. Existing
-- rules get priority 'want', the value their instances were already stamped
-- with.

ALTER TABLE recurring_income_rules
    ADD COLUMN IF NOT EXISTS category_id uuid,
    ADD COLUMN IF NOT EXISTS notes text,
    ADD COLUMN IF NOT EXISTS priority varchar(10) NOT NULL DEFAULT 'want',
    ADD CONSTRAINT recurring_income_rules_priority_check
        CHECK ((priority)::text = ANY ((ARRAY['need', 'want', 'savings'])::text[]));

-- +goose Down
ALTER TABLE recurring_income_rules
    DROP CONSTRAINT IF EXISTS recurring_income_rules_priority_check,
    DROP COLUMN IF EXISTS priority,
    DROP COLUMN IF EXISTS notes,
    DROP COLUMN IF EXISTS category_id;
