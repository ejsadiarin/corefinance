package recurring

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"

	db "github.com/ejsadiarin/corefinance/internal/db/sqlc"
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
	return s.queries.CreateRecurringIncomeRule(ctx, db.CreateRecurringIncomeRuleParams{
		UserID:        userID,
		Amount:        decimal.NewFromFloat(req.Amount),
		Currency:      req.Currency,
		Description:   req.Description,
		RecurringType: req.RecurringType,
		StartDate:     toPgDate(req.StartDate),
		EndDate:       toPgDatePtr(req.EndDate),
	})
}

func (s *Service) ListIncomeRules(ctx context.Context, userID uuid.UUID) ([]db.RecurringIncomeRule, error) {
	return s.queries.ListRecurringIncomeRules(ctx, userID)
}

func (s *Service) GetIncomeRule(ctx context.Context, id uuid.UUID, userID uuid.UUID) (db.RecurringIncomeRule, error) {
	return s.queries.GetRecurringIncomeRule(ctx, db.GetRecurringIncomeRuleParams{
		ID:     id,
		UserID: userID,
	})
}

func (s *Service) UpdateIncomeRule(ctx context.Context, id uuid.UUID, userID uuid.UUID, req UpdateIncomeRuleRequest) (db.RecurringIncomeRule, error) {
	return s.queries.UpdateRecurringIncomeRule(ctx, db.UpdateRecurringIncomeRuleParams{
		ID:            id,
		UserID:        userID,
		Amount:        decimal.NewFromFloat(req.Amount),
		Currency:      req.Currency,
		Description:   req.Description,
		RecurringType: req.RecurringType,
		StartDate:     toPgDate(req.StartDate),
		EndDate:       toPgDatePtr(req.EndDate),
	})
}

func (s *Service) DeleteIncomeRule(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	return s.queries.DeleteRecurringIncomeRule(ctx, db.DeleteRecurringIncomeRuleParams{
		ID:     id,
		UserID: userID,
	})
}

// --- Expense Rules ---

func (s *Service) ListExpenseRules(ctx context.Context, userID uuid.UUID) ([]db.ListRecurringExpenseRulesRow, error) {
	return s.queries.ListRecurringExpenseRules(ctx, userID)
}

// --- Helpers ---

func toPgDate(date string) pgtype.Date {
	if date == "" {
		return pgtype.Date{Valid: false}
	}
	t, err := time.Parse("2006-01-02", date)
	if err != nil {
		return pgtype.Date{Valid: false}
	}
	return pgtype.Date{Time: t, Valid: true}
}

func toPgDatePtr(date *string) pgtype.Date {
	if date == nil {
		return pgtype.Date{Valid: false}
	}
	return toPgDate(*date)
}
