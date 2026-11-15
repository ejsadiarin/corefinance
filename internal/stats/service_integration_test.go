package stats

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"

	"github.com/ejsadiarin/corefinance/internal/category"
	db "github.com/ejsadiarin/corefinance/internal/db/sqlc"
	"github.com/ejsadiarin/corefinance/internal/testutil"
)

func newStatsIntegrationService(t *testing.T) (*Service, *pgxpool.Pool) {
	t.Helper()
	pool := testutil.SetupTestDB(t)
	return NewService(db.New(pool)), pool
}

func seedExpense(t *testing.T, ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID, amount, date, status string) {
	t.Helper()
	_, err := pool.Exec(ctx, `INSERT INTO expenses
		(user_id, amount, currency, description, expense_date, priority, status)
		VALUES ($1::uuid, $2::numeric, 'PHP', 'test expense', $3::date, 'want', $4)`,
		userID.String(), amount, date, status)
	require.NoError(t, err)
}

func seedIncome(t *testing.T, ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID, amount, date, status string) {
	t.Helper()
	_, err := pool.Exec(ctx, `INSERT INTO incomes
		(user_id, amount, currency, description, date, priority, status)
		VALUES ($1::uuid, $2::numeric, 'PHP', 'test income', $3::date, 'want', $4)`,
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
		(user_id, amount, currency, description, date, priority, status, category_id)
		VALUES ($1::uuid, $2::numeric, 'PHP', 'test income', $3::date, 'want', $4, $5::uuid)`,
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

	res, err := svc.CurrentTotalMoney(ctx, userID)
	require.NoError(t, err)
	require.True(t, expected.Equal(res.TotalMoney), "expected lifetime total_money=800, got %s", res.TotalMoney.String())
}
