package budget

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ejsadiarin/corefinance/internal/category"
	db "github.com/ejsadiarin/corefinance/internal/db/sqlc"
	"github.com/ejsadiarin/corefinance/internal/mock"
	"github.com/ejsadiarin/corefinance/internal/rates"
)

type importRates struct {
	err error
}

func (importRates) Rate(context.Context, string, time.Time) (rates.Rate, error) {
	return rates.Rate{Value: decimal.NewFromInt(1)}, nil
}

func (r importRates) Supports(context.Context, string) error { return r.err }

func TestService_PriorityGroups(t *testing.T) {
	svc := &Service{}
	groups := svc.PriorityGroups()

	require.Len(t, groups, 3)
	assert.Equal(t, "need", groups[0].ID)
	assert.Equal(t, "want", groups[1].ID)
	assert.Equal(t, "savings", groups[2].ID)
}

func TestService_Remaining(t *testing.T) {
	userID := uuid.New()

	t.Run("success", func(t *testing.T) {
		m := &mock.MockQuerier{
			GetTotalIncomesByDateRangeFn: func(ctx context.Context, arg db.GetTotalIncomesByDateRangeParams) (decimal.Decimal, error) {
				assert.Equal(t, userID, arg.UserID)
				assert.True(t, arg.Date.Valid)
				assert.True(t, arg.Date_2.Valid)
				return decimal.NewFromFloat(50000), nil
			},
			GetTotalExpensesByDateRangeFn: func(ctx context.Context, arg db.GetTotalExpensesByDateRangeParams) (decimal.Decimal, error) {
				assert.Equal(t, userID, arg.UserID)
				assert.True(t, arg.ExpenseDate.Valid)
				assert.True(t, arg.ExpenseDate_2.Valid)
				return decimal.NewFromFloat(30000), nil
			},
		}
		svc := NewService(m, nil)

		result, err := svc.Remaining(context.Background(), userID)
		require.NoError(t, err)
		assert.Equal(t, "20000", result.Remaining.(decimal.Decimal).String())
		assert.Equal(t, "50000", result.TotalIncome.(decimal.Decimal).String())
		assert.Equal(t, "30000", result.TotalExpense.(decimal.Decimal).String())
		assert.NotEmpty(t, result.PeriodStart)
		assert.NotEmpty(t, result.PeriodEnd)
	})

	t.Run("income query error", func(t *testing.T) {
		m := &mock.MockQuerier{
			GetTotalIncomesByDateRangeFn: func(ctx context.Context, arg db.GetTotalIncomesByDateRangeParams) (decimal.Decimal, error) {
				return decimal.Decimal{}, errors.New("db error")
			},
		}
		svc := NewService(m, nil)

		_, err := svc.Remaining(context.Background(), userID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "db error")
	})

	t.Run("expense query error", func(t *testing.T) {
		m := &mock.MockQuerier{
			GetTotalIncomesByDateRangeFn: func(ctx context.Context, arg db.GetTotalIncomesByDateRangeParams) (decimal.Decimal, error) {
				return decimal.NewFromFloat(50000), nil
			},
			GetTotalExpensesByDateRangeFn: func(ctx context.Context, arg db.GetTotalExpensesByDateRangeParams) (decimal.Decimal, error) {
				return decimal.Decimal{}, errors.New("expense query failed")
			},
		}
		svc := NewService(m, nil)

		_, err := svc.Remaining(context.Background(), userID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "expense query failed")
	})

	t.Run("zero income and expenses", func(t *testing.T) {
		m := &mock.MockQuerier{
			GetTotalIncomesByDateRangeFn: func(ctx context.Context, arg db.GetTotalIncomesByDateRangeParams) (decimal.Decimal, error) {
				return decimal.NewFromFloat(0), nil
			},
			GetTotalExpensesByDateRangeFn: func(ctx context.Context, arg db.GetTotalExpensesByDateRangeParams) (decimal.Decimal, error) {
				return decimal.NewFromFloat(0), nil
			},
		}
		svc := NewService(m, nil)

		result, err := svc.Remaining(context.Background(), userID)
		require.NoError(t, err)
		assert.Equal(t, "0", result.Remaining.(decimal.Decimal).String())
	})
}

func TestService_Export(t *testing.T) {
	userID, tagID, expID, ruleID := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	food, _ := category.ByName(category.Expenses, "food")
	day := pgtype.Date{Time: time.Date(2026, 11, 24, 0, 0, 0, 0, time.UTC), Valid: true}
	m := &mock.MockQuerier{
		ListTagsFn: func(ctx context.Context, uid uuid.UUID) ([]db.Tag, error) {
			return []db.Tag{{ID: tagID, UserID: userID, Name: "essential"}}, nil
		},
		ListAllRulesByUserFn: func(ctx context.Context, uid uuid.UUID) ([]db.Rule, error) {
			return []db.Rule{{ID: ruleID, Kind: "expense", UserID: userID, Amount: decimal.NewFromInt(500),
				Currency: "PHP", CategoryID: pgtype.UUID{Bytes: food.ID, Valid: true}, RecurringType: "monthly",
				StartDate: day, Priority: "need", IsActive: false, MaterializedThrough: day,
				NextDueAt: pgtype.Timestamptz{Time: time.Now(), Valid: true}}}, nil
		},
		ListAllExpensesByUserFn: func(ctx context.Context, uid uuid.UUID) ([]db.Expense, error) {
			return []db.Expense{{ID: expID, UserID: userID, CategoryID: pgtype.UUID{Bytes: food.ID, Valid: true},
				Amount: decimal.NewFromInt(500), Currency: "PHP", Description: "Groceries", ExpenseDate: day,
				SourceRuleID: pgtype.UUID{Bytes: ruleID, Valid: true}}}, nil
		},
		ListAllIncomesByUserFn: func(ctx context.Context, uid uuid.UUID) ([]db.Income, error) {
			return []db.Income{}, nil
		},
		ListAllExpenseTagsByUserFn: func(ctx context.Context, uid uuid.UUID) ([]db.ExpenseTag, error) {
			return []db.ExpenseTag{{ExpenseID: expID, TagID: tagID}}, nil
		},
		ListAllIncomeTagsByUserFn: func(ctx context.Context, uid uuid.UUID) ([]db.IncomeTag, error) {
			return []db.IncomeTag{}, nil
		},
	}

	part, err := NewService(m, nil).Export(context.Background(), userID)
	require.NoError(t, err)
	assert.Equal(t, PartFormat, part.Format)
	assert.Equal(t, PartVersion, part.Version)
	assert.Equal(t, PartSchema, part.Schema)
	require.Len(t, part.Data.Expenses, 1)
	e := part.Data.Expenses[0]
	assert.Equal(t, "2026-11-24", e.Date)
	assert.Equal(t, "food", *e.Category)
	assert.Equal(t, ruleID, *e.RuleID)
	assert.Equal(t, []uuid.UUID{tagID}, e.Tags)
	require.Len(t, part.Data.Rules, 1)
	assert.True(t, part.Data.Rules[0].Paused)
	assert.Equal(t, "2026-11-24", *part.Data.Rules[0].MaterializedThrough)
	assert.Empty(t, part.Data.Incomes)

	file, err := json.Marshal(part)
	require.NoError(t, err)
	assert.NotContains(t, string(file), "user_id")
	assert.NotContains(t, string(file), "next_due_at")

	t.Run("a query error is returned", func(t *testing.T) {
		m := &mock.MockQuerier{ListTagsFn: func(ctx context.Context, uid uuid.UUID) ([]db.Tag, error) {
			return nil, errors.New("tags query failed")
		}}
		_, err := NewService(m, nil).Export(context.Background(), userID)
		require.ErrorContains(t, err, "tags query failed")
	})
}

func TestImportCurrencyValidation(t *testing.T) {
	allow := NewServiceWithRates(nil, nil, importRates{})
	require.NoError(t, allow.checkImportCurrency(context.Background(), "PHP"))
	require.NoError(t, allow.checkImportCurrency(context.Background(), "usd"))

	reject := NewServiceWithRates(nil, nil, importRates{err: rates.ErrUnsupportedCurrency})
	err := reject.checkImportCurrency(context.Background(), "XYZ")
	require.ErrorIs(t, err, rates.ErrUnsupportedCurrency)
}

func TestImportedConversion(t *testing.T) {
	recordDate := pgtype.Date{Time: time.Date(2026, 7, 12, 0, 0, 0, 0, time.UTC), Valid: true}

	t.Run("PHP falls back to rate 1", func(t *testing.T) {
		conv, err := importedConversion(
			decimal.NewFromInt(100), "php", decimal.Zero, decimal.Zero, pgtype.Date{}, recordDate,
		)
		require.NoError(t, err)
		assert.Equal(t, "PHP", conv.Currency)
		assert.True(t, decimal.NewFromInt(1).Equal(conv.Rate))
		assert.True(t, decimal.NewFromInt(100).Equal(conv.AmountPHP))
		assert.Equal(t, "2026-07-12", conv.RateDate.Format("2006-01-02"))
	})

	t.Run("non-PHP trusts the file's conversion", func(t *testing.T) {
		conv, err := importedConversion(
			decimal.NewFromInt(100), "USD", decimal.RequireFromString("61.573"),
			decimal.RequireFromString("6157.30"),
			pgtype.Date{Time: time.Date(2026, 7, 10, 0, 0, 0, 0, time.UTC), Valid: true}, recordDate,
		)
		require.NoError(t, err)
		assert.Equal(t, "USD", conv.Currency)
		assert.True(t, decimal.RequireFromString("6157.30").Equal(conv.AmountPHP))
		assert.Equal(t, "2026-07-10", conv.RateDate.Format("2006-01-02"))
	})

	t.Run("legacy non-PHP is refused", func(t *testing.T) {
		_, err := importedConversion(
			decimal.NewFromInt(100), "USD", decimal.Zero, decimal.Zero, pgtype.Date{}, recordDate,
		)
		require.ErrorIs(t, err, ErrLegacyCurrency)
	})
}
