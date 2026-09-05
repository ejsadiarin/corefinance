package recurring

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

func TestRecurringService_CreateIncomeRule(t *testing.T) {
	userID := uuid.New()
	ruleID := uuid.New()

	t.Run("success", func(t *testing.T) {
		mock := &mock.MockQuerier{
			CreateRecurringIncomeRuleFn: func(ctx context.Context, arg db.CreateRecurringIncomeRuleParams) (db.RecurringIncomeRule, error) {
				assert.Equal(t, userID, arg.UserID)
				assert.True(t, arg.Amount.Equal(decimal.NewFromFloat(5000.50)))
				assert.Equal(t, "PHP", arg.Currency)
				assert.Equal(t, "Monthly Salary", arg.Description)
				assert.Equal(t, "monthly", arg.RecurringType)
				assert.True(t, arg.StartDate.Valid)
				assert.False(t, arg.EndDate.Valid)
				return db.RecurringIncomeRule{
					ID:     ruleID,
					UserID: userID,
					Amount: decimal.NewFromFloat(5000.50),
					Currency:      "PHP",
					Description:   "Monthly Salary",
					RecurringType: "monthly",
				}, nil
			},
		}
		svc := NewService(mock)
		result, err := svc.CreateIncomeRule(context.Background(), userID, CreateIncomeRuleRequest{
			Amount:        5000.50,
			Currency:      "PHP",
			Description:   "Monthly Salary",
			RecurringType: "monthly",
			StartDate:     "2026-01-01",
		})
		require.NoError(t, err)
		assert.Equal(t, ruleID, result.ID)
		assert.Equal(t, "Monthly Salary", result.Description)
	})

	t.Run("with end date", func(t *testing.T) {
		endDate := "2026-12-31"
		mock := &mock.MockQuerier{
			CreateRecurringIncomeRuleFn: func(ctx context.Context, arg db.CreateRecurringIncomeRuleParams) (db.RecurringIncomeRule, error) {
				assert.True(t, arg.EndDate.Valid)
				return db.RecurringIncomeRule{ID: ruleID}, nil
			},
		}
		svc := NewService(mock)
		_, err := svc.CreateIncomeRule(context.Background(), userID, CreateIncomeRuleRequest{
			Amount:        1000,
			Currency:      "USD",
			Description:   "Contract",
			RecurringType: "monthly",
			StartDate:     "2026-01-01",
			EndDate:       &endDate,
		})
		require.NoError(t, err)
	})

	t.Run("error", func(t *testing.T) {
		mock := &mock.MockQuerier{
			CreateRecurringIncomeRuleFn: func(ctx context.Context, arg db.CreateRecurringIncomeRuleParams) (db.RecurringIncomeRule, error) {
				return db.RecurringIncomeRule{}, errors.New("db error")
			},
		}
		svc := NewService(mock)
		_, err := svc.CreateIncomeRule(context.Background(), userID, CreateIncomeRuleRequest{
			Amount:        100,
			Currency:      "PHP",
			Description:   "X",
			RecurringType: "monthly",
			StartDate:     "2026-01-01",
		})
		require.Error(t, err)
	})
}

func TestRecurringService_ListIncomeRules(t *testing.T) {
	userID := uuid.New()

	t.Run("success", func(t *testing.T) {
		mock := &mock.MockQuerier{
			ListRecurringIncomeRulesFn: func(ctx context.Context, uid uuid.UUID) ([]db.RecurringIncomeRule, error) {
				assert.Equal(t, userID, uid)
				return []db.RecurringIncomeRule{
					{ID: uuid.New(), Description: "Salary"},
					{ID: uuid.New(), Description: "Freelance"},
				}, nil
			},
		}
		svc := NewService(mock)
		result, err := svc.ListIncomeRules(context.Background(), userID)
		require.NoError(t, err)
		assert.Len(t, result, 2)
	})

	t.Run("empty", func(t *testing.T) {
		mock := &mock.MockQuerier{
			ListRecurringIncomeRulesFn: func(ctx context.Context, uid uuid.UUID) ([]db.RecurringIncomeRule, error) {
				return []db.RecurringIncomeRule{}, nil
			},
		}
		svc := NewService(mock)
		result, err := svc.ListIncomeRules(context.Background(), userID)
		require.NoError(t, err)
		assert.Empty(t, result)
	})

	t.Run("error", func(t *testing.T) {
		mock := &mock.MockQuerier{
			ListRecurringIncomeRulesFn: func(ctx context.Context, uid uuid.UUID) ([]db.RecurringIncomeRule, error) {
				return nil, errors.New("db error")
			},
		}
		svc := NewService(mock)
		_, err := svc.ListIncomeRules(context.Background(), userID)
		require.Error(t, err)
	})
}

func TestRecurringService_GetIncomeRule(t *testing.T) {
	userID := uuid.New()
	ruleID := uuid.New()

	t.Run("success", func(t *testing.T) {
		mock := &mock.MockQuerier{
			GetRecurringIncomeRuleFn: func(ctx context.Context, arg db.GetRecurringIncomeRuleParams) (db.RecurringIncomeRule, error) {
				assert.Equal(t, ruleID, arg.ID)
				assert.Equal(t, userID, arg.UserID)
				return db.RecurringIncomeRule{
					ID:          ruleID,
					Description: "Salary",
					Amount:      decimal.NewFromFloat(5000),
				}, nil
			},
		}
		svc := NewService(mock)
		result, err := svc.GetIncomeRule(context.Background(), ruleID, userID)
		require.NoError(t, err)
		assert.Equal(t, "Salary", result.Description)
	})

	t.Run("error", func(t *testing.T) {
		mock := &mock.MockQuerier{
			GetRecurringIncomeRuleFn: func(ctx context.Context, arg db.GetRecurringIncomeRuleParams) (db.RecurringIncomeRule, error) {
				return db.RecurringIncomeRule{}, errors.New("not found")
			},
		}
		svc := NewService(mock)
		_, err := svc.GetIncomeRule(context.Background(), ruleID, userID)
		require.Error(t, err)
	})
}

func TestRecurringService_UpdateIncomeRule(t *testing.T) {
	userID := uuid.New()
	ruleID := uuid.New()

	t.Run("success", func(t *testing.T) {
		mock := &mock.MockQuerier{
			UpdateRecurringIncomeRuleFn: func(ctx context.Context, arg db.UpdateRecurringIncomeRuleParams) (db.RecurringIncomeRule, error) {
				assert.Equal(t, ruleID, arg.ID)
				assert.Equal(t, userID, arg.UserID)
				assert.Equal(t, "6000", arg.Amount.String())
				assert.Equal(t, "Updated Salary", arg.Description)
				return db.RecurringIncomeRule{
					ID:          ruleID,
					Description: "Updated Salary",
					Amount:      decimal.NewFromFloat(6000),
				}, nil
			},
		}
		svc := NewService(mock)
		result, err := svc.UpdateIncomeRule(context.Background(), ruleID, userID, UpdateIncomeRuleRequest{
			Amount:        6000,
			Currency:      "PHP",
			Description:   "Updated Salary",
			RecurringType: "monthly",
			StartDate:     "2026-01-01",
		})
		require.NoError(t, err)
		assert.Equal(t, "Updated Salary", result.Description)
	})

	t.Run("error", func(t *testing.T) {
		mock := &mock.MockQuerier{
			UpdateRecurringIncomeRuleFn: func(ctx context.Context, arg db.UpdateRecurringIncomeRuleParams) (db.RecurringIncomeRule, error) {
				return db.RecurringIncomeRule{}, errors.New("db error")
			},
		}
		svc := NewService(mock)
		_, err := svc.UpdateIncomeRule(context.Background(), ruleID, userID, UpdateIncomeRuleRequest{
			Amount:        100,
			Currency:      "PHP",
			Description:   "X",
			RecurringType: "monthly",
			StartDate:     "2026-01-01",
		})
		require.Error(t, err)
	})
}

func TestRecurringService_DeleteIncomeRule(t *testing.T) {
	userID := uuid.New()
	ruleID := uuid.New()

	t.Run("success", func(t *testing.T) {
		mock := &mock.MockQuerier{
			DeleteRecurringIncomeRuleFn: func(ctx context.Context, arg db.DeleteRecurringIncomeRuleParams) error {
				assert.Equal(t, ruleID, arg.ID)
				assert.Equal(t, userID, arg.UserID)
				return nil
			},
		}
		svc := NewService(mock)
		err := svc.DeleteIncomeRule(context.Background(), ruleID, userID)
		require.NoError(t, err)
	})

	t.Run("error", func(t *testing.T) {
		mock := &mock.MockQuerier{
			DeleteRecurringIncomeRuleFn: func(ctx context.Context, arg db.DeleteRecurringIncomeRuleParams) error {
				return errors.New("not found")
			},
		}
		svc := NewService(mock)
		err := svc.DeleteIncomeRule(context.Background(), ruleID, userID)
		require.Error(t, err)
	})
}

func TestRecurringService_CreateExpenseRule(t *testing.T) {
	userID := uuid.New()
	ruleID := uuid.New()

	t.Run("success with category", func(t *testing.T) {
		catID := uuid.New().String()
		notes := "Monthly bills"
		mock := &mock.MockQuerier{
			CreateRecurringExpenseRuleFn: func(ctx context.Context, arg db.CreateRecurringExpenseRuleParams) (db.RecurringExpenseRule, error) {
				assert.Equal(t, userID, arg.UserID)
				assert.Equal(t, "1500.75", arg.Amount.String())
				assert.Equal(t, "PHP", arg.Currency)
				assert.Equal(t, "Rent", arg.Description)
				assert.Equal(t, "monthly", arg.RecurringType)
				assert.Equal(t, "high", arg.Priority)
				assert.True(t, arg.CategoryID.Valid)
				assert.True(t, arg.Notes.Valid)
				assert.Equal(t, "Monthly bills", arg.Notes.String)
				return db.RecurringExpenseRule{
					ID:          ruleID,
					UserID:      userID,
					Description: "Rent",
					Amount:      decimal.NewFromFloat(1500.75),
				}, nil
			},
		}
		svc := NewService(mock)
		result, err := svc.CreateExpenseRule(context.Background(), userID, CreateExpenseRuleRequest{
			Amount:        1500.75,
			Currency:      "PHP",
			Description:   "Rent",
			CategoryID:    &catID,
			Notes:         &notes,
			RecurringType: "monthly",
			StartDate:     "2026-01-01",
			Priority:      "high",
		})
		require.NoError(t, err)
		assert.Equal(t, ruleID, result.ID)
	})

	t.Run("success without optional fields", func(t *testing.T) {
		mock := &mock.MockQuerier{
			CreateRecurringExpenseRuleFn: func(ctx context.Context, arg db.CreateRecurringExpenseRuleParams) (db.RecurringExpenseRule, error) {
				assert.False(t, arg.CategoryID.Valid)
				assert.False(t, arg.Notes.Valid)
				assert.False(t, arg.EndDate.Valid)
				return db.RecurringExpenseRule{ID: ruleID}, nil
			},
		}
		svc := NewService(mock)
		_, err := svc.CreateExpenseRule(context.Background(), userID, CreateExpenseRuleRequest{
			Amount:        100,
			Currency:      "PHP",
			Description:   "Basic",
			RecurringType: "one-time",
			StartDate:     "2026-06-01",
			Priority:      "want",
		})
		require.NoError(t, err)
	})

	t.Run("error", func(t *testing.T) {
		mock := &mock.MockQuerier{
			CreateRecurringExpenseRuleFn: func(ctx context.Context, arg db.CreateRecurringExpenseRuleParams) (db.RecurringExpenseRule, error) {
				return db.RecurringExpenseRule{}, errors.New("db error")
			},
		}
		svc := NewService(mock)
		_, err := svc.CreateExpenseRule(context.Background(), userID, CreateExpenseRuleRequest{
			Amount:        100,
			Currency:      "PHP",
			Description:   "X",
			RecurringType: "monthly",
			StartDate:     "2026-01-01",
			Priority:      "need",
		})
		require.Error(t, err)
	})
}

func TestRecurringService_GetExpenseRule(t *testing.T) {
	userID := uuid.New()
	ruleID := uuid.New()

	t.Run("success", func(t *testing.T) {
		mock := &mock.MockQuerier{
			GetRecurringExpenseRuleFn: func(ctx context.Context, arg db.GetRecurringExpenseRuleParams) (db.GetRecurringExpenseRuleRow, error) {
				assert.Equal(t, ruleID, arg.ID)
				assert.Equal(t, userID, arg.UserID)
				return db.GetRecurringExpenseRuleRow{
					ID:          ruleID,
					Description: "Rent",
					Amount:      decimal.NewFromFloat(1500),
					CategoryName: pgtype.Text{String: "Housing", Valid: true},
				}, nil
			},
		}
		svc := NewService(mock)
		result, err := svc.GetExpenseRule(context.Background(), ruleID, userID)
		require.NoError(t, err)
		assert.Equal(t, "Rent", result.Description)
		assert.Equal(t, "Housing", result.CategoryName.String)
	})

	t.Run("error", func(t *testing.T) {
		mock := &mock.MockQuerier{
			GetRecurringExpenseRuleFn: func(ctx context.Context, arg db.GetRecurringExpenseRuleParams) (db.GetRecurringExpenseRuleRow, error) {
				return db.GetRecurringExpenseRuleRow{}, errors.New("not found")
			},
		}
		svc := NewService(mock)
		_, err := svc.GetExpenseRule(context.Background(), ruleID, userID)
		require.Error(t, err)
	})
}

func TestRecurringService_UpdateExpenseRule(t *testing.T) {
	userID := uuid.New()
	ruleID := uuid.New()

	t.Run("success", func(t *testing.T) {
		catID := uuid.New().String()
		notes := "Updated notes"
		mock := &mock.MockQuerier{
			UpdateRecurringExpenseRuleFn: func(ctx context.Context, arg db.UpdateRecurringExpenseRuleParams) (db.RecurringExpenseRule, error) {
				assert.Equal(t, ruleID, arg.ID)
				assert.Equal(t, userID, arg.UserID)
				assert.Equal(t, "2000", arg.Amount.String())
				assert.Equal(t, "Updated Rent", arg.Description)
				assert.Equal(t, "need", arg.Priority)
				assert.True(t, arg.CategoryID.Valid)
				assert.True(t, arg.Notes.Valid)
				return db.RecurringExpenseRule{
					ID:          ruleID,
					Description: "Updated Rent",
					Amount:      decimal.NewFromFloat(2000),
				}, nil
			},
		}
		svc := NewService(mock)
		result, err := svc.UpdateExpenseRule(context.Background(), ruleID, userID, UpdateExpenseRuleRequest{
			Amount:        2000,
			Currency:      "PHP",
			Description:   "Updated Rent",
			CategoryID:    &catID,
			Notes:         &notes,
			RecurringType: "monthly",
			StartDate:     "2026-01-01",
			Priority:      "need",
			IsActive:      true,
		})
		require.NoError(t, err)
		assert.Equal(t, "Updated Rent", result.Description)
	})

	t.Run("error", func(t *testing.T) {
		mock := &mock.MockQuerier{
			UpdateRecurringExpenseRuleFn: func(ctx context.Context, arg db.UpdateRecurringExpenseRuleParams) (db.RecurringExpenseRule, error) {
				return db.RecurringExpenseRule{}, errors.New("db error")
			},
		}
		svc := NewService(mock)
		_, err := svc.UpdateExpenseRule(context.Background(), ruleID, userID, UpdateExpenseRuleRequest{
			Amount:        100,
			Currency:      "PHP",
			Description:   "X",
			RecurringType: "monthly",
			StartDate:     "2026-01-01",
			Priority:      "want",
		})
		require.Error(t, err)
	})
}

func TestRecurringService_DeleteExpenseRule(t *testing.T) {
	userID := uuid.New()
	ruleID := uuid.New()

	t.Run("success", func(t *testing.T) {
		mock := &mock.MockQuerier{
			DeleteRecurringExpenseRuleFn: func(ctx context.Context, arg db.DeleteRecurringExpenseRuleParams) error {
				assert.Equal(t, ruleID, arg.ID)
				assert.Equal(t, userID, arg.UserID)
				return nil
			},
		}
		svc := NewService(mock)
		err := svc.DeleteExpenseRule(context.Background(), ruleID, userID)
		require.NoError(t, err)
	})

	t.Run("error", func(t *testing.T) {
		mock := &mock.MockQuerier{
			DeleteRecurringExpenseRuleFn: func(ctx context.Context, arg db.DeleteRecurringExpenseRuleParams) error {
				return errors.New("not found")
			},
		}
		svc := NewService(mock)
		err := svc.DeleteExpenseRule(context.Background(), ruleID, userID)
		require.Error(t, err)
	})
}

func TestRecurringService_ListExpenseRules(t *testing.T) {
	userID := uuid.New()

	t.Run("success", func(t *testing.T) {
		mock := &mock.MockQuerier{
			ListRecurringExpenseRulesFn: func(ctx context.Context, uid uuid.UUID) ([]db.ListRecurringExpenseRulesRow, error) {
				assert.Equal(t, userID, uid)
				return []db.ListRecurringExpenseRulesRow{
					{ID: uuid.New(), Description: "Rent", Amount: decimal.NewFromFloat(1500)},
					{ID: uuid.New(), Description: "Utilities", Amount: decimal.NewFromFloat(300)},
					{ID: uuid.New(), Description: "Internet", Amount: decimal.NewFromFloat(100)},
				}, nil
			},
		}
		svc := NewService(mock)
		result, err := svc.ListExpenseRules(context.Background(), userID)
		require.NoError(t, err)
		assert.Len(t, result, 3)
	})

	t.Run("empty", func(t *testing.T) {
		mock := &mock.MockQuerier{
			ListRecurringExpenseRulesFn: func(ctx context.Context, uid uuid.UUID) ([]db.ListRecurringExpenseRulesRow, error) {
				return []db.ListRecurringExpenseRulesRow{}, nil
			},
		}
		svc := NewService(mock)
		result, err := svc.ListExpenseRules(context.Background(), userID)
		require.NoError(t, err)
		assert.Empty(t, result)
	})

	t.Run("error", func(t *testing.T) {
		mock := &mock.MockQuerier{
			ListRecurringExpenseRulesFn: func(ctx context.Context, uid uuid.UUID) ([]db.ListRecurringExpenseRulesRow, error) {
				return nil, errors.New("db error")
			},
		}
		svc := NewService(mock)
		_, err := svc.ListExpenseRules(context.Background(), userID)
		require.Error(t, err)
	})
}
