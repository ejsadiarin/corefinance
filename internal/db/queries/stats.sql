-- name: GetSummary :one
SELECT
    COALESCE(SUM(CASE WHEN e.status != 'skipped' THEN e.amount ELSE 0 END), 0)::numeric AS total_expenses,
    COALESCE(SUM(CASE WHEN i.status != 'skipped' THEN i.amount ELSE 0 END), 0)::numeric AS total_incomes,
    COUNT(DISTINCT e.id) FILTER (WHERE e.status != 'skipped') AS expense_count,
    COUNT(DISTINCT i.id) FILTER (WHERE i.status != 'skipped') AS income_count
FROM expenses e
FULL OUTER JOIN incomes i ON e.user_id = i.user_id
WHERE (e.user_id = @user_id OR i.user_id = @user_id)
  AND (e.expense_date >= @start_date OR e.expense_date IS NULL OR @start_date IS NULL)
  AND (e.expense_date <= @end_date OR e.expense_date IS NULL OR @end_date IS NULL)
  AND (i.date >= @start_date OR i.date IS NULL OR @start_date IS NULL)
  AND (i.date <= @end_date OR i.date IS NULL OR @end_date IS NULL);

-- name: GetTrends :many
SELECT
    TO_CHAR(COALESCE(e.expense_date, i.date), 'YYYY-MM') AS month,
    COALESCE(SUM(CASE WHEN e.status != 'skipped' THEN e.amount ELSE 0 END), 0)::numeric AS total_expenses,
    COALESCE(SUM(CASE WHEN i.status != 'skipped' THEN i.amount ELSE 0 END), 0)::numeric AS total_incomes
FROM expenses e
FULL OUTER JOIN incomes i ON e.user_id = i.user_id AND TO_CHAR(COALESCE(e.expense_date, i.date), 'YYYY-MM') = TO_CHAR(i.date, 'YYYY-MM')
WHERE (e.user_id = @user_id OR i.user_id = @user_id)
  AND (e.expense_date >= @start_date OR e.expense_date IS NULL OR @start_date IS NULL)
  AND (e.expense_date <= @end_date OR e.expense_date IS NULL OR @end_date IS NULL)
  AND (i.date >= @start_date OR i.date IS NULL OR @start_date IS NULL)
  AND (i.date <= @end_date OR i.date IS NULL OR @end_date IS NULL)
GROUP BY TO_CHAR(COALESCE(e.expense_date, i.date), 'YYYY-MM')
ORDER BY month DESC;

-- name: GetCategoryBreakdown :many
SELECT
    ec.id AS category_id,
    ec.name AS category_name,
    ec.color AS category_color,
    COUNT(e.id) AS expense_count,
    COALESCE(SUM(e.amount), 0)::numeric AS total_amount
FROM expense_categories ec
LEFT JOIN expenses e ON ec.id = e.category_id
    AND e.user_id = @user_id
    AND e.status != 'skipped'
    AND (@start_date::date IS NULL OR e.expense_date >= @start_date)
    AND (@end_date::date IS NULL OR e.expense_date <= @end_date)
WHERE ec.user_id = @user_id AND ec.is_active = true
GROUP BY ec.id, ec.name, ec.color
ORDER BY total_amount DESC;

-- name: GetSavingsRate :one
WITH totals AS (
    SELECT
        COALESCE(SUM(CASE WHEN e.status != 'skipped' THEN e.amount ELSE 0 END), 0)::numeric AS total_expenses,
        COALESCE(SUM(CASE WHEN i.status != 'skipped' THEN i.amount ELSE 0 END), 0)::numeric AS total_incomes
    FROM expenses e
    FULL OUTER JOIN incomes i ON e.user_id = i.user_id
    WHERE (e.user_id = @user_id OR i.user_id = @user_id)
      AND (e.expense_date >= @start_date OR e.expense_date IS NULL OR @start_date IS NULL)
      AND (e.expense_date <= @end_date OR e.expense_date IS NULL OR @end_date IS NULL)
      AND (i.date >= @start_date OR i.date IS NULL OR @start_date IS NULL)
      AND (i.date <= @end_date OR i.date IS NULL OR @end_date IS NULL)
)
SELECT
    total_incomes,
    total_expenses,
    CASE
        WHEN total_incomes = 0 THEN 0
        ELSE ((total_incomes - total_expenses) / total_incomes * 100)::numeric
    END AS savings_rate
FROM totals;

-- name: GetFiftyThirtyTwenty :one
WITH totals AS (
    SELECT
        COALESCE(SUM(CASE WHEN e.status != 'skipped' THEN e.amount ELSE 0 END), 0)::numeric AS total_expenses,
        COALESCE(SUM(CASE WHEN i.status != 'skipped' THEN i.amount ELSE 0 END), 0)::numeric AS total_incomes
    FROM expenses e
    FULL OUTER JOIN incomes i ON e.user_id = i.user_id
    WHERE (e.user_id = @user_id OR i.user_id = @user_id)
      AND (e.expense_date >= @start_date OR e.expense_date IS NULL OR @start_date IS NULL)
      AND (e.expense_date <= @end_date OR e.expense_date IS NULL OR @end_date IS NULL)
      AND (i.date >= @start_date OR i.date IS NULL OR @start_date IS NULL)
      AND (i.date <= @end_date OR i.date IS NULL OR @end_date IS NULL)
),
category_totals AS (
    SELECT
        COALESCE(SUM(CASE WHEN e.priority = 'need' AND e.status != 'skipped' THEN e.amount ELSE 0 END), 0)::numeric AS needs,
        COALESCE(SUM(CASE WHEN e.priority = 'want' AND e.status != 'skipped' THEN e.amount ELSE 0 END), 0)::numeric AS wants,
        COALESCE(SUM(CASE WHEN e.priority = 'savings' AND e.status != 'skipped' THEN e.amount ELSE 0 END), 0)::numeric AS savings
    FROM expenses e
    WHERE e.user_id = @user_id
      AND (@start_date::date IS NULL OR e.expense_date >= @start_date)
      AND (@end_date::date IS NULL OR e.expense_date <= @end_date)
)
SELECT
    t.total_incomes,
    t.total_expenses,
    c.needs,
    c.wants,
    c.savings,
    CASE WHEN t.total_expenses = 0 THEN 0 ELSE (c.needs / t.total_expenses * 100)::numeric END AS needs_pct,
    CASE WHEN t.total_expenses = 0 THEN 0 ELSE (c.wants / t.total_expenses * 100)::numeric END AS wants_pct,
    CASE WHEN t.total_expenses = 0 THEN 0 ELSE (c.savings / t.total_expenses * 100)::numeric END AS savings_pct
FROM totals t, category_totals c;

-- name: GetUpcomingRecurringExpenses :many
SELECT r.*, ec.name AS category_name, ec.color AS category_color
FROM recurring_expense_rules r
LEFT JOIN expense_categories ec ON r.category_id = ec.id
WHERE r.user_id = @user_id
  AND r.is_active = true
  AND (r.end_date IS NULL OR r.end_date >= CURRENT_DATE)
ORDER BY r.created_at ASC;

-- name: GetSpendingVelocity :one
SELECT
    COALESCE(AVG(monthly_total), 0)::numeric AS avg_monthly_spending,
    COUNT(*)::int AS months_with_data
FROM (
    SELECT
        DATE_TRUNC('month', e.expense_date) AS month,
        SUM(e.amount)::numeric AS monthly_total
    FROM expenses e
    WHERE e.user_id = @user_id
      AND e.expense_date >= @start_date
      AND e.expense_date <= @end_date
      AND e.status != 'skipped'
    GROUP BY DATE_TRUNC('month', e.expense_date)
    ORDER BY month
) AS monthly;

-- name: GetCurrentTotalMoney :one
SELECT
    (COALESCE(SUM(CASE WHEN i.status != 'skipped' THEN i.amount ELSE 0 END), 0)
     - COALESCE(SUM(CASE WHEN e.status != 'skipped' THEN e.amount ELSE 0 END), 0))::numeric AS total_money
FROM expenses e
FULL OUTER JOIN incomes i ON e.user_id = i.user_id
WHERE (e.user_id = @user_id OR i.user_id = @user_id)
  AND (e.expense_date >= @start_date OR e.expense_date IS NULL OR @start_date IS NULL)
  AND (e.expense_date <= @end_date OR e.expense_date IS NULL OR @end_date IS NULL)
  AND (i.date >= @start_date OR i.date IS NULL OR @start_date IS NULL)
  AND (i.date <= @end_date OR i.date IS NULL OR @end_date IS NULL);

-- name: GetMonthOverMonthTrends :many
SELECT
    month,
    SUM(total)::numeric AS total_amount
FROM (
    SELECT
        TO_CHAR(e.expense_date, 'YYYY-MM') AS month,
        COALESCE(SUM(e.amount), 0)::numeric AS total
    FROM expenses e
    WHERE e.user_id = @user_id
      AND e.expense_date >= @start_date
      AND e.expense_date <= @end_date
      AND e.status != 'skipped'
    GROUP BY TO_CHAR(e.expense_date, 'YYYY-MM')
    UNION ALL
    SELECT
        TO_CHAR(i.date, 'YYYY-MM') AS month,
        COALESCE(SUM(i.amount), 0)::numeric AS total
    FROM incomes i
    WHERE i.user_id = @user_id
      AND i.date >= @start_date
      AND i.date <= @end_date
      AND i.status != 'skipped'
    GROUP BY TO_CHAR(i.date, 'YYYY-MM')
) AS monthly_data
GROUP BY month
ORDER BY month DESC;
