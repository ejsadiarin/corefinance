-- Service endpoints (cross-user reads for service identities).
-- These back scope-gated /internal/* routes called with service JWTs;
-- they intentionally carry no user filter. Row-level shape mirrors the
-- user-scoped recurring rule listings minus ownership checks.

-- name: ListActiveExpenseRulesAllUsers :many
SELECT id, user_id, description, amount, currency, recurring_type,
    start_date, end_date
FROM recurring_expense_rules
WHERE is_active = true
ORDER BY start_date ASC;

-- name: ListActiveIncomeRulesAllUsers :many
SELECT id, user_id, description, amount, currency, recurring_type,
    start_date, end_date
FROM recurring_income_rules
WHERE is_active = true
ORDER BY start_date ASC;
