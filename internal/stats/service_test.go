package stats

import (
	"context"
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ejsadiarin/corefinance/internal/category"
	"github.com/ejsadiarin/corefinance/internal/expense"
	"github.com/ejsadiarin/corefinance/internal/income"
	"github.com/ejsadiarin/corefinance/internal/testutil"
)

func TestStatsService_Summary(t *testing.T) {
	pool := testutil.SetupTestDB(t)
	svc := NewService(pool)
	ctx := context.Background()
	userID := testutil.TestUserIDUUID()

	expenseSvc := expense.NewService(pool)
	incomeSvc := income.NewService(pool)
	catSvc := category.NewService(pool)

	// Create categories
	expCat, err := catSvc.CreateExpense(ctx, userID, category.CreateRequest{Name: "Food"})
	require.NoError(t, err)
	incCat, err := catSvc.CreateIncome(ctx, userID, category.CreateRequest{Name: "Salary"})
	require.NoError(t, err)

	// Create expenses
	for i := 0; i < 3; i++ {
		_, err := expenseSvc.Create(ctx, userID, expense.CreateRequest{
			Amount:      100,
			Currency:    "PHP",
			Description: "Expense",
			CategoryID:  ptrString(expCat.ID.String()),
			ExpenseDate: "2026-01-15",
			Priority:    "want",
			Status:      "posted",
		})
		require.NoError(t, err)
	}

	// Create incomes
	for i := 0; i < 2; i++ {
		_, err := incomeSvc.Create(ctx, userID, income.CreateRequest{
			Amount:      5000,
			Currency:    "PHP",
			Description: "Income",
			CategoryID:  ptrString(incCat.ID.String()),
			Date:        "2026-01-15",
			Priority:    "need",
			Status:      "posted",
		})
		require.NoError(t, err)
	}

	summary, err := svc.Summary(ctx, userID, nil, nil)
	require.NoError(t, err)
	assert.Equal(t, int64(3), summary.ExpenseCount)
	assert.Equal(t, int64(2), summary.IncomeCount)
}

func TestStatsService_Trends(t *testing.T) {
	pool := testutil.SetupTestDB(t)
	svc := NewService(pool)
	ctx := context.Background()
	userID := testutil.TestUserIDUUID()

	expenseSvc := expense.NewService(pool)
	catSvc := category.NewService(pool)

	expCat, err := catSvc.CreateExpense(ctx, userID, category.CreateRequest{Name: "Food"})
	require.NoError(t, err)

	// Create expenses in different months
	_, err = expenseSvc.Create(ctx, userID, expense.CreateRequest{
		Amount:      100,
		Currency:    "PHP",
		Description: "Jan Expense",
		CategoryID:  ptrString(expCat.ID.String()),
		ExpenseDate: "2026-01-15",
		Priority:    "want",
		Status:      "posted",
	})
	require.NoError(t, err)

	_, err = expenseSvc.Create(ctx, userID, expense.CreateRequest{
		Amount:      200,
		Currency:    "PHP",
		Description: "Feb Expense",
		CategoryID:  ptrString(expCat.ID.String()),
		ExpenseDate: "2026-02-15",
		Priority:    "want",
		Status:      "posted",
	})
	require.NoError(t, err)

	trends, err := svc.Trends(ctx, userID, nil, nil)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(trends), 2)
}

func TestStatsService_CategoryBreakdown(t *testing.T) {
	pool := testutil.SetupTestDB(t)
	svc := NewService(pool)
	ctx := context.Background()
	userID := testutil.TestUserIDUUID()

	expenseSvc := expense.NewService(pool)
	catSvc := category.NewService(pool)

	cat1, err := catSvc.CreateExpense(ctx, userID, category.CreateRequest{
		Name:  "Food",
		Color: testutil.StrPtr("#FF0000"),
	})
	require.NoError(t, err)

	cat2, err := catSvc.CreateExpense(ctx, userID, category.CreateRequest{
		Name:  "Transport",
		Color: testutil.StrPtr("#00FF00"),
	})
	require.NoError(t, err)

	// Create expenses
	_, err = expenseSvc.Create(ctx, userID, expense.CreateRequest{
		Amount:      100,
		Currency:    "PHP",
		Description: "Food Expense",
		CategoryID:  ptrString(cat1.ID.String()),
		ExpenseDate: "2026-01-15",
		Priority:    "want",
		Status:      "posted",
	})
	require.NoError(t, err)

	_, err = expenseSvc.Create(ctx, userID, expense.CreateRequest{
		Amount:      50,
		Currency:    "PHP",
		Description: "Transport Expense",
		CategoryID:  ptrString(cat2.ID.String()),
		ExpenseDate: "2026-01-15",
		Priority:    "want",
		Status:      "posted",
	})
	require.NoError(t, err)

	breakdown, err := svc.CategoryBreakdown(ctx, userID, nil, nil)
	require.NoError(t, err)
	assert.Len(t, breakdown, 2)
}

func TestStatsService_SavingsRate(t *testing.T) {
	pool := testutil.SetupTestDB(t)
	svc := NewService(pool)
	ctx := context.Background()
	userID := testutil.TestUserIDUUID()

	expenseSvc := expense.NewService(pool)
	incomeSvc := income.NewService(pool)
	catSvc := category.NewService(pool)

	expCat, err := catSvc.CreateExpense(ctx, userID, category.CreateRequest{Name: "Food"})
	require.NoError(t, err)
	incCat, err := catSvc.CreateIncome(ctx, userID, category.CreateRequest{Name: "Salary"})
	require.NoError(t, err)

	// Create income: 10000
	_, err = incomeSvc.Create(ctx, userID, income.CreateRequest{
		Amount:      10000,
		Currency:    "PHP",
		Description: "Salary",
		CategoryID:  ptrString(incCat.ID.String()),
		Date:        "2026-01-15",
		Priority:    "need",
		Status:      "posted",
	})
	require.NoError(t, err)

	// Create expense: 3000
	_, err = expenseSvc.Create(ctx, userID, expense.CreateRequest{
		Amount:      3000,
		Currency:    "PHP",
		Description: "Food",
		CategoryID:  ptrString(expCat.ID.String()),
		ExpenseDate: "2026-01-15",
		Priority:    "want",
		Status:      "posted",
	})
	require.NoError(t, err)

	sr, err := svc.SavingsRate(ctx, userID, nil, nil)
	require.NoError(t, err)
	// Savings rate = (10000 - 3000) / 10000 * 100 = 70%
	assert.True(t, sr.SavingsRate.GreaterThan(decimal.NewFromFloat(69)))
	assert.True(t, sr.SavingsRate.LessThan(decimal.NewFromFloat(71)))
}

func TestStatsService_FiftyThirtyTwenty(t *testing.T) {
	pool := testutil.SetupTestDB(t)
	svc := NewService(pool)
	ctx := context.Background()
	userID := testutil.TestUserIDUUID()

	expenseSvc := expense.NewService(pool)
	catSvc := category.NewService(pool)

	needCat, err := catSvc.CreateExpense(ctx, userID, category.CreateRequest{Name: "Needs"})
	require.NoError(t, err)
	wantCat, err := catSvc.CreateExpense(ctx, userID, category.CreateRequest{Name: "Wants"})
	require.NoError(t, err)
	savingsCat, err := catSvc.CreateExpense(ctx, userID, category.CreateRequest{Name: "Savings"})
	require.NoError(t, err)

	// Need expense: 500
	_, err = expenseSvc.Create(ctx, userID, expense.CreateRequest{
		Amount:      500,
		Currency:    "PHP",
		Description: "Need",
		CategoryID:  ptrString(needCat.ID.String()),
		ExpenseDate: "2026-01-15",
		Priority:    "need",
		Status:      "posted",
	})
	require.NoError(t, err)

	// Want expense: 300
	_, err = expenseSvc.Create(ctx, userID, expense.CreateRequest{
		Amount:      300,
		Currency:    "PHP",
		Description: "Want",
		CategoryID:  ptrString(wantCat.ID.String()),
		ExpenseDate: "2026-01-15",
		Priority:    "want",
		Status:      "posted",
	})
	require.NoError(t, err)

	// Savings expense: 200
	_, err = expenseSvc.Create(ctx, userID, expense.CreateRequest{
		Amount:      200,
		Currency:    "PHP",
		Description: "Savings",
		CategoryID:  ptrString(savingsCat.ID.String()),
		ExpenseDate: "2026-01-15",
		Priority:    "savings",
		Status:      "posted",
	})
	require.NoError(t, err)

	ftt, err := svc.FiftyThirtyTwenty(ctx, userID, nil, nil)
	require.NoError(t, err)
	// Total expenses = 1000
	// Needs = 500 (50%), Wants = 300 (30%), Savings = 200 (20%)
	assert.True(t, ftt.NeedsPct.GreaterThan(decimal.NewFromFloat(49)))
	assert.True(t, ftt.NeedsPct.LessThan(decimal.NewFromFloat(51)))
	assert.True(t, ftt.WantsPct.GreaterThan(decimal.NewFromFloat(29)))
	assert.True(t, ftt.WantsPct.LessThan(decimal.NewFromFloat(31)))
	assert.True(t, ftt.SavingsPct.GreaterThan(decimal.NewFromFloat(19)))
	assert.True(t, ftt.SavingsPct.LessThan(decimal.NewFromFloat(21)))
}

func TestStatsService_SpendingVelocity(t *testing.T) {
	pool := testutil.SetupTestDB(t)
	svc := NewService(pool)
	ctx := context.Background()
	userID := testutil.TestUserIDUUID()

	expenseSvc := expense.NewService(pool)
	catSvc := category.NewService(pool)

	expCat, err := catSvc.CreateExpense(ctx, userID, category.CreateRequest{Name: "Food"})
	require.NoError(t, err)

	// Create expenses across 2 months
	_, err = expenseSvc.Create(ctx, userID, expense.CreateRequest{
		Amount:      1000,
		Currency:    "PHP",
		Description: "Jan Expense",
		CategoryID:  ptrString(expCat.ID.String()),
		ExpenseDate: "2026-01-15",
		Priority:    "want",
		Status:      "posted",
	})
	require.NoError(t, err)

	_, err = expenseSvc.Create(ctx, userID, expense.CreateRequest{
		Amount:      2000,
		Currency:    "PHP",
		Description: "Feb Expense",
		CategoryID:  ptrString(expCat.ID.String()),
		ExpenseDate: "2026-02-15",
		Priority:    "want",
		Status:      "posted",
	})
	require.NoError(t, err)

	sv, err := svc.SpendingVelocity(ctx, userID, "2026-01-01", "2026-02-28")
	require.NoError(t, err)
	assert.Greater(t, sv.MonthsWithData, 0)
}

func TestStatsService_CurrentTotalMoney(t *testing.T) {
	pool := testutil.SetupTestDB(t)
	svc := NewService(pool)
	ctx := context.Background()
	userID := testutil.TestUserIDUUID()

	expenseSvc := expense.NewService(pool)
	incomeSvc := income.NewService(pool)
	catSvc := category.NewService(pool)

	expCat, err := catSvc.CreateExpense(ctx, userID, category.CreateRequest{Name: "Food"})
	require.NoError(t, err)
	incCat, err := catSvc.CreateIncome(ctx, userID, category.CreateRequest{Name: "Salary"})
	require.NoError(t, err)

	// Income: 10000
	_, err = incomeSvc.Create(ctx, userID, income.CreateRequest{
		Amount:      10000,
		Currency:    "PHP",
		Description: "Salary",
		CategoryID:  ptrString(incCat.ID.String()),
		Date:        "2026-01-15",
		Priority:    "need",
		Status:      "posted",
	})
	require.NoError(t, err)

	// Expense: 3000
	_, err = expenseSvc.Create(ctx, userID, expense.CreateRequest{
		Amount:      3000,
		Currency:    "PHP",
		Description: "Food",
		CategoryID:  ptrString(expCat.ID.String()),
		ExpenseDate: "2026-01-15",
		Priority:    "want",
		Status:      "posted",
	})
	require.NoError(t, err)

	tm, err := svc.CurrentTotalMoney(ctx, userID, nil, nil)
	require.NoError(t, err)
	// Total = 10000 - 3000 = 7000
	assert.True(t, tm.TotalMoney.GreaterThan(decimal.NewFromFloat(6999)))
	assert.True(t, tm.TotalMoney.LessThan(decimal.NewFromFloat(7001)))
}

func TestStatsService_MonthOverMonthTrends(t *testing.T) {
	pool := testutil.SetupTestDB(t)
	svc := NewService(pool)
	ctx := context.Background()
	userID := testutil.TestUserIDUUID()

	expenseSvc := expense.NewService(pool)
	catSvc := category.NewService(pool)

	expCat, err := catSvc.CreateExpense(ctx, userID, category.CreateRequest{Name: "Food"})
	require.NoError(t, err)

	// Create expenses in different months
	_, err = expenseSvc.Create(ctx, userID, expense.CreateRequest{
		Amount:      100,
		Currency:    "PHP",
		Description: "Jan Expense",
		CategoryID:  ptrString(expCat.ID.String()),
		ExpenseDate: "2026-01-15",
		Priority:    "want",
		Status:      "posted",
	})
	require.NoError(t, err)

	_, err = expenseSvc.Create(ctx, userID, expense.CreateRequest{
		Amount:      200,
		Currency:    "PHP",
		Description: "Feb Expense",
		CategoryID:  ptrString(expCat.ID.String()),
		ExpenseDate: "2026-02-15",
		Priority:    "want",
		Status:      "posted",
	})
	require.NoError(t, err)

	start := "2026-01-01"
	end := "2026-12-31"
	mom, err := svc.MonthOverMonthTrends(ctx, userID, &start, &end)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(mom), 2)
}

func ptrString(s string) *string {
	return &s
}
