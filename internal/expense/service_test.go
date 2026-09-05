package expense

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	db "github.com/ejsadiarin/corefinance/internal/db/sqlc"
	"github.com/ejsadiarin/corefinance/internal/mock"
)

func TestCreate_Success(t *testing.T) {
	userID := uuid.New()
	expenseID := uuid.New()
	catID := "550e8400-e29b-41d4-a716-446655440000"
	sourceRuleID := "550e8400-e29b-41d4-a716-446655440001"
	notes := "lunch money"
	startDate := "2026-01-01"
	endDate := "2026-12-31"

	mock := &mock.MockQuerier{
		CreateExpenseFn: func(ctx context.Context, arg db.CreateExpenseParams) (db.Expense, error) {
			assert.Equal(t, userID, arg.UserID)
			assert.True(t, arg.CategoryID.Valid)
			assert.Equal(t, decimal.NewFromFloat(42.50), arg.Amount)
			assert.Equal(t, "PHP", arg.Currency)
			assert.Equal(t, "Food", arg.Description)
			assert.True(t, arg.Notes.Valid)
			assert.Equal(t, notes, arg.Notes.String)
			assert.Equal(t, "recurring", arg.RecurringType)
			assert.Equal(t, "high", arg.Priority)
			assert.Equal(t, "active", arg.Status)
			assert.True(t, arg.IsDebt)
			assert.True(t, arg.StartDate.Valid)
			assert.True(t, arg.EndDate.Valid)
			assert.True(t, arg.SourceRuleID.Valid)
			return db.Expense{
				ID:          expenseID,
				UserID:      userID,
				Amount:      decimal.NewFromFloat(42.50),
				Currency:    "PHP",
				Description: "Food",
			}, nil
		},
	}
	svc := NewService(mock)

	result, err := svc.Create(context.Background(), userID, CreateRequest{
		CategoryID:    &catID,
		Amount:        42.50,
		Currency:      "PHP",
		Description:   "Food",
		Notes:         &notes,
		ExpenseDate:   "2026-09-05",
		RecurringType: "recurring",
		Priority:      "high",
		Status:        "active",
		IsDebt:        true,
		StartDate:     &startDate,
		EndDate:       &endDate,
		SourceRuleID:  &sourceRuleID,
	})

	require.NoError(t, err)
	assert.Equal(t, expenseID, result.ID)
	assert.Equal(t, "PHP", result.Currency)
}

func TestCreate_Error(t *testing.T) {
	userID := uuid.New()
	mock := &mock.MockQuerier{
		CreateExpenseFn: func(ctx context.Context, arg db.CreateExpenseParams) (db.Expense, error) {
			return db.Expense{}, errors.New("db error")
		},
	}
	svc := NewService(mock)

	_, err := svc.Create(context.Background(), userID, CreateRequest{
		Amount:      10.0,
		Currency:    "PHP",
		Description: "test",
		ExpenseDate: "2026-09-05",
	})

	require.Error(t, err)
	assert.EqualError(t, err, "db error")
}

func TestGet_Success(t *testing.T) {
	userID := uuid.New()
	expenseID := uuid.New()
	var capturedArg db.GetExpenseParams

	mock := &mock.MockQuerier{
		GetExpenseFn: func(ctx context.Context, arg db.GetExpenseParams) (db.GetExpenseRow, error) {
			capturedArg = arg
			return db.GetExpenseRow{
				ID:          expenseID,
				UserID:      userID,
				Description: "Electricity",
				Amount:      decimal.NewFromFloat(1500),
			}, nil
		},
	}
	svc := NewService(mock)

	result, err := svc.Get(context.Background(), expenseID, userID)

	require.NoError(t, err)
	assert.Equal(t, expenseID, capturedArg.ID)
	assert.Equal(t, userID, capturedArg.UserID)
	assert.Equal(t, "Electricity", result.Description)
}

func TestGet_Error(t *testing.T) {
	mock := &mock.MockQuerier{
		GetExpenseFn: func(ctx context.Context, arg db.GetExpenseParams) (db.GetExpenseRow, error) {
			return db.GetExpenseRow{}, errors.New("not found")
		},
	}
	svc := NewService(mock)

	_, err := svc.Get(context.Background(), uuid.New(), uuid.New())

	require.Error(t, err)
	assert.EqualError(t, err, "not found")
}

func TestList_DefaultPagination(t *testing.T) {
	userID := uuid.New()
	var capturedArg db.ListExpensesParams

	mock := &mock.MockQuerier{
		ListExpensesFn: func(ctx context.Context, arg db.ListExpensesParams) ([]db.ListExpensesRow, error) {
			capturedArg = arg
			return []db.ListExpensesRow{}, nil
		},
	}
	svc := NewService(mock)

	_, err := svc.List(context.Background(), userID, ListParams{})

	require.NoError(t, err)
	assert.Equal(t, int32(50), capturedArg.PageLimit)
	assert.Equal(t, int32(0), capturedArg.PageOffset)
}

func TestList_PaginationPassedThrough(t *testing.T) {
	userID := uuid.New()
	var capturedArg db.ListExpensesParams

	mock := &mock.MockQuerier{
		ListExpensesFn: func(ctx context.Context, arg db.ListExpensesParams) ([]db.ListExpensesRow, error) {
			capturedArg = arg
			return nil, nil
		},
	}
	svc := NewService(mock)

	_, err := svc.List(context.Background(), userID, ListParams{
		Page:     3,
		PageSize: 20,
	})

	require.NoError(t, err)
	assert.Equal(t, int32(20), capturedArg.PageLimit)
	assert.Equal(t, int32(40), capturedArg.PageOffset)
}

func TestList_FiltersPassedThrough(t *testing.T) {
	userID := uuid.New()
	startDate := "2026-01-01"
	endDate := "2026-06-30"
	catID := "550e8400-e29b-41d4-a716-446655440000"
	priority := "high"
	status := "active"
	recType := "monthly"

	mock := &mock.MockQuerier{
		ListExpensesFn: func(ctx context.Context, arg db.ListExpensesParams) ([]db.ListExpensesRow, error) {
			assert.Equal(t, userID, arg.UserID)
			assert.True(t, arg.StartDate.Valid)
			assert.True(t, arg.EndDate.Valid)
			assert.NotEqual(t, uuid.Nil, arg.CategoryID)
			assert.Equal(t, "high", arg.Priority)
			assert.Equal(t, "active", arg.Status)
			assert.Equal(t, "monthly", arg.RecurringType)
			return nil, nil
		},
	}
	svc := NewService(mock)

	_, err := svc.List(context.Background(), userID, ListParams{
		StartDate:     &startDate,
		EndDate:       &endDate,
		CategoryID:    &catID,
		Priority:      &priority,
		Status:        &status,
		RecurringType: &recType,
		Page:          1,
		PageSize:      10,
	})

	require.NoError(t, err)
}

func TestList_NegativePageDefaults(t *testing.T) {
	var capturedArg db.ListExpensesParams

	mock := &mock.MockQuerier{
		ListExpensesFn: func(ctx context.Context, arg db.ListExpensesParams) ([]db.ListExpensesRow, error) {
			capturedArg = arg
			return nil, nil
		},
	}
	svc := NewService(mock)

	_, err := svc.List(context.Background(), uuid.New(), ListParams{
		Page:     -5,
		PageSize: -10,
	})

	require.NoError(t, err)
	assert.Equal(t, int32(50), capturedArg.PageLimit)
	assert.Equal(t, int32(0), capturedArg.PageOffset)
}

func TestList_Error(t *testing.T) {
	mock := &mock.MockQuerier{
		ListExpensesFn: func(ctx context.Context, arg db.ListExpensesParams) ([]db.ListExpensesRow, error) {
			return nil, errors.New("query failed")
		},
	}
	svc := NewService(mock)

	_, err := svc.List(context.Background(), uuid.New(), ListParams{})

	require.Error(t, err)
	assert.EqualError(t, err, "query failed")
}

func TestSearch_DefaultPagination(t *testing.T) {
	var capturedArg db.SearchExpensesParams

	mock := &mock.MockQuerier{
		SearchExpensesFn: func(ctx context.Context, arg db.SearchExpensesParams) ([]db.SearchExpensesRow, error) {
			capturedArg = arg
			return nil, nil
		},
	}
	svc := NewService(mock)

	_, err := svc.Search(context.Background(), uuid.New(), SearchParams{
		Query: "coffee",
	})

	require.NoError(t, err)
	assert.Equal(t, int32(50), capturedArg.PageLimit)
	assert.Equal(t, int32(0), capturedArg.PageOffset)
	assert.True(t, capturedArg.Query.Valid)
	assert.Equal(t, "coffee", capturedArg.Query.String)
}

func TestSearch_PaginationPassedThrough(t *testing.T) {
	var capturedArg db.SearchExpensesParams

	mock := &mock.MockQuerier{
		SearchExpensesFn: func(ctx context.Context, arg db.SearchExpensesParams) ([]db.SearchExpensesRow, error) {
			capturedArg = arg
			return nil, nil
		},
	}
	svc := NewService(mock)

	_, err := svc.Search(context.Background(), uuid.New(), SearchParams{
		Query:    "lunch",
		Page:     2,
		PageSize: 10,
	})

	require.NoError(t, err)
	assert.Equal(t, int32(10), capturedArg.PageLimit)
	assert.Equal(t, int32(10), capturedArg.PageOffset)
}

func TestSearch_Error(t *testing.T) {
	mock := &mock.MockQuerier{
		SearchExpensesFn: func(ctx context.Context, arg db.SearchExpensesParams) ([]db.SearchExpensesRow, error) {
			return nil, errors.New("search failed")
		},
	}
	svc := NewService(mock)

	_, err := svc.Search(context.Background(), uuid.New(), SearchParams{Query: "test"})

	require.Error(t, err)
}

func TestUpdate_Success(t *testing.T) {
	userID := uuid.New()
	expenseID := uuid.New()
	catID := "550e8400-e29b-41d4-a716-446655440000"
	var capturedArg db.UpdateExpenseParams

	mock := &mock.MockQuerier{
		UpdateExpenseFn: func(ctx context.Context, arg db.UpdateExpenseParams) (db.Expense, error) {
			capturedArg = arg
			return db.Expense{
				ID:          expenseID,
				UserID:      userID,
				Amount:      decimal.NewFromFloat(99.99),
				Description: "Updated",
			}, nil
		},
	}
	svc := NewService(mock)

	result, err := svc.Update(context.Background(), expenseID, userID, UpdateRequest{
		CategoryID:    &catID,
		Amount:        99.99,
		Currency:      "USD",
		Description:   "Updated",
		ExpenseDate:   "2026-09-05",
		RecurringType: "one-time",
		Priority:      "low",
		Status:        "active",
	})

	require.NoError(t, err)
	assert.Equal(t, expenseID, capturedArg.ID)
	assert.Equal(t, userID, capturedArg.UserID)
	assert.Equal(t, "Updated", result.Description)
}

func TestUpdate_Error(t *testing.T) {
	mock := &mock.MockQuerier{
		UpdateExpenseFn: func(ctx context.Context, arg db.UpdateExpenseParams) (db.Expense, error) {
			return db.Expense{}, errors.New("update failed")
		},
	}
	svc := NewService(mock)

	_, err := svc.Update(context.Background(), uuid.New(), uuid.New(), UpdateRequest{
		Amount:      10,
		Currency:    "PHP",
		Description: "x",
		ExpenseDate: "2026-09-05",
	})

	require.Error(t, err)
}

func TestDelete_Success(t *testing.T) {
	userID := uuid.New()
	expenseID := uuid.New()
	var capturedArg db.DeleteExpenseParams

	mock := &mock.MockQuerier{
		DeleteExpenseFn: func(ctx context.Context, arg db.DeleteExpenseParams) error {
			capturedArg = arg
			return nil
		},
	}
	svc := NewService(mock)

	err := svc.Delete(context.Background(), expenseID, userID)

	require.NoError(t, err)
	assert.Equal(t, expenseID, capturedArg.ID)
	assert.Equal(t, userID, capturedArg.UserID)
}

func TestDelete_Error(t *testing.T) {
	mock := &mock.MockQuerier{
		DeleteExpenseFn: func(ctx context.Context, arg db.DeleteExpenseParams) error {
			return errors.New("delete failed")
		},
	}
	svc := NewService(mock)

	err := svc.Delete(context.Background(), uuid.New(), uuid.New())

	require.Error(t, err)
}

func TestSkip_Success(t *testing.T) {
	userID := uuid.New()
	expenseID := uuid.New()
	var capturedArg db.SkipExpenseParams

	mock := &mock.MockQuerier{
		SkipExpenseFn: func(ctx context.Context, arg db.SkipExpenseParams) error {
			capturedArg = arg
			return nil
		},
	}
	svc := NewService(mock)

	err := svc.Skip(context.Background(), expenseID, userID)

	require.NoError(t, err)
	assert.Equal(t, expenseID, capturedArg.ID)
	assert.Equal(t, userID, capturedArg.UserID)
}

func TestSkip_Error(t *testing.T) {
	mock := &mock.MockQuerier{
		SkipExpenseFn: func(ctx context.Context, arg db.SkipExpenseParams) error {
			return errors.New("skip failed")
		},
	}
	svc := NewService(mock)

	err := svc.Skip(context.Background(), uuid.New(), uuid.New())

	require.Error(t, err)
}

func TestCheckSkipped_Success(t *testing.T) {
	userID := uuid.New()
	var capturedArg db.CheckSkippedExpenseParams

	mock := &mock.MockQuerier{
		CheckSkippedExpenseFn: func(ctx context.Context, arg db.CheckSkippedExpenseParams) (bool, error) {
			capturedArg = arg
			return true, nil
		},
	}
	svc := NewService(mock)

	result, err := svc.CheckSkipped(context.Background(), userID, "2026-09-01", "2026-09-30")

	require.NoError(t, err)
	assert.True(t, result)
	assert.Equal(t, userID, capturedArg.UserID)
	assert.True(t, capturedArg.ExpenseDate.Valid)
	assert.True(t, capturedArg.ExpenseDate_2.Valid)
}

func TestCheckSkipped_Error(t *testing.T) {
	mock := &mock.MockQuerier{
		CheckSkippedExpenseFn: func(ctx context.Context, arg db.CheckSkippedExpenseParams) (bool, error) {
			return false, errors.New("check failed")
		},
	}
	svc := NewService(mock)

	_, err := svc.CheckSkipped(context.Background(), uuid.New(), "2026-09-01", "2026-09-30")

	require.Error(t, err)
}

func TestStatsByCategory_WithDates(t *testing.T) {
	userID := uuid.New()
	startDate := "2026-01-01"
	endDate := "2026-06-30"
	var capturedArg db.GetExpenseStatsByCategoryParams

	mock := &mock.MockQuerier{
		GetExpenseStatsByCategoryFn: func(ctx context.Context, arg db.GetExpenseStatsByCategoryParams) ([]db.GetExpenseStatsByCategoryRow, error) {
			capturedArg = arg
			return []db.GetExpenseStatsByCategoryRow{
				{
					CategoryID:   uuid.New(),
					CategoryName: "Food",
					ExpenseCount: 15,
					TotalAmount:  decimal.NewFromFloat(5000),
				},
			}, nil
		},
	}
	svc := NewService(mock)

	result, err := svc.StatsByCategory(context.Background(), userID, &startDate, &endDate)

	require.NoError(t, err)
	require.Len(t, result, 1)
	assert.Equal(t, "Food", result[0].CategoryName)
	assert.Equal(t, int64(15), result[0].ExpenseCount)
	assert.Equal(t, userID, capturedArg.UserID)
	assert.True(t, capturedArg.StartDate.Valid)
	assert.True(t, capturedArg.EndDate.Valid)
}

func TestStatsByCategory_NilDates(t *testing.T) {
	userID := uuid.New()
	var capturedArg db.GetExpenseStatsByCategoryParams

	mock := &mock.MockQuerier{
		GetExpenseStatsByCategoryFn: func(ctx context.Context, arg db.GetExpenseStatsByCategoryParams) ([]db.GetExpenseStatsByCategoryRow, error) {
			capturedArg = arg
			return nil, nil
		},
	}
	svc := NewService(mock)

	_, err := svc.StatsByCategory(context.Background(), userID, nil, nil)

	require.NoError(t, err)
	assert.False(t, capturedArg.StartDate.Valid)
	assert.False(t, capturedArg.EndDate.Valid)
}

func TestStatsByCategory_Error(t *testing.T) {
	mock := &mock.MockQuerier{
		GetExpenseStatsByCategoryFn: func(ctx context.Context, arg db.GetExpenseStatsByCategoryParams) ([]db.GetExpenseStatsByCategoryRow, error) {
			return nil, errors.New("stats failed")
		},
	}
	svc := NewService(mock)

	_, err := svc.StatsByCategory(context.Background(), uuid.New(), nil, nil)

	require.Error(t, err)
}

func TestTotalByDateRange_Success(t *testing.T) {
	userID := uuid.New()
	var capturedArg db.GetTotalExpensesByDateRangeParams
	expectedTotal := decimal.NewFromFloat(15750.50)

	mock := &mock.MockQuerier{
		GetTotalExpensesByDateRangeFn: func(ctx context.Context, arg db.GetTotalExpensesByDateRangeParams) (decimal.Decimal, error) {
			capturedArg = arg
			return expectedTotal, nil
		},
	}
	svc := NewService(mock)

	result, err := svc.TotalByDateRange(context.Background(), userID, "2026-01-01", "2026-12-31")

	require.NoError(t, err)
	assert.True(t, expectedTotal.Equal(result))
	assert.Equal(t, userID, capturedArg.UserID)
	assert.True(t, capturedArg.ExpenseDate.Valid)
	assert.True(t, capturedArg.ExpenseDate_2.Valid)
}

func TestTotalByDateRange_Error(t *testing.T) {
	mock := &mock.MockQuerier{
		GetTotalExpensesByDateRangeFn: func(ctx context.Context, arg db.GetTotalExpensesByDateRangeParams) (decimal.Decimal, error) {
			return decimal.Zero, errors.New("total failed")
		},
	}
	svc := NewService(mock)

	_, err := svc.TotalByDateRange(context.Background(), uuid.New(), "2026-01-01", "2026-12-31")

	require.Error(t, err)
}

func TestCreate_NilOptionalFields(t *testing.T) {
	userID := uuid.New()
	mock := &mock.MockQuerier{
		CreateExpenseFn: func(ctx context.Context, arg db.CreateExpenseParams) (db.Expense, error) {
			assert.False(t, arg.CategoryID.Valid)
			assert.False(t, arg.Notes.Valid)
			assert.False(t, arg.StartDate.Valid)
			assert.False(t, arg.EndDate.Valid)
			assert.False(t, arg.SourceRuleID.Valid)
			assert.False(t, arg.IsDebt)
			return db.Expense{ID: uuid.New()}, nil
		},
	}
	svc := NewService(mock)

	_, err := svc.Create(context.Background(), userID, CreateRequest{
		Amount:      10,
		Currency:    "PHP",
		Description: "test",
		ExpenseDate: "2026-09-05",
	})

	require.NoError(t, err)
}

func TestUpdate_NilOptionalFields(t *testing.T) {
	userID := uuid.New()
	expenseID := uuid.New()
	mock := &mock.MockQuerier{
		UpdateExpenseFn: func(ctx context.Context, arg db.UpdateExpenseParams) (db.Expense, error) {
			assert.False(t, arg.CategoryID.Valid)
			assert.False(t, arg.Notes.Valid)
			assert.False(t, arg.StartDate.Valid)
			assert.False(t, arg.EndDate.Valid)
			assert.False(t, arg.SourceRuleID.Valid)
			return db.Expense{ID: expenseID}, nil
		},
	}
	svc := NewService(mock)

	_, err := svc.Update(context.Background(), expenseID, userID, UpdateRequest{
		Amount:      5,
		Currency:    "USD",
		Description: "minimal",
		ExpenseDate: "2026-09-05",
	})

	require.NoError(t, err)
}
