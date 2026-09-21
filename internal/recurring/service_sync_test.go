package recurring

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
)

// todayPg returns today's date truncated to midnight, as the service's
// time.Now() and the test's time.Now() fall on the same calendar day.
func todayPg() pgtype.Date {
	now := time.Now()
	return pgtype.Date{
		Time:  time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location()),
		Valid: true,
	}
}

func ptrBool(b bool) *bool { return &b }

func TestRecurringService_CreateExpenseRuleSyncsTodayInstance(t *testing.T) {
	userID := uuid.New()
	ruleID := uuid.New()
	today := todayPg()

	t.Run("due today inserts instance", func(t *testing.T) {
		instanceCalled := false
		m := &mock.MockQuerier{
			CreateRecurringExpenseRuleFn: func(ctx context.Context, arg db.CreateRecurringExpenseRuleParams) (db.RecurringExpenseRule, error) {
				return db.RecurringExpenseRule{
					ID:            ruleID,
					UserID:        userID,
					Description:   "Coffee",
					Amount:        decimal.NewFromFloat(150),
					Currency:      "PHP",
					RecurringType: "daily",
					StartDate:     today,
					Priority:      "want",
					IsActive:      true,
				}, nil
			},
			CreateExpenseInstanceFn: func(ctx context.Context, arg db.CreateExpenseInstanceParams) (int64, error) {
				instanceCalled = true
				assert.Equal(t, userID, arg.UserID)
				assert.Equal(t, ruleID, uuid.UUID(arg.SourceRuleID.Bytes))
				assert.True(t, arg.ExpenseDate.Valid)
				assert.True(t, arg.ExpenseDate.Time.Equal(today.Time))
				return 1, nil
			},
		}
		svc := NewService(m)
		result, err := svc.CreateExpenseRule(context.Background(), userID, CreateExpenseRuleRequest{
			Amount:        150,
			Currency:      "PHP",
			Description:   "Coffee",
			RecurringType: "daily",
			StartDate:     today.Time.Format("2006-01-02"),
			Priority:      "want",
		})
		require.NoError(t, err)
		assert.Equal(t, ruleID, result.ID)
		assert.True(t, instanceCalled, "due-today rule must trigger a sync insert")
	})

	t.Run("not due inserts nothing", func(t *testing.T) {
		future := today
		future.Time = future.Time.AddDate(0, 0, 1)
		m := &mock.MockQuerier{
			CreateRecurringExpenseRuleFn: func(ctx context.Context, arg db.CreateRecurringExpenseRuleParams) (db.RecurringExpenseRule, error) {
				return db.RecurringExpenseRule{
					ID:            ruleID,
					UserID:        userID,
					RecurringType: "daily",
					StartDate:     future,
					Priority:      "want",
					IsActive:      true,
				}, nil
			},
			CreateExpenseInstanceFn: func(ctx context.Context, arg db.CreateExpenseInstanceParams) (int64, error) {
				t.Error("sync insert must not run for a rule starting in the future")
				return 0, nil
			},
		}
		svc := NewService(m)
		_, err := svc.CreateExpenseRule(context.Background(), userID, CreateExpenseRuleRequest{
			Amount:        150,
			Currency:      "PHP",
			Description:   "Coffee",
			RecurringType: "daily",
			StartDate:     future.Time.Format("2006-01-02"),
			Priority:      "want",
		})
		require.NoError(t, err)
	})

	t.Run("sync failure keeps rule write", func(t *testing.T) {
		m := &mock.MockQuerier{
			CreateRecurringExpenseRuleFn: func(ctx context.Context, arg db.CreateRecurringExpenseRuleParams) (db.RecurringExpenseRule, error) {
				return db.RecurringExpenseRule{
					ID:            ruleID,
					UserID:        userID,
					Description:   "Coffee",
					RecurringType: "daily",
					StartDate:     today,
					Priority:      "want",
					IsActive:      true,
				}, nil
			},
			CreateExpenseInstanceFn: func(ctx context.Context, arg db.CreateExpenseInstanceParams) (int64, error) {
				return 0, errors.New("db error")
			},
		}
		svc := NewService(m)
		result, err := svc.CreateExpenseRule(context.Background(), userID, CreateExpenseRuleRequest{
			Amount:        150,
			Currency:      "PHP",
			Description:   "Coffee",
			RecurringType: "daily",
			StartDate:     today.Time.Format("2006-01-02"),
			Priority:      "want",
		})
		require.NoError(t, err, "sync failure must not fail the rule write")
		assert.Equal(t, ruleID, result.ID)
	})
}

func TestRecurringService_UpdateExpenseRuleSyncsTodayInstance(t *testing.T) {
	userID := uuid.New()
	ruleID := uuid.New()
	today := todayPg()

	t.Run("update becomes due today inserts instance", func(t *testing.T) {
		instanceCalled := false
		m := &mock.MockQuerier{
			UpdateRecurringExpenseRuleFn: func(ctx context.Context, arg db.UpdateRecurringExpenseRuleParams) (db.RecurringExpenseRule, error) {
				return db.RecurringExpenseRule{
					ID:            ruleID,
					UserID:        userID,
					Description:   "Rent",
					RecurringType: "daily",
					StartDate:     today,
					Priority:      "need",
					IsActive:      true,
				}, nil
			},
			CreateExpenseInstanceFn: func(ctx context.Context, arg db.CreateExpenseInstanceParams) (int64, error) {
				instanceCalled = true
				assert.Equal(t, ruleID, uuid.UUID(arg.SourceRuleID.Bytes))
				return 1, nil
			},
		}
		svc := NewService(m)
		result, err := svc.UpdateExpenseRule(context.Background(), ruleID, userID, UpdateExpenseRuleRequest{
			Amount:        1500,
			Currency:      "PHP",
			Description:   "Rent",
			RecurringType: "daily",
			StartDate:     today.Time.Format("2006-01-02"),
			Priority:      "need",
			IsActive:      ptrBool(true),
		})
		require.NoError(t, err)
		assert.Equal(t, ruleID, result.ID)
		assert.True(t, instanceCalled, "update making rule due today must trigger a sync insert")
	})
}

func TestRecurringService_CreateIncomeRuleSyncsTodayInstance(t *testing.T) {
	userID := uuid.New()
	ruleID := uuid.New()
	today := todayPg()

	t.Run("due today inserts instance", func(t *testing.T) {
		instanceCalled := false
		m := &mock.MockQuerier{
			CreateRecurringIncomeRuleFn: func(ctx context.Context, arg db.CreateRecurringIncomeRuleParams) (db.RecurringIncomeRule, error) {
				return db.RecurringIncomeRule{
					ID:            ruleID,
					UserID:        userID,
					Amount:        decimal.NewFromFloat(5000),
					Currency:      "PHP",
					Description:   "Salary",
					RecurringType: "daily",
					StartDate:     today,
					IsActive:      true,
				}, nil
			},
			CreateIncomeInstanceFn: func(ctx context.Context, arg db.CreateIncomeInstanceParams) (int64, error) {
				instanceCalled = true
				assert.Equal(t, userID, arg.UserID)
				assert.Equal(t, ruleID, uuid.UUID(arg.SourceRuleID.Bytes))
				assert.True(t, arg.Date.Time.Equal(today.Time))
				return 1, nil
			},
		}
		svc := NewService(m)
		result, err := svc.CreateIncomeRule(context.Background(), userID, CreateIncomeRuleRequest{
			Amount:        5000,
			Currency:      "PHP",
			Description:   "Salary",
			RecurringType: "daily",
			StartDate:     today.Time.Format("2006-01-02"),
		})
		require.NoError(t, err)
		assert.Equal(t, ruleID, result.ID)
		assert.True(t, instanceCalled, "due-today rule must trigger a sync insert")
	})

	t.Run("not due inserts nothing", func(t *testing.T) {
		future := today
		future.Time = future.Time.AddDate(0, 0, 1)
		m := &mock.MockQuerier{
			CreateRecurringIncomeRuleFn: func(ctx context.Context, arg db.CreateRecurringIncomeRuleParams) (db.RecurringIncomeRule, error) {
				return db.RecurringIncomeRule{
					ID:            ruleID,
					UserID:        userID,
					RecurringType: "daily",
					StartDate:     future,
					IsActive:      true,
				}, nil
			},
			CreateIncomeInstanceFn: func(ctx context.Context, arg db.CreateIncomeInstanceParams) (int64, error) {
				t.Error("sync insert must not run for a rule starting in the future")
				return 0, nil
			},
		}
		svc := NewService(m)
		_, err := svc.CreateIncomeRule(context.Background(), userID, CreateIncomeRuleRequest{
			Amount:        5000,
			Currency:      "PHP",
			Description:   "Salary",
			RecurringType: "daily",
			StartDate:     future.Time.Format("2006-01-02"),
		})
		require.NoError(t, err)
	})

	t.Run("sync failure keeps rule write", func(t *testing.T) {
		m := &mock.MockQuerier{
			CreateRecurringIncomeRuleFn: func(ctx context.Context, arg db.CreateRecurringIncomeRuleParams) (db.RecurringIncomeRule, error) {
				return db.RecurringIncomeRule{
					ID:            ruleID,
					UserID:        userID,
					Description:   "Salary",
					RecurringType: "daily",
					StartDate:     today,
					IsActive:      true,
				}, nil
			},
			CreateIncomeInstanceFn: func(ctx context.Context, arg db.CreateIncomeInstanceParams) (int64, error) {
				return 0, errors.New("db error")
			},
		}
		svc := NewService(m)
		result, err := svc.CreateIncomeRule(context.Background(), userID, CreateIncomeRuleRequest{
			Amount:        5000,
			Currency:      "PHP",
			Description:   "Salary",
			RecurringType: "daily",
			StartDate:     today.Time.Format("2006-01-02"),
		})
		require.NoError(t, err, "sync failure must not fail the rule write")
		assert.Equal(t, ruleID, result.ID)
	})
}

func TestRecurringService_UpdateIncomeRuleSyncsTodayInstance(t *testing.T) {
	userID := uuid.New()
	ruleID := uuid.New()
	today := todayPg()

	t.Run("update becomes due today inserts instance", func(t *testing.T) {
		instanceCalled := false
		m := &mock.MockQuerier{
			UpdateRecurringIncomeRuleFn: func(ctx context.Context, arg db.UpdateRecurringIncomeRuleParams) (db.RecurringIncomeRule, error) {
				return db.RecurringIncomeRule{
					ID:            ruleID,
					UserID:        userID,
					Description:   "Salary",
					RecurringType: "daily",
					StartDate:     today,
					IsActive:      true,
				}, nil
			},
			CreateIncomeInstanceFn: func(ctx context.Context, arg db.CreateIncomeInstanceParams) (int64, error) {
				instanceCalled = true
				assert.Equal(t, ruleID, uuid.UUID(arg.SourceRuleID.Bytes))
				return 1, nil
			},
		}
		svc := NewService(m)
		result, err := svc.UpdateIncomeRule(context.Background(), ruleID, userID, UpdateIncomeRuleRequest{
			Amount:        5000,
			Currency:      "PHP",
			Description:   "Salary",
			RecurringType: "daily",
			StartDate:     today.Time.Format("2006-01-02"),
			IsActive:      ptrBool(true),
		})
		require.NoError(t, err)
		assert.Equal(t, ruleID, result.ID)
		assert.True(t, instanceCalled, "update making rule due today must trigger a sync insert")
	})

	t.Run("sync failure keeps rule write", func(t *testing.T) {
		m := &mock.MockQuerier{
			UpdateRecurringIncomeRuleFn: func(ctx context.Context, arg db.UpdateRecurringIncomeRuleParams) (db.RecurringIncomeRule, error) {
				return db.RecurringIncomeRule{
					ID:            ruleID,
					UserID:        userID,
					Description:   "Salary",
					RecurringType: "daily",
					StartDate:     today,
					IsActive:      true,
				}, nil
			},
			CreateIncomeInstanceFn: func(ctx context.Context, arg db.CreateIncomeInstanceParams) (int64, error) {
				return 0, errors.New("db error")
			},
		}
		svc := NewService(m)
		result, err := svc.UpdateIncomeRule(context.Background(), ruleID, userID, UpdateIncomeRuleRequest{
			Amount:        5000,
			Currency:      "PHP",
			Description:   "Salary",
			RecurringType: "daily",
			StartDate:     today.Time.Format("2006-01-02"),
			IsActive:      ptrBool(true),
		})
		require.NoError(t, err, "sync failure must not fail the rule write")
		assert.Equal(t, ruleID, result.ID)
	})
}
