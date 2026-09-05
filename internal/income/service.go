package income

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

func (s *Service) Create(ctx context.Context, userID uuid.UUID, req CreateRequest) (db.Income, error) {
	slog.Debug("income.Service.Create", "user_id", userID)
	return s.queries.CreateIncome(ctx, db.CreateIncomeParams{
		UserID:        userID,
		CategoryID:    helper.ToPgUUID(req.CategoryID),
		Amount:        decimal.NewFromFloat(req.Amount),
		Currency:      req.Currency,
		Description:   req.Description,
		Notes:         helper.ToPgText(req.Notes),
		Date:          helper.ToPgDate(req.Date),
		RecurringType: req.RecurringType,
		Priority:      req.Priority,
		Status:        req.Status,
		StartDate:     helper.ToPgDatePtr(req.StartDate),
		EndDate:       helper.ToPgDatePtr(req.EndDate),
		SourceRuleID:  helper.ToPgUUID(req.SourceRuleID),
	})
}

func (s *Service) Get(ctx context.Context, id uuid.UUID, userID uuid.UUID) (db.GetIncomeRow, error) {
	slog.Debug("income.Service.Get", "id", id, "user_id", userID)
	return s.queries.GetIncome(ctx, db.GetIncomeParams{
		ID:     id,
		UserID: userID,
	})
}

func (s *Service) List(ctx context.Context, userID uuid.UUID, params ListParams) ([]db.ListIncomesRow, error) {
	slog.Debug("income.Service.List", "user_id", userID)
	page, pageSize := int32(params.Page), int32(params.PageSize)
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 50
	}
	return s.queries.ListIncomes(ctx, db.ListIncomesParams{
		UserID:        userID,
		StartDate:     helper.ToPgDatePtr(params.StartDate),
		EndDate:       helper.ToPgDatePtr(params.EndDate),
		CategoryID:    helper.ToUUID(params.CategoryID),
		Status:        helper.StrVal(params.Status),
		RecurringType: helper.StrVal(params.RecurringType),
		PageLimit:     pageSize,
		PageOffset:    (page - 1) * pageSize,
	})
}

func (s *Service) Update(ctx context.Context, id uuid.UUID, userID uuid.UUID, req UpdateRequest) (db.Income, error) {
	slog.Debug("income.Service.Update", "id", id, "user_id", userID)
	return s.queries.UpdateIncome(ctx, db.UpdateIncomeParams{
		ID:            id,
		UserID:        userID,
		CategoryID:    helper.ToPgUUID(req.CategoryID),
		Amount:        decimal.NewFromFloat(req.Amount),
		Currency:      req.Currency,
		Description:   req.Description,
		Notes:         helper.ToPgText(req.Notes),
		Date:          helper.ToPgDate(req.Date),
		RecurringType: req.RecurringType,
		Priority:      req.Priority,
		Status:        req.Status,
		StartDate:     helper.ToPgDatePtr(req.StartDate),
		EndDate:       helper.ToPgDatePtr(req.EndDate),
		SourceRuleID:  helper.ToPgUUID(req.SourceRuleID),
	})
}

func (s *Service) Delete(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	slog.Debug("income.Service.Delete", "id", id, "user_id", userID)
	return s.queries.DeleteIncome(ctx, db.DeleteIncomeParams{
		ID:     id,
		UserID: userID,
	})
}

func (s *Service) Skip(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	slog.Debug("income.Service.Skip", "id", id, "user_id", userID)
	return s.queries.SkipIncome(ctx, db.SkipIncomeParams{
		ID:     id,
		UserID: userID,
	})
}

func (s *Service) CheckSkipped(ctx context.Context, userID uuid.UUID, startDate, endDate string) (bool, error) {
	slog.Debug("income.Service.CheckSkipped", "user_id", userID)
	return s.queries.CheckSkippedIncome(ctx, db.CheckSkippedIncomeParams{
		UserID: userID,
		Date:   helper.ToPgDate(startDate),
		Date_2: helper.ToPgDate(endDate),
	})
}

func (s *Service) Occurrences(ctx context.Context, userID uuid.UUID, startDate, endDate *string) ([]db.Income, error) {
	slog.Debug("income.Service.Occurrences", "user_id", userID)
	return s.queries.GetIncomeOccurrences(ctx, db.GetIncomeOccurrencesParams{
		UserID:    userID,
		StartDate: helper.ToPgDatePtr(startDate),
		EndDate:   helper.ToPgDatePtr(endDate),
	})
}

func (s *Service) TotalByDateRange(ctx context.Context, userID uuid.UUID, startDate, endDate string) (decimal.Decimal, error) {
	slog.Debug("income.Service.TotalByDateRange", "user_id", userID)
	return s.queries.GetTotalIncomesByDateRange(ctx, db.GetTotalIncomesByDateRangeParams{
		UserID: userID,
		Date:   helper.ToPgDate(startDate),
		Date_2: helper.ToPgDate(endDate),
	})
}
