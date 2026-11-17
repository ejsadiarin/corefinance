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
