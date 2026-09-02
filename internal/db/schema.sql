-- Corefinance schema for sqlc
-- Extracted from 20260718-coredb.sql

CREATE EXTENSION IF NOT EXISTS pg_uuidv7;

CREATE TABLE expense_categories (
    id uuid DEFAULT uuid_generate_v7() NOT NULL,
    user_id uuid NOT NULL,
    name character varying(100) NOT NULL,
    color character varying(7),
    icon character varying(50),
    is_active boolean DEFAULT true NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);

CREATE TABLE expense_tags (
    expense_id uuid NOT NULL,
    tag_id uuid NOT NULL
);

CREATE TABLE expenses (
    id uuid DEFAULT uuid_generate_v7() NOT NULL,
    user_id uuid NOT NULL,
    category_id uuid,
    amount numeric(10,2) NOT NULL,
    currency character varying(3) DEFAULT 'PHP' NOT NULL,
    description text NOT NULL,
    notes text,
    expense_date date NOT NULL,
    recurring_type character varying(10) DEFAULT 'one-time' NOT NULL,
    priority character varying(10) DEFAULT 'want' NOT NULL,
    status character varying(10) DEFAULT 'posted' NOT NULL,
    is_debt boolean DEFAULT false NOT NULL,
    start_date date,
    end_date date,
    source_rule_id uuid,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT expenses_amount_check CHECK ((amount > (0)::numeric)),
    CONSTRAINT expenses_priority_check CHECK ((priority)::text = ANY ((ARRAY['need', 'want', 'savings'])::text[])),
    CONSTRAINT expenses_recurring_start_date_check CHECK (((recurring_type IS NULL) OR (start_date IS NOT NULL))),
    CONSTRAINT expenses_recurring_type_check CHECK ((((recurring_type)::text = ANY ((ARRAY['daily', 'weekly', 'monthly', 'yearly', 'one-time'])::text[])) OR (recurring_type IS NULL))),
    CONSTRAINT expenses_status_check CHECK ((status)::text = ANY ((ARRAY['pending', 'posted', 'skipped'])::text[]))
);

CREATE TABLE income_categories (
    id uuid DEFAULT uuid_generate_v7() NOT NULL,
    user_id uuid NOT NULL,
    name character varying(100) NOT NULL,
    color character varying(7),
    icon character varying(50),
    is_active boolean DEFAULT true NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);

CREATE TABLE income_tags (
    income_id uuid NOT NULL,
    tag_id uuid NOT NULL
);

CREATE TABLE incomes (
    id uuid DEFAULT uuid_generate_v7() NOT NULL,
    user_id uuid NOT NULL,
    category_id uuid,
    amount numeric(10,2) NOT NULL,
    currency character varying(3) DEFAULT 'PHP' NOT NULL,
    description text NOT NULL,
    notes text,
    date date NOT NULL,
    recurring_type character varying(10) DEFAULT 'one-time' NOT NULL,
    priority character varying(10) DEFAULT 'want' NOT NULL,
    status character varying(10) DEFAULT 'posted' NOT NULL,
    start_date date,
    end_date date,
    source_rule_id uuid,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT incomes_amount_check CHECK ((amount > (0)::numeric)),
    CONSTRAINT incomes_priority_check CHECK ((priority)::text = ANY ((ARRAY['need', 'want', 'savings'])::text[])),
    CONSTRAINT incomes_recurring_start_date_check CHECK (((recurring_type IS NULL) OR (start_date IS NOT NULL))),
    CONSTRAINT incomes_recurring_type_check CHECK ((((recurring_type)::text = ANY ((ARRAY['daily', 'weekly', 'monthly', 'yearly', 'one-time'])::text[])) OR (recurring_type IS NULL))),
    CONSTRAINT incomes_status_check CHECK ((status)::text = ANY ((ARRAY['pending', 'posted', 'skipped'])::text[]))
);

CREATE TABLE recurring_expense_rules (
    id uuid DEFAULT uuid_generate_v7() NOT NULL,
    user_id uuid NOT NULL,
    description text NOT NULL,
    amount numeric(10,2) NOT NULL,
    currency character varying(3) DEFAULT 'PHP' NOT NULL,
    category_id uuid,
    notes text,
    recurring_type character varying(10) NOT NULL,
    start_date date NOT NULL,
    end_date date,
    priority character varying(10) DEFAULT 'want' NOT NULL,
    is_active boolean DEFAULT true NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT recurring_expense_rules_amount_check CHECK ((amount > (0)::numeric)),
    CONSTRAINT recurring_expense_rules_date_range_check CHECK (((end_date IS NULL) OR (start_date <= end_date))),
    CONSTRAINT recurring_expense_rules_priority_check CHECK ((priority)::text = ANY ((ARRAY['need', 'want', 'savings'])::text[])),
    CONSTRAINT recurring_expense_rules_recurring_type_check CHECK ((recurring_type)::text = ANY ((ARRAY['daily', 'weekly', 'monthly', 'yearly'])::text[]))
);

CREATE TABLE recurring_income_rules (
    id uuid DEFAULT uuid_generate_v7() NOT NULL,
    user_id uuid NOT NULL,
    amount numeric(10,2) NOT NULL,
    currency character varying(3) DEFAULT 'PHP' NOT NULL,
    description text NOT NULL,
    recurring_type character varying(10) NOT NULL,
    start_date date NOT NULL,
    end_date date,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT recurring_income_rules_amount_check CHECK ((amount > (0)::numeric)),
    CONSTRAINT recurring_income_rules_date_range_check CHECK (((end_date IS NULL) OR (start_date <= end_date))),
    CONSTRAINT recurring_income_rules_recurring_type_check CHECK ((recurring_type)::text = ANY ((ARRAY['daily', 'weekly', 'monthly', 'yearly'])::text[]))
);

CREATE TABLE tags (
    id uuid DEFAULT uuid_generate_v7() NOT NULL,
    user_id uuid NOT NULL,
    name character varying(100) NOT NULL,
    color character varying(7),
    created_at timestamp with time zone DEFAULT now() NOT NULL
);

-- Primary keys
ALTER TABLE expense_categories ADD CONSTRAINT expense_categories_pkey PRIMARY KEY (id);
ALTER TABLE expense_tags ADD CONSTRAINT expense_tags_pkey PRIMARY KEY (expense_id, tag_id);
ALTER TABLE expenses ADD CONSTRAINT expenses_pkey PRIMARY KEY (id);
ALTER TABLE income_categories ADD CONSTRAINT income_categories_pkey PRIMARY KEY (id);
ALTER TABLE income_tags ADD CONSTRAINT income_tags_pkey PRIMARY KEY (income_id, tag_id);
ALTER TABLE incomes ADD CONSTRAINT incomes_pkey PRIMARY KEY (id);
ALTER TABLE recurring_expense_rules ADD CONSTRAINT recurring_expense_rules_pkey PRIMARY KEY (id);
ALTER TABLE recurring_income_rules ADD CONSTRAINT recurring_income_rules_pkey PRIMARY KEY (id);
ALTER TABLE tags ADD CONSTRAINT tags_pkey PRIMARY KEY (id);

-- Foreign keys
ALTER TABLE expense_tags ADD CONSTRAINT expense_tags_expense_id_fkey FOREIGN KEY (expense_id) REFERENCES expenses(id) ON DELETE CASCADE;
ALTER TABLE expense_tags ADD CONSTRAINT expense_tags_tag_id_fkey FOREIGN KEY (tag_id) REFERENCES tags(id) ON DELETE CASCADE;
ALTER TABLE income_tags ADD CONSTRAINT income_tags_income_id_fkey FOREIGN KEY (income_id) REFERENCES incomes(id) ON DELETE CASCADE;
ALTER TABLE income_tags ADD CONSTRAINT income_tags_tag_id_fkey FOREIGN KEY (tag_id) REFERENCES tags(id) ON DELETE CASCADE;

-- Indexes
CREATE INDEX idx_expenses_category ON expenses USING btree (category_id);
CREATE INDEX idx_expenses_date ON expenses USING btree (expense_date);
CREATE INDEX idx_expenses_recurring ON expenses USING btree (recurring_type);
CREATE INDEX idx_expenses_user ON expenses USING btree (user_id);
CREATE INDEX idx_incomes_category ON incomes USING btree (category_id);
CREATE INDEX idx_incomes_date ON incomes USING btree (date);
CREATE INDEX idx_incomes_recurring ON incomes USING btree (recurring_type);
CREATE INDEX idx_incomes_user ON incomes USING btree (user_id);
CREATE INDEX idx_recurring_expense_rules_user ON recurring_expense_rules USING btree (user_id);
CREATE INDEX idx_recurring_income_rules_user ON recurring_income_rules USING btree (user_id);
