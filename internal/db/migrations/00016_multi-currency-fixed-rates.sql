-- +goose Up
-- Multi-currency records (#3, ADR-0003). Every budget record keeps its
-- original amount and currency, plus the PHP amount fixed at the record's
-- rate date. Existing rows are PHP at rate 1; the one known 2026-07-12 USD
-- gift is converted at its actual 2026-07-10 rate.

ALTER TABLE expenses
    ADD COLUMN exchange_rate numeric(18,8),
    ADD COLUMN amount_php numeric(14,2),
    ADD COLUMN rate_date date;

ALTER TABLE incomes
    ADD COLUMN exchange_rate numeric(18,8),
    ADD COLUMN amount_php numeric(14,2),
    ADD COLUMN rate_date date;

-- Earlier code stored the currency verbatim; normalize it before deciding
-- which rows are PHP. A blank code was the PHP default in practice.
UPDATE expenses
SET currency = CASE WHEN btrim(currency) = '' THEN 'PHP' ELSE upper(btrim(currency)) END;

UPDATE incomes
SET currency = CASE WHEN btrim(currency) = '' THEN 'PHP' ELSE upper(btrim(currency)) END;

UPDATE expenses
SET exchange_rate = 1, amount_php = amount, rate_date = expense_date
WHERE currency = 'PHP';

UPDATE incomes
SET exchange_rate = 1, amount_php = amount, rate_date = date
WHERE currency = 'PHP';

-- The migration cannot invent a historical rate for any other non-PHP row.
-- +goose StatementBegin
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM expenses WHERE currency <> 'PHP')
       OR EXISTS (SELECT 1 FROM incomes WHERE currency <> 'PHP') THEN
        RAISE EXCEPTION 'multi-currency migration: unexpected non-PHP row; convert it before migrating';
    END IF;
END $$;
-- +goose StatementEnd

-- The one known production gift: 2026-07-12, notes naming USD 100. It is
-- stored today as PHP 100 with the USD amount in its notes. Convert it to
-- USD 100 at the ECB rate published for 2026-07-10 (61.573), giving
-- PHP 6,157.30. Match either table; skip silently when the row is absent.
-- +goose StatementBegin
DO $$
DECLARE
    inc_id uuid;
    exp_id uuid;
    inc_count int;
    exp_count int;
BEGIN
    SELECT count(*), (array_agg(id))[1] INTO inc_count, inc_id
    FROM incomes
    WHERE date = '2026-07-12'
      AND currency = 'PHP'
      AND notes ~* 'USD[[:space:]]*100([^0-9]|$)';

    SELECT count(*), (array_agg(id))[1] INTO exp_count, exp_id
    FROM expenses
    WHERE expense_date = '2026-07-12'
      AND currency = 'PHP'
      AND notes ~* 'USD[[:space:]]*100([^0-9]|$)';

    IF inc_count + exp_count > 1 THEN
        RAISE EXCEPTION 'multi-currency migration: more than one 2026-07-12 USD 100 gift';
    END IF;

    IF inc_count = 1 THEN
        UPDATE incomes
        SET amount = 100, currency = 'USD', exchange_rate = 61.573,
            amount_php = 6157.30, rate_date = '2026-07-10'
        WHERE id = inc_id;
    END IF;

    IF exp_count = 1 THEN
        UPDATE expenses
        SET amount = 100, currency = 'USD', exchange_rate = 61.573,
            amount_php = 6157.30, rate_date = '2026-07-10'
        WHERE id = exp_id;
    END IF;
END $$;
-- +goose StatementEnd

ALTER TABLE expenses
    ALTER COLUMN exchange_rate SET NOT NULL,
    ALTER COLUMN amount_php SET NOT NULL,
    ALTER COLUMN rate_date SET NOT NULL,
    ADD CONSTRAINT expenses_exchange_rate_check CHECK (exchange_rate > 0),
    ADD CONSTRAINT expenses_amount_php_check CHECK (amount_php > 0);

ALTER TABLE incomes
    ALTER COLUMN exchange_rate SET NOT NULL,
    ALTER COLUMN amount_php SET NOT NULL,
    ALTER COLUMN rate_date SET NOT NULL,
    ADD CONSTRAINT incomes_exchange_rate_check CHECK (exchange_rate > 0),
    ADD CONSTRAINT incomes_amount_php_check CHECK (amount_php > 0);

-- +goose Down
-- Put the converted gift back the way this migration found it, so a later
-- Up does not trip its own non-PHP guard.
UPDATE incomes
SET amount = 100, currency = 'PHP'
WHERE date = '2026-07-12' AND currency = 'USD'
  AND exchange_rate = 61.573 AND amount_php = 6157.30;

UPDATE expenses
SET amount = 100, currency = 'PHP'
WHERE expense_date = '2026-07-12' AND currency = 'USD'
  AND exchange_rate = 61.573 AND amount_php = 6157.30;

ALTER TABLE incomes
    DROP CONSTRAINT IF EXISTS incomes_exchange_rate_check,
    DROP CONSTRAINT IF EXISTS incomes_amount_php_check,
    DROP COLUMN IF EXISTS exchange_rate,
    DROP COLUMN IF EXISTS amount_php,
    DROP COLUMN IF EXISTS rate_date;

ALTER TABLE expenses
    DROP CONSTRAINT IF EXISTS expenses_exchange_rate_check,
    DROP CONSTRAINT IF EXISTS expenses_amount_php_check,
    DROP COLUMN IF EXISTS exchange_rate,
    DROP COLUMN IF EXISTS amount_php,
    DROP COLUMN IF EXISTS rate_date;
