package recurring

import (
	"context"
	"log/slog"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"

	db "github.com/ejsadiarin/corefinance/internal/db/sqlc"
	"github.com/ejsadiarin/corefinance/internal/helper"
)

type Service struct {
	queries *db.Queries
}

func NewService(pool *pgxpool.Pool) *Service {
	return &Service{
		queries: db.New(pool),
	}
}

// --- Income Rules ---

func (s *Service) CreateIncomeRule(ctx context.Context, userID uuid.UUID, req CreateIncomeRuleRequest) (db.RecurringIncomeRule, error) {
	slog.Debug("recurring.Service.CreateIncomeRule", "user_id", userID)
	return s.queries.CreateRecurringIncomeRule(ctx, db.CreateRecurringIncomeRuleParams{
		UserID:        userID,
		Amount:        decimal.NewFromFloat(req.Amount),
		Currency:      req.Currency,
		Description:   req.Description,
		RecurringType: req.RecurringType,
		StartDate:     helper.ToPgDate(req.StartDate),
		EndDate:       helper.ToPgDatePtr(req.EndDate),
	})
}

func (s *Service) ListIncomeRules(ctx context.Context, userID uuid.UUID) ([]db.RecurringIncomeRule, error) {
	slog.Debug("recurring.Service.ListIncomeRules", "user_id", userID)
	return s.queries.ListRecurringIncomeRules(ctx, userID)
}

func (s *Service) GetIncomeRule(ctx context.Context, id uuid.UUID, userID uuid.UUID) (db.RecurringIncomeRule, error) {
	slog.Debug("recurring.Service.GetIncomeRule", "id", id, "user_id", userID)
	return s.queries.GetRecurringIncomeRule(ctx, db.GetRecurringIncomeRuleParams{
		ID:     id,
		UserID: userID,
	})
}

func (s *Service) UpdateIncomeRule(ctx context.Context, id uuid.UUID, userID uuid.UUID, req UpdateIncomeRuleRequest) (db.RecurringIncomeRule, error) {
	slog.Debug("recurring.Service.UpdateIncomeRule", "id", id, "user_id", userID)
	return s.queries.UpdateRecurringIncomeRule(ctx, db.UpdateRecurringIncomeRuleParams{
		ID:            id,
		UserID:        userID,
		Amount:        decimal.NewFromFloat(req.Amount),
		Currency:      req.Currency,
		Description:   req.Description,
		RecurringType: req.RecurringType,
		StartDate:     helper.ToPgDate(req.StartDate),
		EndDate:       helper.ToPgDatePtr(req.EndDate),
	})
}

func (s *Service) DeleteIncomeRule(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	slog.Debug("recurring.Service.DeleteIncomeRule", "id", id, "user_id", userID)
	return s.queries.DeleteRecurringIncomeRule(ctx, db.DeleteRecurringIncomeRuleParams{
		ID:     id,
		UserID: userID,
	})
}

// --- Expense Rules ---

func (s *Service) CreateExpenseRule(ctx context.Context, userID uuid.UUID, req CreateExpenseRuleRequest) (db.RecurringExpenseRule, error) {
	slog.Debug("recurring.Service.CreateExpenseRule", "user_id", userID)
	return s.queries.CreateRecurringExpenseRule(ctx, db.CreateRecurringExpenseRuleParams{
		UserID:        userID,
		Description:   req.Description,
		Amount:        decimal.NewFromFloat(req.Amount),
		Currency:      req.Currency,
		CategoryID:    helper.ToPgUUID(req.CategoryID),
		Notes:         helper.ToPgText(req.Notes),
		RecurringType: req.RecurringType,
		StartDate:     helper.ToPgDate(req.StartDate),
		EndDate:       helper.ToPgDatePtr(req.EndDate),
		Priority:      req.Priority,
	})
}

func (s *Service) GetExpenseRule(ctx context.Context, id uuid.UUID, userID uuid.UUID) (db.GetRecurringExpenseRuleRow, error) {
	slog.Debug("recurring.Service.GetExpenseRule", "id", id, "user_id", userID)
	return s.queries.GetRecurringExpenseRule(ctx, db.GetRecurringExpenseRuleParams{
		ID:     id,
		UserID: userID,
	})
}

func (s *Service) UpdateExpenseRule(ctx context.Context, id uuid.UUID, userID uuid.UUID, req UpdateExpenseRuleRequest) (db.RecurringExpenseRule, error) {
	slog.Debug("recurring.Service.UpdateExpenseRule", "id", id, "user_id", userID)
	return s.queries.UpdateRecurringExpenseRule(ctx, db.UpdateRecurringExpenseRuleParams{
		ID:            id,
		UserID:        userID,
		Description:   req.Description,
		Amount:        decimal.NewFromFloat(req.Amount),
		Currency:      req.Currency,
		CategoryID:    helper.ToPgUUID(req.CategoryID),
		Notes:         helper.ToPgText(req.Notes),
		RecurringType: req.RecurringType,
		StartDate:     helper.ToPgDate(req.StartDate),
		EndDate:       helper.ToPgDatePtr(req.EndDate),
		Priority:      req.Priority,
	})
}

func (s *Service) DeleteExpenseRule(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	slog.Debug("recurring.Service.DeleteExpenseRule", "id", id, "user_id", userID)
	return s.queries.DeleteRecurringExpenseRule(ctx, db.DeleteRecurringExpenseRuleParams{
		ID:     id,
		UserID: userID,
	})
}

func (s *Service) ListExpenseRules(ctx context.Context, userID uuid.UUID) ([]db.ListRecurringExpenseRulesRow, error) {
	slog.Debug("recurring.Service.ListExpenseRules", "user_id", userID)
	return s.queries.ListRecurringExpenseRules(ctx, userID)
}
