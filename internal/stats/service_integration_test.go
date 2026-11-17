package stats

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ejsadiarin/corefinance/internal/auth"
	"github.com/ejsadiarin/corefinance/internal/category"
	db "github.com/ejsadiarin/corefinance/internal/db/sqlc"
	"github.com/ejsadiarin/corefinance/internal/expense"
	"github.com/ejsadiarin/corefinance/internal/income"
	"github.com/ejsadiarin/corefinance/internal/rates"
	"github.com/ejsadiarin/corefinance/internal/testutil"
)

// tripRates is the synthetic provider for the November 2026 Japan trip
// fixture; it never touches the network.
type tripRates struct{}

func (tripRates) Rate(_ context.Context, currency string, date time.Time) (rates.Rate, error) {
	switch currency {
	case "USD":
		return rates.Rate{Value: decimal.RequireFromString("63.813"), Date: date}, nil
	case "JPY":
		return rates.Rate{Value: decimal.RequireFromString("0.39884"), Date: date}, nil
	}
	return rates.Rate{}, rates.ErrUnsupportedCurrency
}

func newStatsIntegrationService(t *testing.T) (*Service, *pgxpool.Pool) {
	t.Helper()
	pool := testutil.SetupTestDB(t)
	return NewService(db.New(pool)), pool
}

func seedExpense(t *testing.T, ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID, amount, date, status string) {
	t.Helper()
	_, err := pool.Exec(ctx, `INSERT INTO expenses
		(user_id, amount, currency, exchange_rate, amount_php, rate_date, description, expense_date, priority, status)
		VALUES ($1::uuid, $2::numeric, 'PHP', 1, $2::numeric, $3::date, 'test expense', $3::date, 'want', $4)`,
		userID.String(), amount, date, status)
	require.NoError(t, err)
}

func seedIncome(t *testing.T, ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID, amount, date, status string) {
	t.Helper()
	_, err := pool.Exec(ctx, `INSERT INTO incomes
		(user_id, amount, currency, exchange_rate, amount_php, rate_date, description, date, priority, status)
		VALUES ($1::uuid, $2::numeric, 'PHP', 1, $2::numeric, $3::date, 'test income', $3::date, 'want', $4)`,
		userID.String(), amount, date, status)
	require.NoError(t, err)
}

func seedIncomeInCategory(t *testing.T, ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID, amount, date, status string, categoryID *string) {
	t.Helper()
	var cat any
	if categoryID != nil {
		cat = *categoryID
	}
	_, err := pool.Exec(ctx, `INSERT INTO incomes
		(user_id, amount, currency, exchange_rate, amount_php, rate_date, description, date, priority, status, category_id)
		VALUES ($1::uuid, $2::numeric, 'PHP', 1, $2::numeric, $3::date, 'test income', $3::date, 'want', $4, $5::uuid)`,
		userID.String(), amount, date, status, cat)
	require.NoError(t, err)
}

// Spec (#24): income filed under the reimbursement category is left out of
// every report's income, reported as its own figure, and the savings rate is
// computed from the rest. A skipped reimbursement counts nowhere, an income
// with no category is ordinary income, and money on hand still counts
// reimbursements because the money arrived (ADR-0002, decision 3).
func TestStatsService_Reimbursements_Integration(t *testing.T) {
	svc, pool := newStatsIntegrationService(t)
	ctx := context.Background()

	salary, found := category.ByName(category.Incomes, "salary")
	require.True(t, found)
	salaryID := salary.ID.String()
	reimbursementID := category.Reimbursement.String()

	userID := uuid.New()
	seedIncomeInCategory(t, ctx, pool, userID, "20000", "2026-04-01", "posted", &salaryID)
	seedIncomeInCategory(t, ctx, pool, userID, "500", "2026-04-10", "posted", &reimbursementID)
	seedIncomeInCategory(t, ctx, pool, userID, "777", "2026-04-15", "skipped", &reimbursementID)
	seedExpense(t, ctx, pool, userID, "8000", "2026-04-05", "posted")

	start, end := "2026-04-01", "2026-04-30"

	summary, err := svc.Summary(ctx, userID, &start, &end)
	require.NoError(t, err)
	require.True(t, decimal.NewFromInt(8000).Equal(summary.TotalExpenses), "expected total_expenses=8000, got %s", summary.TotalExpenses.String())
	require.True(t, decimal.NewFromInt(20000).Equal(summary.TotalIncomes), "expected total_incomes=20000, got %s", summary.TotalIncomes.String())
	require.True(t, decimal.NewFromInt(500).Equal(summary.TotalReimbursements), "expected total_reimbursements=500, got %s", summary.TotalReimbursements.String())
	require.Equal(t, int64(1), summary.IncomeCount)

	trends, err := svc.Trends(ctx, userID, &start, &end)
	require.NoError(t, err)
	require.Len(t, trends, 1)
	require.Equal(t, "2026-04", trends[0].Month)
	require.True(t, decimal.NewFromInt(8000).Equal(trends[0].TotalExpenses), "expected total_expenses=8000 for 2026-04, got %s", trends[0].TotalExpenses.String())
	require.True(t, decimal.NewFromInt(20000).Equal(trends[0].TotalIncomes), "expected total_incomes=20000 for 2026-04, got %s", trends[0].TotalIncomes.String())
	require.True(t, decimal.NewFromInt(500).Equal(trends[0].TotalReimbursements), "expected total_reimbursements=500 for 2026-04, got %s", trends[0].TotalReimbursements.String())

	rate, err := svc.SavingsRate(ctx, userID, &start, &end)
	require.NoError(t, err)
	require.True(t, decimal.NewFromInt(20000).Equal(rate.TotalIncomes), "expected total_incomes=20000, got %s", rate.TotalIncomes.String())
	require.True(t, decimal.NewFromInt(8000).Equal(rate.TotalExpenses), "expected total_expenses=8000, got %s", rate.TotalExpenses.String())
	require.True(t, decimal.NewFromInt(500).Equal(rate.TotalReimbursements), "expected total_reimbursements=500, got %s", rate.TotalReimbursements.String())
	require.True(t, decimal.NewFromInt(60).Equal(rate.SavingsRate), "expected savings_rate=60 from 20000 income, got %s", rate.SavingsRate.String())

	ftt, err := svc.FiftyThirtyTwenty(ctx, userID, &start, &end)
	require.NoError(t, err)
	require.True(t, decimal.NewFromInt(20000).Equal(ftt.TotalIncomes), "expected total_incomes=20000, got %s", ftt.TotalIncomes.String())
	require.True(t, decimal.NewFromInt(500).Equal(ftt.TotalReimbursements), "expected total_reimbursements=500, got %s", ftt.TotalReimbursements.String())

	// Month over month has no reimbursement figure: its income part just
	// leaves reimbursements out (8000 expense + 20000 income, not 28500).
	mom, err := svc.MonthOverMonthTrends(ctx, userID, &start, &end)
	require.NoError(t, err)
	require.Len(t, mom, 1)
	require.Equal(t, "2026-04", mom[0].Month)
	require.True(t, decimal.NewFromInt(28000).Equal(mom[0].TotalAmount), "expected total_amount=28000 for 2026-04, got %s", mom[0].TotalAmount.String())

	// Money on hand still counts reimbursements; a skipped one counts nowhere.
	svc.now = func() time.Time { return time.Date(2026, 12, 31, 12, 0, 0, 0, time.UTC) }
	moneyUser := uuid.New()
	seedIncomeInCategory(t, ctx, pool, moneyUser, "20000", "2026-04-01", "posted", &salaryID)
	seedIncomeInCategory(t, ctx, pool, moneyUser, "500", "2026-04-10", "posted", &reimbursementID)
	seedIncomeInCategory(t, ctx, pool, moneyUser, "777", "2026-04-15", "skipped", &reimbursementID)
	money, err := svc.CurrentTotalMoney(ctx, moneyUser)
	require.NoError(t, err)
	require.True(t, decimal.NewFromInt(20500).Equal(money.TotalMoney), "expected total_money=20500, got %s", money.TotalMoney.String())

	// A NULL category is ordinary income, not a reimbursement.
	plainUser := uuid.New()
	seedIncome(t, ctx, pool, plainUser, "300", "2026-04-20", "posted")
	plainSummary, err := svc.Summary(ctx, plainUser, &start, &end)
	require.NoError(t, err)
	require.True(t, decimal.NewFromInt(300).Equal(plainSummary.TotalIncomes), "expected total_incomes=300, got %s", plainSummary.TotalIncomes.String())
	require.Equal(t, int64(1), plainSummary.IncomeCount)
	require.True(t, decimal.Zero.Equal(plainSummary.TotalReimbursements), "expected total_reimbursements=0, got %s", plainSummary.TotalReimbursements.String())
}

// Spec: 3 posted expenses of 100 + 2 posted incomes of 500 in range must give
// exact totals (300/1000, counts 3/2) — no FULL OUTER JOIN fan-out inflation.
func TestStatsService_Summary_Integration_NoFanOut(t *testing.T) {
	svc, pool := newStatsIntegrationService(t)
	ctx := context.Background()
	userID := uuid.New()

	seedExpense(t, ctx, pool, userID, "100", "2026-01-05", "posted")
	seedExpense(t, ctx, pool, userID, "100", "2026-01-10", "posted")
	seedExpense(t, ctx, pool, userID, "100", "2026-01-15", "posted")
	seedExpense(t, ctx, pool, userID, "50", "2026-02-01", "posted") // out of range
	seedExpense(t, ctx, pool, userID, "999", "2026-01-20", "skipped")
	seedIncome(t, ctx, pool, userID, "500", "2026-01-07", "posted")
	seedIncome(t, ctx, pool, userID, "500", "2026-01-21", "posted")
	seedIncome(t, ctx, pool, userID, "9999", "2026-01-25", "skipped")

	start, end := "2026-01-01", "2026-01-31"
	res, err := svc.Summary(ctx, userID, &start, &end)
	require.NoError(t, err)
	require.True(t, decimal.NewFromInt(300).Equal(res.TotalExpenses), "expected total_expenses=300, got %s", res.TotalExpenses.String())
	require.True(t, decimal.NewFromInt(1000).Equal(res.TotalIncomes), "expected total_incomes=1000, got %s", res.TotalIncomes.String())
	require.Equal(t, int64(3), res.ExpenseCount)
	require.Equal(t, int64(2), res.IncomeCount)
}

// Spec: expenses in 2026-08 but incomes only in 2026-09 must return a row per
// month with the missing side reported as 0.
func TestStatsService_Trends_Integration_SingleSidedMonths(t *testing.T) {
	svc, pool := newStatsIntegrationService(t)
	ctx := context.Background()
	userID := uuid.New()

	seedExpense(t, ctx, pool, userID, "150", "2026-08-15", "posted")
	seedIncome(t, ctx, pool, userID, "2000", "2026-09-10", "posted")

	start, end := "2026-08-01", "2026-09-30"
	rows, err := svc.Trends(ctx, userID, &start, &end)
	require.NoError(t, err)
	require.Len(t, rows, 2)

	require.Equal(t, "2026-09", rows[0].Month)
	require.True(t, decimal.Zero.Equal(rows[0].TotalExpenses), "expected 0 expenses for 2026-09, got %s", rows[0].TotalExpenses.String())
	require.True(t, decimal.NewFromInt(2000).Equal(rows[0].TotalIncomes), "expected total_incomes=2000 for 2026-09, got %s", rows[0].TotalIncomes.String())

	require.Equal(t, "2026-08", rows[1].Month)
	require.True(t, decimal.NewFromInt(150).Equal(rows[1].TotalExpenses), "expected total_expenses=150 for 2026-08, got %s", rows[1].TotalExpenses.String())
	require.True(t, decimal.Zero.Equal(rows[1].TotalIncomes), "expected 0 incomes for 2026-08, got %s", rows[1].TotalIncomes.String())
}

// Spec: 1000 income + 400 expenses in range must give savings_rate=60.
func TestStatsService_SavingsRate_Integration(t *testing.T) {
	svc, pool := newStatsIntegrationService(t)
	ctx := context.Background()
	userID := uuid.New()

	seedIncome(t, ctx, pool, userID, "500", "2026-03-05", "posted")
	seedIncome(t, ctx, pool, userID, "500", "2026-03-12", "posted")
	seedExpense(t, ctx, pool, userID, "100", "2026-03-01", "posted")
	seedExpense(t, ctx, pool, userID, "100", "2026-03-08", "posted")
	seedExpense(t, ctx, pool, userID, "100", "2026-03-15", "posted")
	seedExpense(t, ctx, pool, userID, "100", "2026-03-22", "posted")

	start, end := "2026-03-01", "2026-03-31"
	res, err := svc.SavingsRate(ctx, userID, &start, &end)
	require.NoError(t, err)
	require.True(t, decimal.NewFromInt(1000).Equal(res.TotalIncomes), "expected total_incomes=1000, got %s", res.TotalIncomes.String())
	require.True(t, decimal.NewFromInt(400).Equal(res.TotalExpenses), "expected total_expenses=400, got %s", res.TotalExpenses.String())
	require.True(t, decimal.NewFromInt(60).Equal(res.SavingsRate), "expected savings_rate=60, got %s", res.SavingsRate.String())
}

// Spec: lifetime total ignores period boundaries and skips.
func TestStatsService_CurrentTotalMoney_Integration_Lifetime(t *testing.T) {
	svc, pool := newStatsIntegrationService(t)
	ctx := context.Background()
	userID := uuid.New()

	seedIncome(t, ctx, pool, userID, "1000", "2024-01-01", "posted")
	seedExpense(t, ctx, pool, userID, "200", "2026-09-01", "posted")
	seedIncome(t, ctx, pool, userID, "500", "2026-05-01", "skipped")

	expected := decimal.NewFromInt(800)
	svc.now = func() time.Time { return time.Date(2026, 12, 31, 12, 0, 0, 0, time.UTC) }

	res, err := svc.CurrentTotalMoney(ctx, userID)
	require.NoError(t, err)
	require.True(t, expected.Equal(res.TotalMoney), "expected lifetime total_money=800, got %s", res.TotalMoney.String())
}

// Spec (#25): money on hand counts only records dated today or earlier in
// the owner's timezone. Future-dated expenses are committed, skipped records
// count nowhere, reimbursements count as money, and runway is money on hand
// over the trailing 30-day net spend.
func TestStatsService_MoneyOnHand_Integration(t *testing.T) {
	svc, pool := newStatsIntegrationService(t)
	ctx := auth.WithTimezone(context.Background(), "Asia/Manila")
	salary, found := category.ByName(category.Incomes, "salary")
	require.True(t, found)
	salaryID := salary.ID.String()
	reimbursementID := category.Reimbursement.String()

	t.Run("future expense, skipped records and a reimbursement", func(t *testing.T) {
		// 20:00 on 17 November in Manila.
		svc.now = func() time.Time { return time.Date(2026, 11, 17, 12, 0, 0, 0, time.UTC) }
		userID := uuid.New()
		seedIncomeInCategory(t, ctx, pool, userID, "20000", "2026-11-01", "posted", &salaryID)
		seedIncomeInCategory(t, ctx, pool, userID, "500", "2026-11-10", "posted", &reimbursementID)
		seedExpense(t, ctx, pool, userID, "1000", "2026-11-15", "posted")
		seedExpense(t, ctx, pool, userID, "2716", "2026-12-15", "posted")
		seedExpense(t, ctx, pool, userID, "999", "2026-11-20", "skipped")
		seedExpense(t, ctx, pool, userID, "777", "2026-11-16", "skipped")

		money, err := svc.CurrentTotalMoney(ctx, userID)
		require.NoError(t, err)
		assert.True(t, decimal.NewFromInt(19500).Equal(money.TotalMoney), money.TotalMoney.String())
		assert.True(t, decimal.NewFromInt(2716).Equal(money.Committed), money.Committed.String())
		assert.Equal(t, "16.67", money.AverageDailySpend.String())
		require.NotNil(t, money.RunwayDays)
		assert.Equal(t, int64(1170), *money.RunwayDays)
	})

	t.Run("day boundary follows the owner's zone", func(t *testing.T) {
		// 07:30 on 18 November in Manila, still 17 November in UTC: the
		// expense dated the 18th is today, the one on the 19th is committed.
		svc.now = func() time.Time { return time.Date(2026, 11, 17, 23, 30, 0, 0, time.UTC) }
		userID := uuid.New()
		seedIncome(t, ctx, pool, userID, "1000", "2026-11-01", "posted")
		seedExpense(t, ctx, pool, userID, "200", "2026-11-18", "posted")
		seedExpense(t, ctx, pool, userID, "300", "2026-11-19", "posted")

		money, err := svc.CurrentTotalMoney(ctx, userID)
		require.NoError(t, err)
		assert.True(t, decimal.NewFromInt(800).Equal(money.TotalMoney), money.TotalMoney.String())
		assert.True(t, decimal.NewFromInt(300).Equal(money.Committed), money.Committed.String())
		require.NotNil(t, money.RunwayDays)
		assert.Equal(t, int64(120), *money.RunwayDays)
	})

	t.Run("no seen zone is UTC", func(t *testing.T) {
		svc.now = func() time.Time { return time.Date(2026, 11, 17, 12, 0, 0, 0, time.UTC) }
		userID := uuid.New()
		seedIncome(t, ctx, pool, userID, "1000", "2026-11-17", "posted")
		seedExpense(t, ctx, pool, userID, "200", "2026-11-18", "posted")

		money, err := svc.CurrentTotalMoney(context.Background(), userID)
		require.NoError(t, err)
		assert.True(t, decimal.NewFromInt(1000).Equal(money.TotalMoney), money.TotalMoney.String())
		assert.True(t, decimal.NewFromInt(200).Equal(money.Committed), money.Committed.String())
		assert.Nil(t, money.RunwayDays)
	})
}

// Spec (#26): a period budget is funded by the period's income, spent only
// through today, and holds future expenses as bills still due. Reimbursements
// reduce spent, skipped records count nowhere.
func TestStatsService_PeriodBudget_Integration(t *testing.T) {
	svc, pool := newStatsIntegrationService(t)
	svc.now = func() time.Time { return time.Date(2026, 11, 18, 12, 0, 0, 0, time.UTC) }
	ctx := auth.WithTimezone(context.Background(), "Asia/Manila")
	reimbursementID := category.Reimbursement.String()

	t.Run("day nets reimbursements and excludes skipped records", func(t *testing.T) {
		userID := uuid.New()
		salary, found := category.ByName(category.Incomes, "salary")
		require.True(t, found)
		salaryID := salary.ID.String()
		seedIncome(t, ctx, pool, userID, "1000", "2026-11-18", "posted")
		seedIncomeInCategory(t, ctx, pool, userID, "200", "2026-11-18", "posted", &reimbursementID)
		// Pending income is not funding, but a reimbursement still nets out
		// of spent while pending.
		seedIncome(t, ctx, pool, userID, "700", "2026-11-18", "pending")
		seedIncomeInCategory(t, ctx, pool, userID, "50", "2026-11-18", "pending", &reimbursementID)
		seedExpense(t, ctx, pool, userID, "300", "2026-11-18", "posted")
		seedExpense(t, ctx, pool, userID, "999", "2026-11-18", "skipped")
		seedExpense(t, ctx, pool, userID, "500", "2026-11-19", "posted")
		// Due today and not written yet: the worker will still write it.
		seedRule(t, ctx, pool, userID, "income", "daily", "2026-11-18", "100", true, nil, &salaryID, ptrDate("2026-11-17"))

		budget, err := svc.PeriodBudget(ctx, userID, "day", "2026-11-18")
		require.NoError(t, err)
		assert.Equal(t, "day", budget.Period)
		assert.Equal(t, "2026-11-18", budget.StartDate)
		assert.Equal(t, "2026-11-18", budget.EndDate)
		assert.True(t, decimal.NewFromInt(100).Equal(budget.ProjectedIncome), budget.ProjectedIncome.String())
		assert.True(t, decimal.NewFromInt(1100).Equal(budget.Funded), budget.Funded.String())
		assert.True(t, decimal.NewFromInt(50).Equal(budget.Spent), budget.Spent.String())
		assert.True(t, decimal.Zero.Equal(budget.BillsStillDue), budget.BillsStillDue.String())
		assert.True(t, decimal.NewFromInt(1050).Equal(budget.Left), budget.Left.String())
		require.NotNil(t, budget.SafeToSpendPerDay)
		assert.True(t, decimal.NewFromInt(1050).Equal(*budget.SafeToSpendPerDay), budget.SafeToSpendPerDay.String())
		assert.Equal(t, "2026-11-17", budget.PreviousPeriodStart)
		assert.Equal(t, "2026-11-17", budget.PreviousPeriodEnd)
		assert.True(t, decimal.Zero.Equal(budget.PreviousSpentToPoint), budget.PreviousSpentToPoint.String())
		assert.True(t, decimal.Zero.Equal(budget.PreviousSpentTotal), budget.PreviousSpentTotal.String())
	})

	t.Run("week bounds, bills and previous period to the same point", func(t *testing.T) {
		userID := uuid.New()
		seedIncome(t, ctx, pool, userID, "1000", "2026-11-16", "posted")
		seedExpense(t, ctx, pool, userID, "100", "2026-11-16", "posted")
		seedIncomeInCategory(t, ctx, pool, userID, "50", "2026-11-17", "posted", &reimbursementID)
		seedExpense(t, ctx, pool, userID, "999", "2026-11-17", "skipped")
		seedExpense(t, ctx, pool, userID, "200", "2026-11-18", "posted")
		seedExpense(t, ctx, pool, userID, "300", "2026-11-20", "posted")
		seedExpense(t, ctx, pool, userID, "400", "2026-11-23", "posted")

		// The previous week: 450 in total, 210 of it in its first 3 days.
		seedExpense(t, ctx, pool, userID, "50", "2026-11-09", "posted")
		seedExpense(t, ctx, pool, userID, "70", "2026-11-10", "posted")
		seedExpense(t, ctx, pool, userID, "90", "2026-11-11", "posted")
		seedExpense(t, ctx, pool, userID, "110", "2026-11-12", "posted")
		seedExpense(t, ctx, pool, userID, "130", "2026-11-13", "posted")

		budget, err := svc.PeriodBudget(ctx, userID, "week", "2026-11-18")
		require.NoError(t, err)
		assert.Equal(t, "2026-11-16", budget.StartDate)
		assert.Equal(t, "2026-11-22", budget.EndDate)
		assert.True(t, decimal.NewFromInt(1000).Equal(budget.Funded), budget.Funded.String())
		assert.True(t, decimal.NewFromInt(250).Equal(budget.Spent), budget.Spent.String())
		assert.True(t, decimal.NewFromInt(300).Equal(budget.BillsStillDue), budget.BillsStillDue.String())
		assert.True(t, decimal.NewFromInt(450).Equal(budget.Left), budget.Left.String())
		require.NotNil(t, budget.SafeToSpendPerDay)
		assert.True(t, decimal.NewFromInt(90).Equal(*budget.SafeToSpendPerDay), budget.SafeToSpendPerDay.String())
		assert.Equal(t, "2026-11-09", budget.PreviousPeriodStart)
		assert.Equal(t, "2026-11-15", budget.PreviousPeriodEnd)
		assert.True(t, decimal.NewFromInt(210).Equal(budget.PreviousSpentToPoint), budget.PreviousSpentToPoint.String())
		assert.True(t, decimal.NewFromInt(450).Equal(budget.PreviousSpentTotal), budget.PreviousSpentTotal.String())
	})

	t.Run("past period has no safe to spend", func(t *testing.T) {
		userID := uuid.New()
		seedIncome(t, ctx, pool, userID, "1000", "2026-11-17", "posted")
		seedIncome(t, ctx, pool, userID, "500", "2026-11-17", "posted")
		seedExpense(t, ctx, pool, userID, "200", "2026-11-17", "posted")
		seedExpense(t, ctx, pool, userID, "999", "2026-11-17", "skipped")
		seedExpense(t, ctx, pool, userID, "50", "2026-11-16", "posted")

		budget, err := svc.PeriodBudget(ctx, userID, "day", "2026-11-17")
		require.NoError(t, err)
		assert.True(t, decimal.NewFromInt(1500).Equal(budget.Funded), budget.Funded.String())
		assert.True(t, decimal.NewFromInt(200).Equal(budget.Spent), budget.Spent.String())
		assert.True(t, decimal.Zero.Equal(budget.BillsStillDue), budget.BillsStillDue.String())
		assert.True(t, decimal.NewFromInt(1300).Equal(budget.Left), budget.Left.String())
		assert.Nil(t, budget.SafeToSpendPerDay)
		assert.Equal(t, "2026-11-16", budget.PreviousPeriodStart)
		assert.True(t, decimal.NewFromInt(50).Equal(budget.PreviousSpentToPoint), budget.PreviousSpentToPoint.String())
		assert.True(t, decimal.NewFromInt(50).Equal(budget.PreviousSpentTotal), budget.PreviousSpentTotal.String())
	})

	t.Run("future period counts every day as remaining", func(t *testing.T) {
		userID := uuid.New()
		salary, found := category.ByName(category.Incomes, "salary")
		require.True(t, found)
		salaryID := salary.ID.String()
		seedIncome(t, ctx, pool, userID, "3100", "2026-12-01", "posted")
		seedExpense(t, ctx, pool, userID, "310", "2026-12-15", "posted")
		seedIncomeInCategory(t, ctx, pool, userID, "100", "2026-12-20", "posted", &reimbursementID)
		seedExpense(t, ctx, pool, userID, "500", "2026-11-05", "posted")
		seedExpense(t, ctx, pool, userID, "700", "2026-11-25", "posted")
		// Rules project only inside December, not the November gap between
		// today and the period: 31 days each, not 43.
		seedRule(t, ctx, pool, userID, "income", "daily", "2026-11-20", "10", true, nil, &salaryID, nil)
		seedRule(t, ctx, pool, userID, "expense", "daily", "2026-11-20", "5", true, nil, nil, nil)

		budget, err := svc.PeriodBudget(ctx, userID, "month", "2026-12-01")
		require.NoError(t, err)
		assert.Equal(t, "2026-12-01", budget.StartDate)
		assert.Equal(t, "2026-12-31", budget.EndDate)
		assert.True(t, decimal.NewFromInt(310).Equal(budget.ProjectedIncome), budget.ProjectedIncome.String())
		assert.True(t, decimal.NewFromInt(3410).Equal(budget.Funded), budget.Funded.String())
		assert.True(t, decimal.Zero.Equal(budget.Spent), budget.Spent.String())
		assert.True(t, decimal.NewFromInt(465).Equal(budget.BillsStillDue), budget.BillsStillDue.String())
		assert.True(t, decimal.NewFromInt(2945).Equal(budget.Left), budget.Left.String())
		require.NotNil(t, budget.SafeToSpendPerDay)
		assert.True(t, decimal.NewFromInt(95).Equal(*budget.SafeToSpendPerDay), budget.SafeToSpendPerDay.String())
		assert.Equal(t, "2026-11-01", budget.PreviousPeriodStart)
		assert.Equal(t, "2026-11-30", budget.PreviousPeriodEnd)
		assert.True(t, decimal.Zero.Equal(budget.PreviousSpentToPoint), budget.PreviousSpentToPoint.String())
		assert.True(t, decimal.NewFromInt(1200).Equal(budget.PreviousSpentTotal), budget.PreviousSpentTotal.String())
	})

	t.Run("projects the rules' remaining occurrences in the period", func(t *testing.T) {
		// 20:00 on 15 February in Manila; the month runs to the 28th.
		svc.now = func() time.Time { return time.Date(2026, 2, 15, 12, 0, 0, 0, time.UTC) }
		userID := uuid.New()
		salary, found := category.ByName(category.Incomes, "salary")
		require.True(t, found)
		salaryID := salary.ID.String()

		dailyIncome := seedRule(t, ctx, pool, userID, "income", "daily", "2026-02-01", "10", true, nil, &salaryID, ptrDate("2026-02-14"))
		weeklyExpense := seedRule(t, ctx, pool, userID, "expense", "weekly", "2026-01-26", "100", true, nil, nil, ptrDate("2026-02-14"))
		dailyExpense := seedRule(t, ctx, pool, userID, "expense", "daily", "2026-02-01", "7", true, nil, nil, ptrDate("2026-02-14"))
		seedRule(t, ctx, pool, userID, "income", "monthly", "2026-01-31", "700", true, nil, &salaryID, ptrDate("2026-02-14"))
		seedRule(t, ctx, pool, userID, "expense", "yearly", "2024-02-29", "1200", true, nil, nil, ptrDate("2026-02-14"))
		// Paused, ended and not-yet-started rules project nothing.
		seedRule(t, ctx, pool, userID, "income", "daily", "2026-02-01", "999", false, nil, &salaryID, ptrDate("2026-02-14"))
		ended := "2026-02-10"
		seedRule(t, ctx, pool, userID, "expense", "daily", "2026-02-01", "50", true, &ended, nil, ptrDate("2026-02-14"))
		seedRule(t, ctx, pool, userID, "expense", "daily", "2026-03-01", "999", true, nil, nil, ptrDate("2026-02-14"))
		// A projected reimbursement is not funding.
		reimbursementRuleCategory := category.Reimbursement.String()
		seedRule(t, ctx, pool, userID, "income", "daily", "2026-02-01", "500", true, nil, &reimbursementRuleCategory, ptrDate("2026-02-14"))

		// The weekly expense's 16 February entry is already written; the
		// projection must not count that date again. The daily income's
		// 20 February entry is written too, and counts as posted income.
		seedRuleExpense(t, ctx, pool, userID, "100", "2026-02-16", "posted", weeklyExpense)
		seedRuleExpense(t, ctx, pool, userID, "7", "2026-02-24", "skipped", dailyExpense)
		seedRuleIncome(t, ctx, pool, userID, "10", "2026-02-20", dailyIncome)
		seedExpense(t, ctx, pool, userID, "50", "2026-02-20", "posted")

		seedIncome(t, ctx, pool, userID, "2000", "2026-02-10", "posted")
		seedExpense(t, ctx, pool, userID, "300", "2026-02-10", "posted")
		seedIncomeInCategory(t, ctx, pool, userID, "100", "2026-02-12", "posted", &reimbursementID)
		seedExpense(t, ctx, pool, userID, "999", "2026-02-12", "skipped")

		seedExpense(t, ctx, pool, userID, "100", "2026-01-05", "posted")
		seedIncomeInCategory(t, ctx, pool, userID, "50", "2026-01-10", "posted", &reimbursementID)
		seedExpense(t, ctx, pool, userID, "200", "2026-01-15", "posted")
		seedExpense(t, ctx, pool, userID, "300", "2026-01-20", "posted")

		budget, err := svc.PeriodBudget(ctx, userID, "month", "2026-02-15")
		require.NoError(t, err)
		assert.Equal(t, "2026-02-01", budget.StartDate)
		assert.Equal(t, "2026-02-28", budget.EndDate)
		assert.True(t, decimal.NewFromInt(830).Equal(budget.ProjectedIncome), budget.ProjectedIncome.String())
		assert.True(t, decimal.NewFromInt(2840).Equal(budget.Funded), budget.Funded.String())
		assert.True(t, decimal.NewFromInt(200).Equal(budget.Spent), budget.Spent.String())
		// 150 future expenses already written, 100 + 1200 projected weekly
		// and yearly, and 91 of the daily rule's 98 (24 February is skipped).
		assert.True(t, decimal.NewFromInt(1541).Equal(budget.BillsStillDue), budget.BillsStillDue.String())
		assert.True(t, decimal.NewFromInt(1099).Equal(budget.Left), budget.Left.String())
		require.NotNil(t, budget.SafeToSpendPerDay)
		assert.True(t, decimal.RequireFromString("78.5").Equal(*budget.SafeToSpendPerDay), budget.SafeToSpendPerDay.String())
		assert.True(t, decimal.NewFromInt(250).Equal(budget.PreviousSpentToPoint), budget.PreviousSpentToPoint.String())
		assert.True(t, decimal.NewFromInt(550).Equal(budget.PreviousSpentTotal), budget.PreviousSpentTotal.String())
	})

	t.Run("bounds cross month and year edges and default to today", func(t *testing.T) {
		svc.now = func() time.Time { return time.Date(2026, 11, 18, 12, 0, 0, 0, time.UTC) }
		userID := uuid.New()

		today, err := svc.PeriodBudget(ctx, userID, "day", "")
		require.NoError(t, err)
		assert.Equal(t, "2026-11-18", today.StartDate)
		assert.Equal(t, "2026-11-18", today.EndDate)

		week, err := svc.PeriodBudget(ctx, userID, "week", "2026-12-31")
		require.NoError(t, err)
		assert.Equal(t, "2026-12-28", week.StartDate)
		assert.Equal(t, "2027-01-03", week.EndDate)
		assert.Equal(t, "2026-12-21", week.PreviousPeriodStart)
		assert.Equal(t, "2026-12-27", week.PreviousPeriodEnd)

		february, err := svc.PeriodBudget(ctx, userID, "month", "2026-02-10")
		require.NoError(t, err)
		assert.Equal(t, "2026-02-01", february.StartDate)
		assert.Equal(t, "2026-02-28", february.EndDate)
		assert.Equal(t, "2026-01-01", february.PreviousPeriodStart)
		assert.Equal(t, "2026-01-31", february.PreviousPeriodEnd)

		leap, err := svc.PeriodBudget(ctx, userID, "month", "2028-02-10")
		require.NoError(t, err)
		assert.Equal(t, "2028-02-29", leap.EndDate)
	})

	t.Run("a behind worker still owes a past period its rule entries", func(t *testing.T) {
		svc.now = func() time.Time { return time.Date(2026, 4, 15, 12, 0, 0, 0, time.UTC) }
		userID := uuid.New()
		salary, found := category.ByName(category.Incomes, "salary")
		require.True(t, found)
		salaryID := salary.ID.String()
		// Filled through 10 March, so 11-31 March (21 days) are still owed.
		seedRule(t, ctx, pool, userID, "income", "daily", "2026-03-01", "10", true, nil, &salaryID, ptrDate("2026-03-10"))
		// This one is caught up and adds nothing.
		seedRule(t, ctx, pool, userID, "income", "daily", "2026-03-01", "100", true, nil, &salaryID, ptrDate("2026-03-31"))
		// Owed past expenses are not bills still due.
		seedRule(t, ctx, pool, userID, "expense", "daily", "2026-03-01", "5", true, nil, nil, ptrDate("2026-03-10"))

		budget, err := svc.PeriodBudget(ctx, userID, "month", "2026-03-05")
		require.NoError(t, err)
		assert.True(t, decimal.NewFromInt(210).Equal(budget.ProjectedIncome), budget.ProjectedIncome.String())
		assert.True(t, decimal.NewFromInt(210).Equal(budget.Funded), budget.Funded.String())
		assert.True(t, decimal.Zero.Equal(budget.BillsStillDue), budget.BillsStillDue.String())
		assert.Nil(t, budget.SafeToSpendPerDay)
	})

	t.Run("period bounds follow the owner's zone", func(t *testing.T) {
		// 07:30 on 18 November in Manila, still 17 November in UTC.
		svc.now = func() time.Time { return time.Date(2026, 11, 17, 23, 30, 0, 0, time.UTC) }
		userID := uuid.New()
		seedIncome(t, ctx, pool, userID, "1000", "2026-11-18", "posted")
		seedExpense(t, ctx, pool, userID, "200", "2026-11-18", "posted")
		seedExpense(t, ctx, pool, userID, "300", "2026-11-19", "posted")

		day, err := svc.PeriodBudget(ctx, userID, "day", "")
		require.NoError(t, err)
		assert.Equal(t, "2026-11-18", day.StartDate)
		assert.Equal(t, "2026-11-18", day.EndDate)
		assert.True(t, decimal.NewFromInt(1000).Equal(day.Funded), day.Funded.String())
		assert.True(t, decimal.NewFromInt(200).Equal(day.Spent), day.Spent.String())
		assert.True(t, decimal.Zero.Equal(day.BillsStillDue), day.BillsStillDue.String())

		week, err := svc.PeriodBudget(ctx, userID, "week", "")
		require.NoError(t, err)
		assert.Equal(t, "2026-11-16", week.StartDate)
		assert.Equal(t, "2026-11-22", week.EndDate)
	})

	t.Run("previous to-point stays inside a shorter month", func(t *testing.T) {
		svc.now = func() time.Time { return time.Date(2026, 4, 15, 12, 0, 0, 0, time.UTC) }
		userID := uuid.New()
		seedExpense(t, ctx, pool, userID, "100", "2026-02-28", "posted")
		seedExpense(t, ctx, pool, userID, "900", "2026-03-01", "posted")

		budget, err := svc.PeriodBudget(ctx, userID, "month", "2026-03-05")
		require.NoError(t, err)
		assert.Equal(t, "2026-02-01", budget.PreviousPeriodStart)
		assert.Equal(t, "2026-02-28", budget.PreviousPeriodEnd)
		assert.True(t, decimal.NewFromInt(100).Equal(budget.PreviousSpentToPoint), budget.PreviousSpentToPoint.String())
		assert.True(t, decimal.NewFromInt(100).Equal(budget.PreviousSpentTotal), budget.PreviousSpentTotal.String())
	})
}

func ptrDate(s string) *string { return &s }

func seedRule(t *testing.T, ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID, kind, cadence, start, amount string, active bool, end, categoryID, materializedThrough *string) uuid.UUID {
	t.Helper()
	id := uuid.New()
	var endValue, categoryValue, throughValue any
	if end != nil {
		endValue = *end
	}
	if categoryID != nil {
		categoryValue = *categoryID
	}
	if materializedThrough != nil {
		throughValue = *materializedThrough
	}
	_, err := pool.Exec(ctx, `INSERT INTO rules
		(id, kind, user_id, description, amount, currency, category_id, recurring_type, start_date, end_date, priority, is_active, materialized_through)
		VALUES ($1, $2, $3, 'test rule', $4::numeric, 'PHP', $5::uuid, $6, $7::date, $8::date, 'want', $9, $10::date)`,
		id, kind, userID, amount, categoryValue, cadence, start, endValue, active, throughValue)
	require.NoError(t, err)
	return id
}

func seedRuleExpense(t *testing.T, ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID, amount, date, status string, ruleID uuid.UUID) {
	t.Helper()
	_, err := pool.Exec(ctx, `INSERT INTO expenses
		(user_id, amount, currency, exchange_rate, amount_php, rate_date, description, expense_date, priority, status, source_rule_id)
		VALUES ($1::uuid, $2::numeric, 'PHP', 1, $2::numeric, $3::date, 'rule entry', $3::date, 'want', $5, $4::uuid)`,
		userID.String(), amount, date, ruleID, status)
	require.NoError(t, err)
}

func seedRuleIncome(t *testing.T, ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID, amount, date string, ruleID uuid.UUID) {
	t.Helper()
	_, err := pool.Exec(ctx, `INSERT INTO incomes
		(user_id, amount, currency, exchange_rate, amount_php, rate_date, description, date, priority, status, source_rule_id)
		VALUES ($1::uuid, $2::numeric, 'PHP', 1, $2::numeric, $3::date, 'rule entry', $3::date, 'want', 'posted', $4::uuid)`,
		userID.String(), amount, date, ruleID)
	require.NoError(t, err)
}

// Reports total the fixed PHP amount, not the original amount (#3,
// ADR-0003).
func TestStatsService_MultiCurrencyUsesPHPAmount_Integration(t *testing.T) {
	svc, pool := newStatsIntegrationService(t)
	ctx := context.Background()
	user := uuid.New()
	rate := decimal.RequireFromString("61.573")
	php := decimal.RequireFromString("6157.30")
	rateDate := pgtype.Date{Time: time.Date(2026, 7, 10, 0, 0, 0, 0, time.UTC), Valid: true}
	day := pgtype.Date{Time: time.Date(2026, 7, 12, 0, 0, 0, 0, time.UTC), Valid: true}
	food, _ := category.ByName(category.Expenses, "food")
	salary, _ := category.ByName(category.Incomes, "salary")

	_, err := pool.Exec(ctx, `INSERT INTO expenses
		(user_id, category_id, amount, currency, exchange_rate, amount_php, rate_date, description, expense_date, priority, status)
		VALUES ($1, $2, 100, 'USD', $3, $4, $5, 'gift', $6, 'want', 'posted')`,
		user, food.ID, rate, php, rateDate, day)
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `INSERT INTO incomes
		(user_id, category_id, amount, currency, exchange_rate, amount_php, rate_date, description, date, priority, status)
		VALUES ($1, $2, 100, 'USD', $3, $4, $5, 'gift', $6, 'want', 'posted')`,
		user, salary.ID, rate, php, rateDate, day)
	require.NoError(t, err)

	summary, err := svc.Summary(ctx, user, nil, nil)
	require.NoError(t, err)
	assert.True(t, php.Equal(summary.TotalExpenses), summary.TotalExpenses.String())
	assert.True(t, php.Equal(summary.TotalIncomes), summary.TotalIncomes.String())

	trends, err := svc.Trends(ctx, user, nil, nil)
	require.NoError(t, err)
	require.Len(t, trends, 1)
	assert.True(t, php.Equal(trends[0].TotalExpenses))
	assert.True(t, php.Equal(trends[0].TotalIncomes))

	svc.now = func() time.Time { return time.Date(2026, 12, 31, 12, 0, 0, 0, time.UTC) }
	money, err := svc.CurrentTotalMoney(ctx, user)
	require.NoError(t, err)
	assert.True(t, decimal.Zero.Equal(money.TotalMoney), money.TotalMoney.String())

	savings, err := svc.SavingsRate(ctx, user, nil, nil)
	require.NoError(t, err)
	assert.True(t, php.Equal(savings.TotalIncomes))
	assert.True(t, php.Equal(savings.TotalExpenses))

	split, err := svc.FiftyThirtyTwenty(ctx, user, nil, nil)
	require.NoError(t, err)
	assert.True(t, php.Equal(split.TotalIncomes))
	assert.True(t, php.Equal(split.TotalExpenses))
	assert.True(t, php.Equal(split.Wants))

	byCategory, err := svc.CategoryBreakdown(ctx, user, nil, nil)
	require.NoError(t, err)
	var foodTotal decimal.Decimal
	for _, c := range byCategory {
		if c.CategoryName == "food" {
			foodTotal = c.TotalAmount
		}
	}
	assert.True(t, php.Equal(foodTotal), foodTotal.String())

	start, end := "2026-07-01", "2026-07-31"
	monthly, err := svc.MonthOverMonthTrends(ctx, user, &start, &end)
	require.NoError(t, err)
	require.Len(t, monthly, 1)
	assert.True(t, decimal.RequireFromString("12314.60").Equal(monthly[0].TotalAmount), monthly[0].TotalAmount.String())

	velocity, err := svc.SpendingVelocity(ctx, user, start, end)
	require.NoError(t, err)
	assert.True(t, php.Equal(velocity.AvgMonthlySpending))
}

// Synthetic trip fixture (#3): a USD 1,000 income and JPY expenses for
// 18-23 November 2026 total at their fixed PHP amounts. Never production
// data.
func TestStatsService_MultiCurrencyTripFixture_Integration(t *testing.T) {
	svc, pool := newStatsIntegrationService(t)
	ctx := context.Background()
	user := uuid.New()
	q := db.New(pool)
	incomeSvc := income.NewServiceWithRates(q, tripRates{})
	expenseSvc := expense.NewServiceWithRates(q, tripRates{})

	_, err := incomeSvc.Create(ctx, user, income.CreateRequest{
		Amount: 1000, Currency: "USD", Description: "trip fund", Date: "2026-11-16",
		Priority: "need", Status: "posted",
	})
	require.NoError(t, err)
	for day := 18; day <= 23; day++ {
		_, err := expenseSvc.Create(ctx, user, expense.CreateRequest{
			Amount: 10000, Currency: "JPY", Description: "Japan trip",
			ExpenseDate: time.Date(2026, 11, day, 0, 0, 0, 0, time.UTC).Format("2006-01-02"),
			Priority:    "want", Status: "posted",
		})
		require.NoError(t, err)
	}

	summary, err := svc.Summary(ctx, user, nil, nil)
	require.NoError(t, err)
	assert.True(t, decimal.RequireFromString("63813").Equal(summary.TotalIncomes), summary.TotalIncomes.String())
	assert.True(t, decimal.RequireFromString("23930.40").Equal(summary.TotalExpenses), summary.TotalExpenses.String())

	// All trip days are behind this clock, so every expense counts.
	svc.now = func() time.Time { return time.Date(2026, 12, 31, 12, 0, 0, 0, time.UTC) }
	money, err := svc.CurrentTotalMoney(ctx, user)
	require.NoError(t, err)
	assert.True(t, decimal.RequireFromString("39882.60").Equal(money.TotalMoney), money.TotalMoney.String())
}
