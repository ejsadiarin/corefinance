package income

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
	incomeID := uuid.New()
	catID := "550e8400-e29b-41d4-a716-446655440000"
	sourceRuleID := "550e8400-e29b-41d4-a716-446655440001"
	notes := "monthly salary"
	startDate := "2026-01-01"
	endDate := "2026-12-31"

	mq := &mock.MockQuerier{
		CreateIncomeFn: func(ctx context.Context, arg db.CreateIncomeParams) (db.Income, error) {
			assert.Equal(t, userID, arg.UserID)
			assert.True(t, arg.CategoryID.Valid)
			assert.Equal(t, decimal.NewFromFloat(50000), arg.Amount)
			assert.Equal(t, "PHP", arg.Currency)
			assert.Equal(t, "Salary", arg.Description)
			assert.True(t, arg.Notes.Valid)
			assert.Equal(t, notes, arg.Notes.String)
			assert.Equal(t, "monthly", arg.RecurringType)
			assert.Equal(t, "high", arg.Priority)
			assert.Equal(t, "active", arg.Status)
			assert.True(t, arg.StartDate.Valid)
			assert.True(t, arg.EndDate.Valid)
			assert.True(t, arg.SourceRuleID.Valid)
			return db.Income{
				ID:          incomeID,
				UserID:      userID,
				Amount:      decimal.NewFromFloat(50000),
				Currency:    "PHP",
				Description: "Salary",
			}, nil
		},
	}
	svc := NewService(mq)

	result, err := svc.Create(context.Background(), userID, CreateRequest{
		CategoryID:    &catID,
		Amount:        50000,
		Currency:      "PHP",
		Description:   "Salary",
		Notes:         &notes,
		Date:          "2026-09-05",
		RecurringType: "monthly",
		Priority:      "high",
		Status:        "active",
		StartDate:     &startDate,
		EndDate:       &endDate,
		SourceRuleID:  &sourceRuleID,
	})

	require.NoError(t, err)
	assert.Equal(t, incomeID, result.ID)
	assert.Equal(t, "PHP", result.Currency)
}

func TestCreate_Error(t *testing.T) {
	userID := uuid.New()
	mq := &mock.MockQuerier{
		CreateIncomeFn: func(ctx context.Context, arg db.CreateIncomeParams) (db.Income, error) {
			return db.Income{}, errors.New("db error")
		},
	}
	svc := NewService(mq)

	_, err := svc.Create(context.Background(), userID, CreateRequest{
		Amount:      10.0,
		Currency:    "PHP",
		Description: "test",
		Date:        "2026-09-05",
	})

	require.Error(t, err)
	assert.EqualError(t, err, "db error")
}

func TestCreate_NilOptionalFields(t *testing.T) {
	userID := uuid.New()
	mq := &mock.MockQuerier{
		CreateIncomeFn: func(ctx context.Context, arg db.CreateIncomeParams) (db.Income, error) {
			assert.False(t, arg.CategoryID.Valid)
			assert.False(t, arg.Notes.Valid)
			assert.False(t, arg.StartDate.Valid)
			assert.False(t, arg.EndDate.Valid)
			assert.False(t, arg.SourceRuleID.Valid)
			return db.Income{ID: uuid.New()}, nil
		},
	}
	svc := NewService(mq)

	_, err := svc.Create(context.Background(), userID, CreateRequest{
		Amount:      10,
		Currency:    "PHP",
		Description: "test",
		Date:        "2026-09-05",
	})

	require.NoError(t, err)
}

func TestGet_Success(t *testing.T) {
	userID := uuid.New()
	incomeID := uuid.New()
	var capturedArg db.GetIncomeParams

	mq := &mock.MockQuerier{
		GetIncomeFn: func(ctx context.Context, arg db.GetIncomeParams) (db.GetIncomeRow, error) {
			capturedArg = arg
			return db.GetIncomeRow{
				ID:          incomeID,
				UserID:      userID,
				Description: "Freelance",
				Amount:      decimal.NewFromFloat(25000),
			}, nil
		},
	}
	svc := NewService(mq)

	result, err := svc.Get(context.Background(), incomeID, userID)

	require.NoError(t, err)
	assert.Equal(t, incomeID, capturedArg.ID)
	assert.Equal(t, userID, capturedArg.UserID)
	assert.Equal(t, "Freelance", result.Description)
}

func TestGet_Error(t *testing.T) {
	mq := &mock.MockQuerier{
		GetIncomeFn: func(ctx context.Context, arg db.GetIncomeParams) (db.GetIncomeRow, error) {
			return db.GetIncomeRow{}, errors.New("not found")
		},
	}
	svc := NewService(mq)

	_, err := svc.Get(context.Background(), uuid.New(), uuid.New())

	require.Error(t, err)
	assert.EqualError(t, err, "not found")
}

func TestList_DefaultPagination(t *testing.T) {
	userID := uuid.New()
	var capturedArg db.ListIncomesParams

	mq := &mock.MockQuerier{
		ListIncomesFn: func(ctx context.Context, arg db.ListIncomesParams) ([]db.ListIncomesRow, error) {
			capturedArg = arg
			return []db.ListIncomesRow{}, nil
		},
	}
	svc := NewService(mq)

	_, err := svc.List(context.Background(), userID, ListParams{})

	require.NoError(t, err)
	assert.Equal(t, int32(50), capturedArg.PageLimit)
	assert.Equal(t, int32(0), capturedArg.PageOffset)
}

func TestList_PaginationPassedThrough(t *testing.T) {
	userID := uuid.New()
	var capturedArg db.ListIncomesParams

	mq := &mock.MockQuerier{
		ListIncomesFn: func(ctx context.Context, arg db.ListIncomesParams) ([]db.ListIncomesRow, error) {
			capturedArg = arg
			return nil, nil
		},
	}
	svc := NewService(mq)

	_, err := svc.List(context.Background(), userID, ListParams{
		Page:     4,
		PageSize: 15,
	})

	require.NoError(t, err)
	assert.Equal(t, int32(15), capturedArg.PageLimit)
	assert.Equal(t, int32(45), capturedArg.PageOffset)
}

func TestList_FiltersPassedThrough(t *testing.T) {
	userID := uuid.New()
	startDate := "2026-01-01"
	endDate := "2026-06-30"
	catID := "550e8400-e29b-41d4-a716-446655440000"
	status := "active"
	recType := "monthly"

	mq := &mock.MockQuerier{
		ListIncomesFn: func(ctx context.Context, arg db.ListIncomesParams) ([]db.ListIncomesRow, error) {
			assert.Equal(t, userID, arg.UserID)
			assert.True(t, arg.StartDate.Valid)
			assert.True(t, arg.EndDate.Valid)
			assert.NotEqual(t, uuid.Nil, arg.CategoryID)
			assert.Equal(t, "active", arg.Status)
			assert.Equal(t, "monthly", arg.RecurringType)
			return nil, nil
		},
	}
	svc := NewService(mq)

	_, err := svc.List(context.Background(), userID, ListParams{
		StartDate:     &startDate,
		EndDate:       &endDate,
		CategoryID:    &catID,
		Status:        &status,
		RecurringType: &recType,
		Page:          1,
		PageSize:      10,
	})

	require.NoError(t, err)
}

func TestList_NegativePageDefaults(t *testing.T) {
	var capturedArg db.ListIncomesParams

	mq := &mock.MockQuerier{
		ListIncomesFn: func(ctx context.Context, arg db.ListIncomesParams) ([]db.ListIncomesRow, error) {
			capturedArg = arg
			return nil, nil
		},
	}
	svc := NewService(mq)

	_, err := svc.List(context.Background(), uuid.New(), ListParams{
		Page:     -3,
		PageSize: -20,
	})

	require.NoError(t, err)
	assert.Equal(t, int32(50), capturedArg.PageLimit)
	assert.Equal(t, int32(0), capturedArg.PageOffset)
}

func TestList_Error(t *testing.T) {
	mq := &mock.MockQuerier{
		ListIncomesFn: func(ctx context.Context, arg db.ListIncomesParams) ([]db.ListIncomesRow, error) {
			return nil, errors.New("query failed")
		},
	}
	svc := NewService(mq)

	_, err := svc.List(context.Background(), uuid.New(), ListParams{})

	require.Error(t, err)
	assert.EqualError(t, err, "query failed")
}

func TestUpdate_Success(t *testing.T) {
	userID := uuid.New()
	incomeID := uuid.New()
	catID := "550e8400-e29b-41d4-a716-446655440000"
	var capturedArg db.UpdateIncomeParams

	mq := &mock.MockQuerier{
		UpdateIncomeFn: func(ctx context.Context, arg db.UpdateIncomeParams) (db.Income, error) {
			capturedArg = arg
			return db.Income{
				ID:          incomeID,
				UserID:      userID,
				Amount:      decimal.NewFromFloat(75000),
				Description: "Updated Salary",
			}, nil
		},
	}
	svc := NewService(mq)

	result, err := svc.Update(context.Background(), incomeID, userID, UpdateRequest{
		CategoryID:    &catID,
		Amount:        75000,
		Currency:      "USD",
		Description:   "Updated Salary",
		Date:          "2026-09-05",
		RecurringType: "monthly",
		Priority:      "high",
		Status:        "active",
	})

	require.NoError(t, err)
	assert.Equal(t, incomeID, capturedArg.ID)
	assert.Equal(t, userID, capturedArg.UserID)
	assert.Equal(t, "Updated Salary", result.Description)
}

func TestUpdate_Error(t *testing.T) {
	mq := &mock.MockQuerier{
		UpdateIncomeFn: func(ctx context.Context, arg db.UpdateIncomeParams) (db.Income, error) {
			return db.Income{}, errors.New("update failed")
		},
	}
	svc := NewService(mq)

	_, err := svc.Update(context.Background(), uuid.New(), uuid.New(), UpdateRequest{
		Amount:      10,
		Currency:    "PHP",
		Description: "x",
		Date:        "2026-09-05",
	})

	require.Error(t, err)
}

func TestUpdate_NilOptionalFields(t *testing.T) {
	userID := uuid.New()
	incomeID := uuid.New()
	mq := &mock.MockQuerier{
		UpdateIncomeFn: func(ctx context.Context, arg db.UpdateIncomeParams) (db.Income, error) {
			assert.False(t, arg.CategoryID.Valid)
			assert.False(t, arg.Notes.Valid)
			assert.False(t, arg.StartDate.Valid)
			assert.False(t, arg.EndDate.Valid)
			assert.False(t, arg.SourceRuleID.Valid)
			return db.Income{ID: incomeID}, nil
		},
	}
	svc := NewService(mq)

	_, err := svc.Update(context.Background(), incomeID, userID, UpdateRequest{
		Amount:      5,
		Currency:    "USD",
		Description: "minimal",
		Date:        "2026-09-05",
	})

	require.NoError(t, err)
}

func TestDelete_Success(t *testing.T) {
	userID := uuid.New()
	incomeID := uuid.New()
	var capturedArg db.DeleteIncomeParams

	mq := &mock.MockQuerier{
		DeleteIncomeFn: func(ctx context.Context, arg db.DeleteIncomeParams) error {
			capturedArg = arg
			return nil
		},
	}
	svc := NewService(mq)

	err := svc.Delete(context.Background(), incomeID, userID)

	require.NoError(t, err)
	assert.Equal(t, incomeID, capturedArg.ID)
	assert.Equal(t, userID, capturedArg.UserID)
}

func TestDelete_Error(t *testing.T) {
	mq := &mock.MockQuerier{
		DeleteIncomeFn: func(ctx context.Context, arg db.DeleteIncomeParams) error {
			return errors.New("delete failed")
		},
	}
	svc := NewService(mq)

	err := svc.Delete(context.Background(), uuid.New(), uuid.New())

	require.Error(t, err)
}

func TestSkip_Success(t *testing.T) {
	userID := uuid.New()
	incomeID := uuid.New()
	var capturedArg db.SkipIncomeParams

	mq := &mock.MockQuerier{
		SkipIncomeFn: func(ctx context.Context, arg db.SkipIncomeParams) error {
			capturedArg = arg
			return nil
		},
	}
	svc := NewService(mq)

	err := svc.Skip(context.Background(), incomeID, userID)

	require.NoError(t, err)
	assert.Equal(t, incomeID, capturedArg.ID)
	assert.Equal(t, userID, capturedArg.UserID)
}

func TestSkip_Error(t *testing.T) {
	mq := &mock.MockQuerier{
		SkipIncomeFn: func(ctx context.Context, arg db.SkipIncomeParams) error {
			return errors.New("skip failed")
		},
	}
	svc := NewService(mq)

	err := svc.Skip(context.Background(), uuid.New(), uuid.New())

	require.Error(t, err)
}

func TestCheckSkipped_Success(t *testing.T) {
	userID := uuid.New()
	var capturedArg db.CheckSkippedIncomeParams

	mq := &mock.MockQuerier{
		CheckSkippedIncomeFn: func(ctx context.Context, arg db.CheckSkippedIncomeParams) (bool, error) {
			capturedArg = arg
			return true, nil
		},
	}
	svc := NewService(mq)

	result, err := svc.CheckSkipped(context.Background(), userID, "2026-09-01", "2026-09-30")

	require.NoError(t, err)
	assert.True(t, result)
	assert.Equal(t, userID, capturedArg.UserID)
	assert.True(t, capturedArg.Date.Valid)
	assert.True(t, capturedArg.Date_2.Valid)
}

func TestCheckSkipped_Error(t *testing.T) {
	mq := &mock.MockQuerier{
		CheckSkippedIncomeFn: func(ctx context.Context, arg db.CheckSkippedIncomeParams) (bool, error) {
			return false, errors.New("check failed")
		},
	}
	svc := NewService(mq)

	_, err := svc.CheckSkipped(context.Background(), uuid.New(), "2026-09-01", "2026-09-30")

	require.Error(t, err)
}

func TestOccurrences_WithDates(t *testing.T) {
	userID := uuid.New()
	startDate := "2026-01-01"
	endDate := "2026-06-30"
	var capturedArg db.GetIncomeOccurrencesParams

	mq := &mock.MockQuerier{
		GetIncomeOccurrencesFn: func(ctx context.Context, arg db.GetIncomeOccurrencesParams) ([]db.Income, error) {
			capturedArg = arg
			return []db.Income{
				{ID: uuid.New(), Description: "Salary", Amount: decimal.NewFromFloat(50000)},
			}, nil
		},
	}
	svc := NewService(mq)

	result, err := svc.Occurrences(context.Background(), userID, &startDate, &endDate)

	require.NoError(t, err)
	require.Len(t, result, 1)
	assert.Equal(t, "Salary", result[0].Description)
	assert.Equal(t, userID, capturedArg.UserID)
	assert.True(t, capturedArg.StartDate.Valid)
	assert.True(t, capturedArg.EndDate.Valid)
}

func TestOccurrences_NilDates(t *testing.T) {
	userID := uuid.New()
	var capturedArg db.GetIncomeOccurrencesParams

	mq := &mock.MockQuerier{
		GetIncomeOccurrencesFn: func(ctx context.Context, arg db.GetIncomeOccurrencesParams) ([]db.Income, error) {
			capturedArg = arg
			return nil, nil
		},
	}
	svc := NewService(mq)

	_, err := svc.Occurrences(context.Background(), userID, nil, nil)

	require.NoError(t, err)
	assert.False(t, capturedArg.StartDate.Valid)
	assert.False(t, capturedArg.EndDate.Valid)
}

func TestOccurrences_Error(t *testing.T) {
	mq := &mock.MockQuerier{
		GetIncomeOccurrencesFn: func(ctx context.Context, arg db.GetIncomeOccurrencesParams) ([]db.Income, error) {
			return nil, errors.New("occurrences failed")
		},
	}
	svc := NewService(mq)

	_, err := svc.Occurrences(context.Background(), uuid.New(), nil, nil)

	require.Error(t, err)
}

func TestTotalByDateRange_Success(t *testing.T) {
	userID := uuid.New()
	var capturedArg db.GetTotalIncomesByDateRangeParams
	expectedTotal := decimal.NewFromFloat(250000)

	mq := &mock.MockQuerier{
		GetTotalIncomesByDateRangeFn: func(ctx context.Context, arg db.GetTotalIncomesByDateRangeParams) (decimal.Decimal, error) {
			capturedArg = arg
			return expectedTotal, nil
		},
	}
	svc := NewService(mq)

	result, err := svc.TotalByDateRange(context.Background(), userID, "2026-01-01", "2026-12-31")

	require.NoError(t, err)
	assert.True(t, expectedTotal.Equal(result))
	assert.Equal(t, userID, capturedArg.UserID)
	assert.True(t, capturedArg.Date.Valid)
	assert.True(t, capturedArg.Date_2.Valid)
}

func TestTotalByDateRange_Error(t *testing.T) {
	mq := &mock.MockQuerier{
		GetTotalIncomesByDateRangeFn: func(ctx context.Context, arg db.GetTotalIncomesByDateRangeParams) (decimal.Decimal, error) {
			return decimal.Zero, errors.New("total failed")
		},
	}
	svc := NewService(mq)

	_, err := svc.TotalByDateRange(context.Background(), uuid.New(), "2026-01-01", "2026-12-31")

	require.Error(t, err)
}
