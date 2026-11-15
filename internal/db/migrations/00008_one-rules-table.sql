-- +goose Up
-- One rules table for expense and income rules (#11, ADR-0001). The old table
-- names stay as views over it, one per kind, with their original column
-- order, so existing queries, generated models and the rule endpoints keep
-- working unchanged. New code reads and writes `rules` directly; the views go
-- away once nothing uses them.

CREATE TABLE rules (
    id uuid DEFAULT uuid_generate_v7() NOT NULL,
    kind varchar(7) NOT NULL,
    user_id uuid NOT NULL,
    description text NOT NULL,
    amount numeric(10,2) NOT NULL,
    currency varchar(3) DEFAULT 'PHP' NOT NULL,
    category_id uuid,
    notes text,
    recurring_type varchar(10) NOT NULL,
    start_date date NOT NULL,
    end_date date,
    priority varchar(10) DEFAULT 'want' NOT NULL,
    is_active boolean DEFAULT true NOT NULL,
    materialized_through date,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT rules_pkey PRIMARY KEY (id),
    CONSTRAINT rules_kind_check CHECK ((kind)::text = ANY ((ARRAY['expense', 'income'])::text[])),
    CONSTRAINT rules_amount_check CHECK ((amount > (0)::numeric)),
    CONSTRAINT rules_date_range_check CHECK (((end_date IS NULL) OR (start_date <= end_date))),
    CONSTRAINT rules_priority_check CHECK ((priority)::text = ANY ((ARRAY['need', 'want', 'savings'])::text[])),
    CONSTRAINT rules_recurring_type_check CHECK ((recurring_type)::text = ANY ((ARRAY['daily', 'weekly', 'monthly', 'yearly'])::text[]))
);
CREATE INDEX idx_rules_user_kind ON rules USING btree (user_id, kind);

INSERT INTO rules (id, kind, user_id, description, amount, currency, category_id, notes, recurring_type,
                   start_date, end_date, priority, is_active, materialized_through, created_at, updated_at)
SELECT id, 'expense', user_id, description, amount, currency, category_id, notes, recurring_type,
       start_date, end_date, priority, is_active, materialized_through, created_at, updated_at
FROM recurring_expense_rules;

INSERT INTO rules (id, kind, user_id, description, amount, currency, category_id, notes, recurring_type,
                   start_date, end_date, priority, is_active, materialized_through, created_at, updated_at)
SELECT id, 'income', user_id, description, amount, currency, category_id, notes, recurring_type,
       start_date, end_date, priority, is_active, materialized_through, created_at, updated_at
FROM recurring_income_rules;

DROP TABLE recurring_expense_rules;
DROP TABLE recurring_income_rules;

-- Column order matches the old tables; kind is last and defaults to the
-- view's kind, so an INSERT through the view needs no change.
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

-- +goose Down
DROP VIEW IF EXISTS recurring_expense_rules;
DROP VIEW IF EXISTS recurring_income_rules;

CREATE TABLE recurring_expense_rules AS
SELECT id, user_id, description, amount, currency, category_id, notes, recurring_type, start_date,
       end_date, priority, is_active, created_at, updated_at, materialized_through
FROM rules WHERE kind = 'expense';
CREATE TABLE recurring_income_rules AS
SELECT id, user_id, amount, currency, description, recurring_type, start_date, end_date, created_at,
       updated_at, is_active, category_id, notes, priority, materialized_through
FROM rules WHERE kind = 'income';

ALTER TABLE recurring_expense_rules
    ALTER COLUMN id SET DEFAULT uuid_generate_v7(), ALTER COLUMN id SET NOT NULL,
    ALTER COLUMN user_id SET NOT NULL, ALTER COLUMN description SET NOT NULL,
    ALTER COLUMN amount SET NOT NULL, ALTER COLUMN currency SET DEFAULT 'PHP', ALTER COLUMN currency SET NOT NULL,
    ALTER COLUMN recurring_type SET NOT NULL, ALTER COLUMN start_date SET NOT NULL,
    ALTER COLUMN priority SET DEFAULT 'want', ALTER COLUMN priority SET NOT NULL,
    ALTER COLUMN is_active SET DEFAULT true, ALTER COLUMN is_active SET NOT NULL,
    ALTER COLUMN created_at SET DEFAULT now(), ALTER COLUMN created_at SET NOT NULL,
    ALTER COLUMN updated_at SET DEFAULT now(), ALTER COLUMN updated_at SET NOT NULL,
    ADD CONSTRAINT recurring_expense_rules_pkey PRIMARY KEY (id),
    ADD CONSTRAINT recurring_expense_rules_amount_check CHECK ((amount > (0)::numeric)),
    ADD CONSTRAINT recurring_expense_rules_date_range_check CHECK (((end_date IS NULL) OR (start_date <= end_date))),
    ADD CONSTRAINT recurring_expense_rules_priority_check CHECK ((priority)::text = ANY ((ARRAY['need', 'want', 'savings'])::text[])),
    ADD CONSTRAINT recurring_expense_rules_recurring_type_check CHECK ((recurring_type)::text = ANY ((ARRAY['daily', 'weekly', 'monthly', 'yearly'])::text[]));
CREATE INDEX idx_recurring_expense_rules_user ON recurring_expense_rules USING btree (user_id);

ALTER TABLE recurring_income_rules
    ALTER COLUMN id SET DEFAULT uuid_generate_v7(), ALTER COLUMN id SET NOT NULL,
    ALTER COLUMN user_id SET NOT NULL, ALTER COLUMN amount SET NOT NULL,
    ALTER COLUMN currency SET DEFAULT 'PHP', ALTER COLUMN currency SET NOT NULL,
    ALTER COLUMN description SET NOT NULL, ALTER COLUMN recurring_type SET NOT NULL,
    ALTER COLUMN start_date SET NOT NULL,
    ALTER COLUMN created_at SET DEFAULT now(), ALTER COLUMN created_at SET NOT NULL,
    ALTER COLUMN updated_at SET DEFAULT now(), ALTER COLUMN updated_at SET NOT NULL,
    ALTER COLUMN is_active SET DEFAULT true, ALTER COLUMN is_active SET NOT NULL,
    ALTER COLUMN priority SET DEFAULT 'want', ALTER COLUMN priority SET NOT NULL,
    ADD CONSTRAINT recurring_income_rules_pkey PRIMARY KEY (id),
    ADD CONSTRAINT recurring_income_rules_amount_check CHECK ((amount > (0)::numeric)),
    ADD CONSTRAINT recurring_income_rules_date_range_check CHECK (((end_date IS NULL) OR (start_date <= end_date))),
    ADD CONSTRAINT recurring_income_rules_priority_check CHECK ((priority)::text = ANY ((ARRAY['need', 'want', 'savings'])::text[])),
    ADD CONSTRAINT recurring_income_rules_recurring_type_check CHECK ((recurring_type)::text = ANY ((ARRAY['daily', 'weekly', 'monthly', 'yearly'])::text[]));
CREATE INDEX idx_recurring_income_rules_user ON recurring_income_rules USING btree (user_id);

DROP TABLE rules;
