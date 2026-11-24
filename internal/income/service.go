package income

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/shopspring/decimal"

	db "github.com/ejsadiarin/corefinance/internal/db/sqlc"
	"github.com/ejsadiarin/corefinance/internal/helper"
	"github.com/ejsadiarin/corefinance/internal/occurred"
	"github.com/ejsadiarin/corefinance/internal/rates"
)

type Service struct {
	queries db.Querier
	rates   rates.Provider
}

func NewService(queries db.Querier) *Service {
	return NewServiceWithRates(queries, rates.PHPOnly{})
}

// NewServiceWithRates converts every record through provider (ADR-0003).
func NewServiceWithRates(queries db.Querier, provider rates.Provider) *Service {
	return &Service{queries: queries, rates: provider}
}

func (s *Service) conversion(ctx context.Context, currency, date string, amount decimal.Decimal) (rates.Conversion, error) {
	d, err := time.Parse("2006-01-02", date)
	if err != nil {
		return rates.Conversion{}, fmt.Errorf("income date: %w", err)
	}
	return rates.Convert(ctx, s.rates, currency, d, amount)
}

func (s *Service) Create(ctx context.Context, userID uuid.UUID, req CreateRequest) (db.Income, error) {
	slog.Debug("income.Service.Create", "user_id", userID)
	amount := decimal.NewFromFloat(req.Amount)
	conv, err := s.conversion(ctx, req.Currency, req.Date, amount)
	if err != nil {
		return db.Income{}, err
	}
	at, err := occurred.For(ctx, s.queries, userID, req.Date, req.Time)
	if err != nil {
		return db.Income{}, err
	}
	return s.queries.CreateIncome(ctx, db.CreateIncomeParams{
		UserID:       userID,
		CategoryID:   helper.ToPgUUID(req.CategoryID),
		Amount:       amount,
		Currency:     conv.Currency,
		ExchangeRate: conv.Rate,
		AmountPhp:    conv.AmountPHP,
		RateDate:     pgtype.Date{Time: conv.RateDate, Valid: true},
		Description:  req.Description,
		Notes:        helper.ToPgText(req.Notes),
		Date:         helper.ToPgDate(req.Date),
		Priority:     req.Priority,
		Status:       req.Status,
		SourceRuleID: helper.ToPgUUID(req.SourceRuleID),
		OccurredAt:   at,
	})
}

func (s *Service) Get(ctx context.Context, id uuid.UUID, userID uuid.UUID) (db.GetIncomeRow, error) {
	slog.Debug("income.Service.Get", "id", id, "user_id", userID)
	return s.queries.GetIncome(ctx, db.GetIncomeParams{
		ID:     id,
		UserID: userID,
	})
}

// ListByIDs returns the user's incomes among ids; the rest are left out.
func (s *Service) ListByIDs(ctx context.Context, userID uuid.UUID, ids []uuid.UUID) ([]db.ListIncomesByIDsRow, error) {
	slog.Debug("income.Service.ListByIDs", "user_id", userID, "count", len(ids))
	return s.queries.ListIncomesByIDs(ctx, db.ListIncomesByIDsParams{UserID: userID, Ids: ids})
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
		UserID:     userID,
		StartDate:  helper.ToPgDatePtr(params.StartDate),
		EndDate:    helper.ToPgDatePtr(params.EndDate),
		CategoryID: helper.ToUUID(params.CategoryID),
		Status:     helper.StrVal(params.Status),
		PageLimit:  pageSize,
		PageOffset: (page - 1) * pageSize,
	})
}

func (s *Service) Update(ctx context.Context, id uuid.UUID, userID uuid.UUID, req UpdateRequest) (db.Income, error) {
	slog.Debug("income.Service.Update", "id", id, "user_id", userID)
	existing, err := s.queries.GetIncome(ctx, db.GetIncomeParams{ID: id, UserID: userID})
	if err != nil {
		return db.Income{}, err
	}

	amount := decimal.NewFromFloat(req.Amount)
	date, err := time.Parse("2006-01-02", req.Date)
	if err != nil {
		return db.Income{}, fmt.Errorf("income date: %w", err)
	}
	currency := rates.Normalize(req.Currency)
	if req.Currency == "" {
		currency = rates.Normalize(existing.Currency)
	}
	conv, err := rates.ConvertForUpdate(ctx, s.rates, rates.Existing{
		Currency: existing.Currency,
		Date:     existing.Date.Time,
		Rate:     existing.ExchangeRate,
		RateDate: existing.RateDate.Time,
	}, currency, date, amount)
	if err != nil {
		return db.Income{}, err
	}
	at, err := occurred.For(ctx, s.queries, userID, req.Date, req.Time)
	if err != nil {
		return db.Income{}, err
	}
	return s.queries.UpdateIncome(ctx, db.UpdateIncomeParams{
		ID:           id,
		UserID:       userID,
		CategoryID:   helper.ToPgUUID(req.CategoryID),
		Amount:       amount,
		Currency:     conv.Currency,
		ExchangeRate: conv.Rate,
		AmountPhp:    conv.AmountPHP,
		RateDate:     pgtype.Date{Time: conv.RateDate, Valid: true},
		Description:  req.Description,
		Notes:        helper.ToPgText(req.Notes),
		Date:         helper.ToPgDate(req.Date),
		Priority:     req.Priority,
		Status:       req.Status,
		SourceRuleID: helper.ToPgUUID(req.SourceRuleID),
		OccurredAt:   at,
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
