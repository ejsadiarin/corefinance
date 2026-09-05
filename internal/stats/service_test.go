package stats

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	db "github.com/ejsadiarin/corefinance/internal/db/sqlc"
	"github.com/ejsadiarin/corefinance/internal/mock"
)

func TestStatsService_Summary(t *testing.T) {
	userID := uuid.New()

	t.Run("success with date range", func(t *testing.T) {
		start := "2026-01-01"
		end := "2026-01-31"
		mock := &mock.MockQuerier{
			GetSummaryFn: func(ctx context.Context, arg db.GetSummaryParams) (db.GetSummaryRow, error) {
				assert.Equal(t, userID, arg.UserID)
				assert.True(t, arg.StartDate.Valid)
				assert.True(t, arg.EndDate.Valid)
				return db.GetSummaryRow{
					TotalExpenses: decimal.NewFromFloat(5000),
					TotalIncomes:  decimal.NewFromFloat(8000),
					ExpenseCount:  15,
					IncomeCount:   3,
				}, nil
			},
		}
		svc := NewService(mock)
		result, err := svc.Summary(context.Background(), userID, &start, &end)
		require.NoError(t, err)
		assert.Equal(t, "5000", result.TotalExpenses.String())
		assert.Equal(t, "8000", result.TotalIncomes.String())
		assert.Equal(t, int64(15), result.ExpenseCount)
		assert.Equal(t, int64(3), result.IncomeCount)
	})

	t.Run("nil dates", func(t *testing.T) {
		mock := &mock.MockQuerier{
			GetSummaryFn: func(ctx context.Context, arg db.GetSummaryParams) (db.GetSummaryRow, error) {
				assert.False(t, arg.StartDate.Valid)
				assert.False(t, arg.EndDate.Valid)
				return db.GetSummaryRow{
					TotalExpenses: decimal.Zero,
					TotalIncomes:  decimal.NewFromFloat(1000),
				}, nil
			},
		}
		svc := NewService(mock)
		result, err := svc.Summary(context.Background(), userID, nil, nil)
		require.NoError(t, err)
		assert.Equal(t, "0", result.TotalExpenses.String())
	})

	t.Run("error", func(t *testing.T) {
		mock := &mock.MockQuerier{
			GetSummaryFn: func(ctx context.Context, arg db.GetSummaryParams) (db.GetSummaryRow, error) {
				return db.GetSummaryRow{}, errors.New("db error")
			},
		}
		svc := NewService(mock)
		_, err := svc.Summary(context.Background(), userID, nil, nil)
		require.Error(t, err)
	})
}

func TestStatsService_Trends(t *testing.T) {
	userID := uuid.New()

	t.Run("success", func(t *testing.T) {
		mock := &mock.MockQuerier{
			GetTrendsFn: func(ctx context.Context, arg db.GetTrendsParams) ([]db.GetTrendsRow, error) {
				assert.Equal(t, userID, arg.UserID)
				return []db.GetTrendsRow{
					{Month: "2026-01", TotalExpenses: decimal.NewFromFloat(3000), TotalIncomes: decimal.NewFromFloat(5000)},
					{Month: "2025-12", TotalExpenses: decimal.NewFromFloat(2500), TotalIncomes: decimal.NewFromFloat(4800)},
				}, nil
			},
		}
		svc := NewService(mock)
		result, err := svc.Trends(context.Background(), userID, nil, nil)
		require.NoError(t, err)
		assert.Len(t, result, 2)
		assert.Equal(t, "2026-01", result[0].Month)
		assert.Equal(t, "3000", result[0].TotalExpenses.String())
	})

	t.Run("empty", func(t *testing.T) {
		mock := &mock.MockQuerier{
			GetTrendsFn: func(ctx context.Context, arg db.GetTrendsParams) ([]db.GetTrendsRow, error) {
				return []db.GetTrendsRow{}, nil
			},
		}
		svc := NewService(mock)
		result, err := svc.Trends(context.Background(), userID, nil, nil)
		require.NoError(t, err)
		assert.Empty(t, result)
	})

	t.Run("error", func(t *testing.T) {
		mock := &mock.MockQuerier{
			GetTrendsFn: func(ctx context.Context, arg db.GetTrendsParams) ([]db.GetTrendsRow, error) {
				return nil, errors.New("db error")
			},
		}
		svc := NewService(mock)
		_, err := svc.Trends(context.Background(), userID, nil, nil)
		require.Error(t, err)
	})
}

func TestStatsService_CategoryBreakdown(t *testing.T) {
	userID := uuid.New()
	catID := uuid.New()

	t.Run("success with color", func(t *testing.T) {
		mock := &mock.MockQuerier{
			GetCategoryBreakdownFn: func(ctx context.Context, arg db.GetCategoryBreakdownParams) ([]db.GetCategoryBreakdownRow, error) {
				assert.Equal(t, userID, arg.UserID)
				return []db.GetCategoryBreakdownRow{
					{
						CategoryID:    catID,
						CategoryName:  "Food",
						CategoryColor: pgtype.Text{String: "#FF0000", Valid: true},
						ExpenseCount:  10,
						TotalAmount:   decimal.NewFromFloat(2500),
					},
				}, nil
			},
		}
		svc := NewService(mock)
		result, err := svc.CategoryBreakdown(context.Background(), userID, nil, nil)
		require.NoError(t, err)
		assert.Len(t, result, 1)
		assert.Equal(t, catID.String(), result[0].CategoryID)
		assert.Equal(t, "Food", result[0].CategoryName)
		require.NotNil(t, result[0].CategoryColor)
		assert.Equal(t, "#FF0000", *result[0].CategoryColor)
		assert.Equal(t, int64(10), result[0].ExpenseCount)
	})

	t.Run("nil color", func(t *testing.T) {
		mock := &mock.MockQuerier{
			GetCategoryBreakdownFn: func(ctx context.Context, arg db.GetCategoryBreakdownParams) ([]db.GetCategoryBreakdownRow, error) {
				return []db.GetCategoryBreakdownRow{
					{
						CategoryID:    catID,
						CategoryName:  "Uncategorized",
						CategoryColor: pgtype.Text{Valid: false},
						ExpenseCount:  0,
						TotalAmount:   decimal.Zero,
					},
				}, nil
			},
		}
		svc := NewService(mock)
		result, err := svc.CategoryBreakdown(context.Background(), userID, nil, nil)
		require.NoError(t, err)
		assert.Len(t, result, 1)
		require.NotNil(t, result[0].CategoryColor)
		assert.Equal(t, "", *result[0].CategoryColor)
	})

	t.Run("error", func(t *testing.T) {
		mock := &mock.MockQuerier{
			GetCategoryBreakdownFn: func(ctx context.Context, arg db.GetCategoryBreakdownParams) ([]db.GetCategoryBreakdownRow, error) {
				return nil, errors.New("db error")
			},
		}
		svc := NewService(mock)
		_, err := svc.CategoryBreakdown(context.Background(), userID, nil, nil)
		require.Error(t, err)
	})
}

func TestStatsService_SavingsRate(t *testing.T) {
	userID := uuid.New()

	t.Run("success", func(t *testing.T) {
		mock := &mock.MockQuerier{
			GetSavingsRateFn: func(ctx context.Context, arg db.GetSavingsRateParams) (db.GetSavingsRateRow, error) {
				assert.Equal(t, userID, arg.UserID)
				return db.GetSavingsRateRow{
					TotalIncomes:  decimal.NewFromFloat(10000),
					TotalExpenses: decimal.NewFromFloat(7000),
					SavingsRate:   decimal.NewFromFloat(30),
				}, nil
			},
		}
		svc := NewService(mock)
		result, err := svc.SavingsRate(context.Background(), userID, nil, nil)
		require.NoError(t, err)
		assert.Equal(t, "10000", result.TotalIncomes.String())
		assert.Equal(t, "7000", result.TotalExpenses.String())
		assert.Equal(t, "30", result.SavingsRate.String())
	})

	t.Run("error", func(t *testing.T) {
		mock := &mock.MockQuerier{
			GetSavingsRateFn: func(ctx context.Context, arg db.GetSavingsRateParams) (db.GetSavingsRateRow, error) {
				return db.GetSavingsRateRow{}, errors.New("db error")
			},
		}
		svc := NewService(mock)
		_, err := svc.SavingsRate(context.Background(), userID, nil, nil)
		require.Error(t, err)
	})
}

func TestStatsService_FiftyThirtyTwenty(t *testing.T) {
	userID := uuid.New()

	t.Run("success", func(t *testing.T) {
		mock := &mock.MockQuerier{
			GetFiftyThirtyTwentyFn: func(ctx context.Context, arg db.GetFiftyThirtyTwentyParams) (db.GetFiftyThirtyTwentyRow, error) {
				assert.Equal(t, userID, arg.UserID)
				return db.GetFiftyThirtyTwentyRow{
					TotalIncomes:  decimal.NewFromFloat(10000),
					TotalExpenses: decimal.NewFromFloat(8000),
					Needs:         decimal.NewFromFloat(4000),
					Wants:         decimal.NewFromFloat(2400),
					Savings:       decimal.NewFromFloat(1600),
					NeedsPct:      decimal.NewFromFloat(50),
					WantsPct:      decimal.NewFromFloat(30),
					SavingsPct:    decimal.NewFromFloat(20),
				}, nil
			},
		}
		svc := NewService(mock)
		result, err := svc.FiftyThirtyTwenty(context.Background(), userID, nil, nil)
		require.NoError(t, err)
		assert.Equal(t, "10000", result.TotalIncomes.String())
		assert.Equal(t, "8000", result.TotalExpenses.String())
		assert.Equal(t, "4000", result.Needs.String())
		assert.Equal(t, "2400", result.Wants.String())
		assert.Equal(t, "1600", result.Savings.String())
		assert.Equal(t, "50", result.NeedsPct.String())
		assert.Equal(t, "30", result.WantsPct.String())
		assert.Equal(t, "20", result.SavingsPct.String())
	})

	t.Run("zero expenses", func(t *testing.T) {
		mock := &mock.MockQuerier{
			GetFiftyThirtyTwentyFn: func(ctx context.Context, arg db.GetFiftyThirtyTwentyParams) (db.GetFiftyThirtyTwentyRow, error) {
				return db.GetFiftyThirtyTwentyRow{
					TotalIncomes:  decimal.NewFromFloat(10000),
					TotalExpenses: decimal.Zero,
					Needs:         decimal.Zero,
					Wants:         decimal.Zero,
					Savings:       decimal.Zero,
					NeedsPct:      decimal.Zero,
					WantsPct:      decimal.Zero,
					SavingsPct:    decimal.Zero,
				}, nil
			},
		}
		svc := NewService(mock)
		result, err := svc.FiftyThirtyTwenty(context.Background(), userID, nil, nil)
		require.NoError(t, err)
		assert.Equal(t, "0", result.TotalExpenses.String())
		assert.Equal(t, "0", result.NeedsPct.String())
	})

	t.Run("error", func(t *testing.T) {
		mock := &mock.MockQuerier{
			GetFiftyThirtyTwentyFn: func(ctx context.Context, arg db.GetFiftyThirtyTwentyParams) (db.GetFiftyThirtyTwentyRow, error) {
				return db.GetFiftyThirtyTwentyRow{}, errors.New("db error")
			},
		}
		svc := NewService(mock)
		_, err := svc.FiftyThirtyTwenty(context.Background(), userID, nil, nil)
		require.Error(t, err)
	})
}

func TestStatsService_UpcomingBills(t *testing.T) {
	userID := uuid.New()

	t.Run("success", func(t *testing.T) {
		mock := &mock.MockQuerier{
			GetUpcomingRecurringExpensesFn: func(ctx context.Context, uid uuid.UUID) ([]db.GetUpcomingRecurringExpensesRow, error) {
				assert.Equal(t, userID, uid)
				return []db.GetUpcomingRecurringExpensesRow{
					{
						ID:          uuid.New(),
						Description: "Rent",
						Amount:      decimal.NewFromFloat(1500),
						CategoryName: pgtype.Text{String: "Housing", Valid: true},
					},
					{
						ID:          uuid.New(),
						Description: "Internet",
						Amount:      decimal.NewFromFloat(100),
					},
				}, nil
			},
		}
		svc := NewService(mock)
		result, err := svc.UpcomingBills(context.Background(), userID)
		require.NoError(t, err)
		assert.Len(t, result, 2)
		assert.Equal(t, "Rent", result[0].Description)
		assert.Equal(t, "Internet", result[1].Description)
	})

	t.Run("empty", func(t *testing.T) {
		mock := &mock.MockQuerier{
			GetUpcomingRecurringExpensesFn: func(ctx context.Context, uid uuid.UUID) ([]db.GetUpcomingRecurringExpensesRow, error) {
				return []db.GetUpcomingRecurringExpensesRow{}, nil
			},
		}
		svc := NewService(mock)
		result, err := svc.UpcomingBills(context.Background(), userID)
		require.NoError(t, err)
		assert.Empty(t, result)
	})

	t.Run("error", func(t *testing.T) {
		mock := &mock.MockQuerier{
			GetUpcomingRecurringExpensesFn: func(ctx context.Context, uid uuid.UUID) ([]db.GetUpcomingRecurringExpensesRow, error) {
				return nil, errors.New("db error")
			},
		}
		svc := NewService(mock)
		_, err := svc.UpcomingBills(context.Background(), userID)
		require.Error(t, err)
	})
}

func TestStatsService_SpendingVelocity(t *testing.T) {
	userID := uuid.New()

	t.Run("success", func(t *testing.T) {
		mock := &mock.MockQuerier{
			GetSpendingVelocityFn: func(ctx context.Context, arg db.GetSpendingVelocityParams) (db.GetSpendingVelocityRow, error) {
				assert.Equal(t, userID, arg.UserID)
				assert.True(t, arg.StartDate.Valid)
				assert.True(t, arg.EndDate.Valid)
				return db.GetSpendingVelocityRow{
					AvgMonthlySpending: decimal.NewFromFloat(3500.75),
					MonthsWithData:     6,
				}, nil
			},
		}
		svc := NewService(mock)
		result, err := svc.SpendingVelocity(context.Background(), userID, "2026-01-01", "2026-06-30")
		require.NoError(t, err)
		assert.Equal(t, "3500.75", result.AvgMonthlySpending.String())
		assert.Equal(t, 6, result.MonthsWithData)
	})

	t.Run("error", func(t *testing.T) {
		mock := &mock.MockQuerier{
			GetSpendingVelocityFn: func(ctx context.Context, arg db.GetSpendingVelocityParams) (db.GetSpendingVelocityRow, error) {
				return db.GetSpendingVelocityRow{}, errors.New("db error")
			},
		}
		svc := NewService(mock)
		_, err := svc.SpendingVelocity(context.Background(), userID, "2026-01-01", "2026-06-30")
		require.Error(t, err)
	})
}

func TestStatsService_CurrentTotalMoney(t *testing.T) {
	userID := uuid.New()

	t.Run("success", func(t *testing.T) {
		mock := &mock.MockQuerier{
			GetCurrentTotalMoneyFn: func(ctx context.Context, arg db.GetCurrentTotalMoneyParams) (decimal.Decimal, error) {
				assert.Equal(t, userID, arg.UserID)
				return decimal.NewFromFloat(15000), nil
			},
		}
		svc := NewService(mock)
		result, err := svc.CurrentTotalMoney(context.Background(), userID, nil, nil)
		require.NoError(t, err)
		assert.Equal(t, "15000", result.TotalMoney.String())
	})

	t.Run("with dates", func(t *testing.T) {
		start := "2026-01-01"
		end := "2026-06-30"
		mock := &mock.MockQuerier{
			GetCurrentTotalMoneyFn: func(ctx context.Context, arg db.GetCurrentTotalMoneyParams) (decimal.Decimal, error) {
				assert.True(t, arg.StartDate.Valid)
				assert.True(t, arg.EndDate.Valid)
				return decimal.NewFromFloat(5000), nil
			},
		}
		svc := NewService(mock)
		result, err := svc.CurrentTotalMoney(context.Background(), userID, &start, &end)
		require.NoError(t, err)
		assert.Equal(t, "5000", result.TotalMoney.String())
	})

	t.Run("error", func(t *testing.T) {
		mock := &mock.MockQuerier{
			GetCurrentTotalMoneyFn: func(ctx context.Context, arg db.GetCurrentTotalMoneyParams) (decimal.Decimal, error) {
				return decimal.Zero, errors.New("db error")
			},
		}
		svc := NewService(mock)
		_, err := svc.CurrentTotalMoney(context.Background(), userID, nil, nil)
		require.Error(t, err)
	})
}

func TestStatsService_MonthOverMonthTrends(t *testing.T) {
	userID := uuid.New()

	t.Run("success", func(t *testing.T) {
		mock := &mock.MockQuerier{
			GetMonthOverMonthTrendsFn: func(ctx context.Context, arg db.GetMonthOverMonthTrendsParams) ([]db.GetMonthOverMonthTrendsRow, error) {
				assert.Equal(t, userID, arg.UserID)
				assert.False(t, arg.StartDate.Valid)
				assert.False(t, arg.EndDate.Valid)
				return []db.GetMonthOverMonthTrendsRow{
					{Month: "2026-01", TotalAmount: decimal.NewFromFloat(5000)},
					{Month: "2025-12", TotalAmount: decimal.NewFromFloat(4200)},
				}, nil
			},
		}
		svc := NewService(mock)
		result, err := svc.MonthOverMonthTrends(context.Background(), userID, nil, nil)
		require.NoError(t, err)
		assert.Len(t, result, 2)
		assert.Equal(t, "2026-01", result[0].Month)
		assert.Equal(t, "5000", result[0].TotalAmount.String())
	})

	t.Run("with dates", func(t *testing.T) {
		start := "2026-01-01"
		end := "2026-06-30"
		mock := &mock.MockQuerier{
			GetMonthOverMonthTrendsFn: func(ctx context.Context, arg db.GetMonthOverMonthTrendsParams) ([]db.GetMonthOverMonthTrendsRow, error) {
				assert.True(t, arg.StartDate.Valid)
				assert.True(t, arg.EndDate.Valid)
				return []db.GetMonthOverMonthTrendsRow{}, nil
			},
		}
		svc := NewService(mock)
		result, err := svc.MonthOverMonthTrends(context.Background(), userID, &start, &end)
		require.NoError(t, err)
		assert.Empty(t, result)
	})

	t.Run("error", func(t *testing.T) {
		mock := &mock.MockQuerier{
			GetMonthOverMonthTrendsFn: func(ctx context.Context, arg db.GetMonthOverMonthTrendsParams) ([]db.GetMonthOverMonthTrendsRow, error) {
				return nil, errors.New("db error")
			},
		}
		svc := NewService(mock)
		_, err := svc.MonthOverMonthTrends(context.Background(), userID, nil, nil)
		require.Error(t, err)
	})
}
