package expense

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
		return rates.Conversion{}, fmt.Errorf("expense date: %w", err)
	}
	return rates.Convert(ctx, s.rates, currency, d, amount)
}

func (s *Service) Create(ctx context.Context, userID uuid.UUID, req CreateRequest) (db.Expense, error) {
	slog.Debug("expense.Service.Create", "user_id", userID)
	amount := decimal.NewFromFloat(req.Amount)
	conv, err := s.conversion(ctx, req.Currency, req.ExpenseDate, amount)
	if err != nil {
		return db.Expense{}, err
	}
	at, err := occurred.For(ctx, s.queries, userID, req.ExpenseDate, req.Time)
	if err != nil {
		return db.Expense{}, err
	}
	return s.queries.CreateExpense(ctx, db.CreateExpenseParams{
		UserID:       userID,
		CategoryID:   helper.ToPgUUID(req.CategoryID),
		Amount:       amount,
		Currency:     conv.Currency,
		ExchangeRate: conv.Rate,
		AmountPhp:    conv.AmountPHP,
		RateDate:     pgtype.Date{Time: conv.RateDate, Valid: true},
		Description:  req.Description,
		Notes:        helper.ToPgText(req.Notes),
		ExpenseDate:  helper.ToPgDate(req.ExpenseDate),
		Priority:     req.Priority,
		Status:       req.Status,
		IsDebt:       req.IsDebt,
		SourceRuleID: helper.ToPgUUID(req.SourceRuleID),
		OccurredAt:   at,
	})
}

func (s *Service) Get(ctx context.Context, id uuid.UUID, userID uuid.UUID) (db.GetExpenseRow, error) {
	slog.Debug("expense.Service.Get", "id", id, "user_id", userID)
	return s.queries.GetExpense(ctx, db.GetExpenseParams{
		ID:     id,
		UserID: userID,
	})
}

func (s *Service) List(ctx context.Context, userID uuid.UUID, params ListParams) ([]db.ListExpensesRow, error) {
	slog.Debug("expense.Service.List", "user_id", userID)
	page, pageSize := int32(params.Page), int32(params.PageSize)
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 50
	}
	return s.queries.ListExpenses(ctx, db.ListExpensesParams{
		UserID:     userID,
		StartDate:  helper.ToPgDatePtr(params.StartDate),
		EndDate:    helper.ToPgDatePtr(params.EndDate),
		CategoryID: helper.ToUUID(params.CategoryID),
		Priority:   helper.StrVal(params.Priority),
		Status:     helper.StrVal(params.Status),
		PageLimit:  pageSize,
		PageOffset: (page - 1) * pageSize,
	})
}

// ListByIDs returns the user's expenses among ids; the rest are left out.
func (s *Service) ListByIDs(ctx context.Context, userID uuid.UUID, ids []uuid.UUID) ([]db.ListExpensesByIDsRow, error) {
	slog.Debug("expense.Service.ListByIDs", "user_id", userID, "count", len(ids))
	return s.queries.ListExpensesByIDs(ctx, db.ListExpensesByIDsParams{UserID: userID, Ids: ids})
}

func (s *Service) Search(ctx context.Context, userID uuid.UUID, params SearchParams) ([]db.SearchExpensesRow, error) {
	slog.Debug("expense.Service.Search", "user_id", userID, "query", params.Query)
	page, pageSize := int32(params.Page), int32(params.PageSize)
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 50
	}
	return s.queries.SearchExpenses(ctx, db.SearchExpensesParams{
		UserID:     userID,
		Query:      helper.ToPgText(&params.Query),
		PageLimit:  pageSize,
		PageOffset: (page - 1) * pageSize,
	})
}

func (s *Service) Update(ctx context.Context, id uuid.UUID, userID uuid.UUID, req UpdateRequest) (db.Expense, error) {
	slog.Debug("expense.Service.Update", "id", id, "user_id", userID)
	existing, err := s.queries.GetExpense(ctx, db.GetExpenseParams{ID: id, UserID: userID})
	if err != nil {
		return db.Expense{}, err
	}

	amount := decimal.NewFromFloat(req.Amount)
	date, err := time.Parse("2006-01-02", req.ExpenseDate)
	if err != nil {
		return db.Expense{}, fmt.Errorf("expense date: %w", err)
	}
	currency := rates.Normalize(req.Currency)
	if req.Currency == "" {
		currency = rates.Normalize(existing.Currency)
	}
	conv, err := rates.ConvertForUpdate(ctx, s.rates, rates.Existing{
		Currency: existing.Currency,
		Date:     existing.ExpenseDate.Time,
		Rate:     existing.ExchangeRate,
		RateDate: existing.RateDate.Time,
	}, currency, date, amount)
	if err != nil {
		return db.Expense{}, err
	}
	at, err := occurred.For(ctx, s.queries, userID, req.ExpenseDate, req.Time)
	if err != nil {
		return db.Expense{}, err
	}
	return s.queries.UpdateExpense(ctx, db.UpdateExpenseParams{
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
		ExpenseDate:  helper.ToPgDate(req.ExpenseDate),
		Priority:     req.Priority,
		Status:       req.Status,
		IsDebt:       req.IsDebt,
		SourceRuleID: helper.ToPgUUID(req.SourceRuleID),
		OccurredAt:   at,
	})
}

func (s *Service) Delete(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	slog.Debug("expense.Service.Delete", "id", id, "user_id", userID)
	return s.queries.DeleteExpense(ctx, db.DeleteExpenseParams{
		ID:     id,
		UserID: userID,
	})
}

func (s *Service) Skip(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	slog.Debug("expense.Service.Skip", "id", id, "user_id", userID)
	return s.queries.SkipExpense(ctx, db.SkipExpenseParams{
		ID:     id,
		UserID: userID,
	})
}

func (s *Service) CheckSkipped(ctx context.Context, userID uuid.UUID, startDate, endDate string) (bool, error) {
	slog.Debug("expense.Service.CheckSkipped", "user_id", userID)
	return s.queries.CheckSkippedExpense(ctx, db.CheckSkippedExpenseParams{
		UserID:        userID,
		ExpenseDate:   helper.ToPgDate(startDate),
		ExpenseDate_2: helper.ToPgDate(endDate),
	})
}

func (s *Service) StatsByCategory(ctx context.Context, userID uuid.UUID, startDate, endDate *string) ([]db.GetExpenseStatsByCategoryRow, error) {
	slog.Debug("expense.Service.StatsByCategory", "user_id", userID)
	return s.queries.GetExpenseStatsByCategory(ctx, db.GetExpenseStatsByCategoryParams{
		UserID:    userID,
		StartDate: helper.ToPgDatePtr(startDate),
		EndDate:   helper.ToPgDatePtr(endDate),
	})
}

func (s *Service) TotalByDateRange(ctx context.Context, userID uuid.UUID, startDate, endDate string) (decimal.Decimal, error) {
	slog.Debug("expense.Service.TotalByDateRange", "user_id", userID)
	return s.queries.GetTotalExpensesByDateRange(ctx, db.GetTotalExpensesByDateRangeParams{
		UserID:        userID,
		ExpenseDate:   helper.ToPgDate(startDate),
		ExpenseDate_2: helper.ToPgDate(endDate),
	})
}
