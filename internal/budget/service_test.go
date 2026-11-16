package budget

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

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
	userID := uuid.New()

	t.Run("success", func(t *testing.T) {
		m := &mock.MockQuerier{
			ListExpenseCategoriesFn: func(ctx context.Context) ([]db.ExpenseCategory, error) {
				return []db.ExpenseCategory{
					{ID: uuid.New(), Name: "Food", IsActive: true},
				}, nil
			},
			ListIncomeCategoriesFn: func(ctx context.Context) ([]db.IncomeCategory, error) {
				return []db.IncomeCategory{
					{ID: uuid.New(), Name: "Salary", IsActive: true},
				}, nil
			},
			ListTagsFn: func(ctx context.Context, uid uuid.UUID) ([]db.Tag, error) {
				return []db.Tag{
					{ID: uuid.New(), UserID: userID, Name: "essential"},
				}, nil
			},
			ListAllExpensesByUserFn: func(ctx context.Context, uid uuid.UUID) ([]db.Expense, error) {
				return []db.Expense{}, nil
			},
			ListAllIncomesByUserFn: func(ctx context.Context, uid uuid.UUID) ([]db.Income, error) {
				return []db.Income{}, nil
			},
			ListRecurringExpenseRulesFn: func(ctx context.Context, uid uuid.UUID) ([]db.ListRecurringExpenseRulesRow, error) {
				return []db.ListRecurringExpenseRulesRow{}, nil
			},
			ListRecurringIncomeRulesFn: func(ctx context.Context, uid uuid.UUID) ([]db.Rule, error) {
				return []db.Rule{}, nil
			},
			ListAllExpenseTagsByUserFn: func(ctx context.Context, uid uuid.UUID) ([]db.ExpenseTag, error) {
				return []db.ExpenseTag{}, nil
			},
			ListAllIncomeTagsByUserFn: func(ctx context.Context, uid uuid.UUID) ([]db.IncomeTag, error) {
				return []db.IncomeTag{}, nil
			},
		}
		svc := NewService(m, nil)

		result, err := svc.Export(context.Background(), userID)
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Len(t, result.ExpenseCategories, 1)
		assert.Len(t, result.IncomeCategories, 1)
		assert.Len(t, result.Tags, 1)
		assert.Empty(t, result.Expenses)
		assert.Empty(t, result.Incomes)
	})

	t.Run("expense categories query error", func(t *testing.T) {
		m := &mock.MockQuerier{
			ListExpenseCategoriesFn: func(ctx context.Context) ([]db.ExpenseCategory, error) {
				return nil, errors.New("db error")
			},
		}
		svc := NewService(m, nil)

		_, err := svc.Export(context.Background(), userID)
		require.Error(t, err)
	})

	t.Run("tags query error", func(t *testing.T) {
		m := &mock.MockQuerier{
			ListExpenseCategoriesFn: func(ctx context.Context) ([]db.ExpenseCategory, error) {
				return []db.ExpenseCategory{}, nil
			},
			ListIncomeCategoriesFn: func(ctx context.Context) ([]db.IncomeCategory, error) {
				return []db.IncomeCategory{}, nil
			},
			ListTagsFn: func(ctx context.Context, uid uuid.UUID) ([]db.Tag, error) {
				return nil, errors.New("tags query failed")
			},
		}
		svc := NewService(m, nil)

		_, err := svc.Export(context.Background(), userID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "tags query failed")
	})

	t.Run("expenses query error", func(t *testing.T) {
		m := &mock.MockQuerier{
			ListExpenseCategoriesFn: func(ctx context.Context) ([]db.ExpenseCategory, error) {
				return []db.ExpenseCategory{}, nil
			},
			ListIncomeCategoriesFn: func(ctx context.Context) ([]db.IncomeCategory, error) {
				return []db.IncomeCategory{}, nil
			},
			ListTagsFn: func(ctx context.Context, uid uuid.UUID) ([]db.Tag, error) {
				return []db.Tag{}, nil
			},
			ListAllExpensesByUserFn: func(ctx context.Context, uid uuid.UUID) ([]db.Expense, error) {
				return nil, errors.New("expenses query failed")
			},
		}
		svc := NewService(m, nil)

		_, err := svc.Export(context.Background(), userID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "expenses query failed")
	})

	t.Run("incomes query error", func(t *testing.T) {
		m := &mock.MockQuerier{
			ListExpenseCategoriesFn: func(ctx context.Context) ([]db.ExpenseCategory, error) {
				return []db.ExpenseCategory{}, nil
			},
			ListIncomeCategoriesFn: func(ctx context.Context) ([]db.IncomeCategory, error) {
				return []db.IncomeCategory{}, nil
			},
			ListTagsFn: func(ctx context.Context, uid uuid.UUID) ([]db.Tag, error) {
				return []db.Tag{}, nil
			},
			ListAllExpensesByUserFn: func(ctx context.Context, uid uuid.UUID) ([]db.Expense, error) {
				return []db.Expense{}, nil
			},
			ListAllIncomesByUserFn: func(ctx context.Context, uid uuid.UUID) ([]db.Income, error) {
				return nil, errors.New("incomes query failed")
			},
		}
		svc := NewService(m, nil)

		_, err := svc.Export(context.Background(), userID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "incomes query failed")
	})

	t.Run("recurring expenses query error", func(t *testing.T) {
		m := &mock.MockQuerier{
			ListExpenseCategoriesFn: func(ctx context.Context) ([]db.ExpenseCategory, error) {
				return []db.ExpenseCategory{}, nil
			},
			ListIncomeCategoriesFn: func(ctx context.Context) ([]db.IncomeCategory, error) {
				return []db.IncomeCategory{}, nil
			},
			ListTagsFn: func(ctx context.Context, uid uuid.UUID) ([]db.Tag, error) {
				return []db.Tag{}, nil
			},
			ListAllExpensesByUserFn: func(ctx context.Context, uid uuid.UUID) ([]db.Expense, error) {
				return []db.Expense{}, nil
			},
			ListAllIncomesByUserFn: func(ctx context.Context, uid uuid.UUID) ([]db.Income, error) {
				return []db.Income{}, nil
			},
			ListRecurringExpenseRulesFn: func(ctx context.Context, uid uuid.UUID) ([]db.ListRecurringExpenseRulesRow, error) {
				return nil, errors.New("recurring expenses failed")
			},
		}
		svc := NewService(m, nil)

		_, err := svc.Export(context.Background(), userID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "recurring expenses failed")
	})

	t.Run("recurring incomes query error", func(t *testing.T) {
		m := &mock.MockQuerier{
			ListExpenseCategoriesFn: func(ctx context.Context) ([]db.ExpenseCategory, error) {
				return []db.ExpenseCategory{}, nil
			},
			ListIncomeCategoriesFn: func(ctx context.Context) ([]db.IncomeCategory, error) {
				return []db.IncomeCategory{}, nil
			},
			ListTagsFn: func(ctx context.Context, uid uuid.UUID) ([]db.Tag, error) {
				return []db.Tag{}, nil
			},
			ListAllExpensesByUserFn: func(ctx context.Context, uid uuid.UUID) ([]db.Expense, error) {
				return []db.Expense{}, nil
			},
			ListAllIncomesByUserFn: func(ctx context.Context, uid uuid.UUID) ([]db.Income, error) {
				return []db.Income{}, nil
			},
			ListRecurringExpenseRulesFn: func(ctx context.Context, uid uuid.UUID) ([]db.ListRecurringExpenseRulesRow, error) {
				return []db.ListRecurringExpenseRulesRow{}, nil
			},
			ListRecurringIncomeRulesFn: func(ctx context.Context, uid uuid.UUID) ([]db.Rule, error) {
				return nil, errors.New("recurring incomes failed")
			},
		}
		svc := NewService(m, nil)

		_, err := svc.Export(context.Background(), userID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "recurring incomes failed")
	})

	t.Run("expense tags query error", func(t *testing.T) {
		m := &mock.MockQuerier{
			ListExpenseCategoriesFn: func(ctx context.Context) ([]db.ExpenseCategory, error) {
				return []db.ExpenseCategory{}, nil
			},
			ListIncomeCategoriesFn: func(ctx context.Context) ([]db.IncomeCategory, error) {
				return []db.IncomeCategory{}, nil
			},
			ListTagsFn: func(ctx context.Context, uid uuid.UUID) ([]db.Tag, error) {
				return []db.Tag{}, nil
			},
			ListAllExpensesByUserFn: func(ctx context.Context, uid uuid.UUID) ([]db.Expense, error) {
				return []db.Expense{}, nil
			},
			ListAllIncomesByUserFn: func(ctx context.Context, uid uuid.UUID) ([]db.Income, error) {
				return []db.Income{}, nil
			},
			ListRecurringExpenseRulesFn: func(ctx context.Context, uid uuid.UUID) ([]db.ListRecurringExpenseRulesRow, error) {
				return []db.ListRecurringExpenseRulesRow{}, nil
			},
			ListRecurringIncomeRulesFn: func(ctx context.Context, uid uuid.UUID) ([]db.Rule, error) {
				return []db.Rule{}, nil
			},
			ListAllExpenseTagsByUserFn: func(ctx context.Context, uid uuid.UUID) ([]db.ExpenseTag, error) {
				return nil, errors.New("expense tags failed")
			},
		}
		svc := NewService(m, nil)

		_, err := svc.Export(context.Background(), userID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "expense tags failed")
	})

	t.Run("income tags query error", func(t *testing.T) {
		m := &mock.MockQuerier{
			ListExpenseCategoriesFn: func(ctx context.Context) ([]db.ExpenseCategory, error) {
				return []db.ExpenseCategory{}, nil
			},
			ListIncomeCategoriesFn: func(ctx context.Context) ([]db.IncomeCategory, error) {
				return []db.IncomeCategory{}, nil
			},
			ListTagsFn: func(ctx context.Context, uid uuid.UUID) ([]db.Tag, error) {
				return []db.Tag{}, nil
			},
			ListAllExpensesByUserFn: func(ctx context.Context, uid uuid.UUID) ([]db.Expense, error) {
				return []db.Expense{}, nil
			},
			ListAllIncomesByUserFn: func(ctx context.Context, uid uuid.UUID) ([]db.Income, error) {
				return []db.Income{}, nil
			},
			ListRecurringExpenseRulesFn: func(ctx context.Context, uid uuid.UUID) ([]db.ListRecurringExpenseRulesRow, error) {
				return []db.ListRecurringExpenseRulesRow{}, nil
			},
			ListRecurringIncomeRulesFn: func(ctx context.Context, uid uuid.UUID) ([]db.Rule, error) {
				return []db.Rule{}, nil
			},
			ListAllExpenseTagsByUserFn: func(ctx context.Context, uid uuid.UUID) ([]db.ExpenseTag, error) {
				return []db.ExpenseTag{}, nil
			},
			ListAllIncomeTagsByUserFn: func(ctx context.Context, uid uuid.UUID) ([]db.IncomeTag, error) {
				return nil, errors.New("income tags failed")
			},
		}
		svc := NewService(m, nil)

		_, err := svc.Export(context.Background(), userID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "income tags failed")
	})

	t.Run("export with real data", func(t *testing.T) {
		catID := uuid.New()
		tagID := uuid.New()
		expID := uuid.New()

		m := &mock.MockQuerier{
			ListExpenseCategoriesFn: func(ctx context.Context) ([]db.ExpenseCategory, error) {
				return []db.ExpenseCategory{
					{ID: catID, Name: "Food", Color: pgtype.Text{String: "#FF0000", Valid: true}, IsActive: true},
				}, nil
			},
			ListIncomeCategoriesFn: func(ctx context.Context) ([]db.IncomeCategory, error) {
				return []db.IncomeCategory{}, nil
			},
			ListTagsFn: func(ctx context.Context, uid uuid.UUID) ([]db.Tag, error) {
				return []db.Tag{
					{ID: tagID, UserID: userID, Name: "essential"},
				}, nil
			},
			ListAllExpensesByUserFn: func(ctx context.Context, uid uuid.UUID) ([]db.Expense, error) {
				return []db.Expense{
					{
						ID:          expID,
						UserID:      userID,
						CategoryID:  pgtype.UUID{Bytes: catID, Valid: true},
						Amount:      decimal.NewFromFloat(500),
						Currency:    "PHP",
						Description: "Groceries",
					},
				}, nil
			},
			ListAllIncomesByUserFn: func(ctx context.Context, uid uuid.UUID) ([]db.Income, error) {
				return []db.Income{}, nil
			},
			ListRecurringExpenseRulesFn: func(ctx context.Context, uid uuid.UUID) ([]db.ListRecurringExpenseRulesRow, error) {
				return []db.ListRecurringExpenseRulesRow{}, nil
			},
			ListRecurringIncomeRulesFn: func(ctx context.Context, uid uuid.UUID) ([]db.Rule, error) {
				return []db.Rule{}, nil
			},
			ListAllExpenseTagsByUserFn: func(ctx context.Context, uid uuid.UUID) ([]db.ExpenseTag, error) {
				return []db.ExpenseTag{
					{ExpenseID: expID, TagID: tagID},
				}, nil
			},
			ListAllIncomeTagsByUserFn: func(ctx context.Context, uid uuid.UUID) ([]db.IncomeTag, error) {
				return []db.IncomeTag{}, nil
			},
		}
		svc := NewService(m, nil)

		result, err := svc.Export(context.Background(), userID)
		require.NoError(t, err)
		assert.Len(t, result.ExpenseCategories, 1)
		assert.Equal(t, "Food", result.ExpenseCategories[0].Name)
		assert.Len(t, result.Expenses, 1)
		assert.Equal(t, "500", result.Expenses[0].Amount.String())
		assert.Len(t, result.ExpenseTags, 1)
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
