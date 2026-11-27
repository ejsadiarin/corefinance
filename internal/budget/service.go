package budget

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/shopspring/decimal"

	db "github.com/ejsadiarin/corefinance/internal/db/sqlc"
	"github.com/ejsadiarin/corefinance/internal/helper"
	"github.com/ejsadiarin/corefinance/internal/rates"
)

type Transactioner interface {
	Begin(ctx context.Context) (pgx.Tx, error)
}

type Service struct {
	queries db.Querier
	db      Transactioner
	rates   rates.Provider
}

func NewService(queries db.Querier, db Transactioner) *Service {
	return NewServiceWithRates(queries, db, rates.PHPOnly{})
}

// NewServiceWithRates validates imported currencies against the provider.
func NewServiceWithRates(queries db.Querier, db Transactioner, provider rates.Provider) *Service {
	return &Service{queries: queries, db: db, rates: provider}
}

func (s *Service) checkImportCurrency(ctx context.Context, currency string) error {
	c := rates.Normalize(currency)
	if c == "PHP" {
		return nil
	}
	return rates.Supports(ctx, s.rates, c)
}

// PriorityGroups returns the hardcoded reference data for priority groups.
func (s *Service) PriorityGroups() []PriorityGroup {
	return []PriorityGroup{
		{ID: "need", Name: "Need", Description: "Essential expenses"},
		{ID: "want", Name: "Want", Description: "Non-essential expenses"},
		{ID: "savings", Name: "Savings", Description: "Savings and investments"},
	}
}

// Remaining calculates income minus expenses for the current month.
func (s *Service) Remaining(ctx context.Context, userID uuid.UUID) (*BudgetRemainingResponse, error) {
	slog.Debug("budget.Service.Remaining: called", "user_id", userID)

	now := time.Now()
	startOfMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	endOfMonth := startOfMonth.AddDate(0, 1, -1)
	startStr := startOfMonth.Format("2006-01-02")
	endStr := endOfMonth.Format("2006-01-02")

	totalIncome, err := s.queries.GetTotalIncomesByDateRange(ctx, db.GetTotalIncomesByDateRangeParams{
		UserID: userID,
		Date:   helper.ToPgDate(startStr),
		Date_2: helper.ToPgDate(endStr),
	})
	if err != nil {
		slog.Error("budget.Service.Remaining: failed to get income", "error", err, "user_id", userID)
		return nil, err
	}

	totalExpenses, err := s.queries.GetTotalExpensesByDateRange(ctx, db.GetTotalExpensesByDateRangeParams{
		UserID:        userID,
		ExpenseDate:   helper.ToPgDate(startStr),
		ExpenseDate_2: helper.ToPgDate(endStr),
	})
	if err != nil {
		slog.Error("budget.Service.Remaining: failed to get expenses", "error", err, "user_id", userID)
		return nil, err
	}

	incomeDec := helper.ToDecimal(totalIncome)
	expenseDec := helper.ToDecimal(totalExpenses)
	remaining := incomeDec.Sub(expenseDec)

	slog.Debug("budget.Service.Remaining: success", "user_id", userID, "remaining", remaining)

	return &BudgetRemainingResponse{
		PeriodStart:  startStr,
		PeriodEnd:    endStr,
		TotalIncome:  incomeDec,
		TotalExpense: expenseDec,
		Remaining:    remaining,
	}, nil
}

// importedConversion trusts a file's conversion fields for non-PHP rows and
// falls back to rate 1 for PHP. A non-PHP row without them predates ADR-0003
// and is refused rather than fetched again.
func importedConversion(
	amount decimal.Decimal,
	currency string,
	rate decimal.Decimal,
	amountPHP decimal.Decimal,
	rateDate pgtype.Date,
	recordDate pgtype.Date,
) (rates.Conversion, error) {
	c := rates.Normalize(currency)
	if c == "PHP" {
		return rates.Conversion{
			Currency:  c,
			Rate:      decimal.NewFromInt(1),
			AmountPHP: amount.Round(2),
			RateDate:  recordDate.Time,
		}, nil
	}
	if !rate.IsPositive() || !amountPHP.IsPositive() || !rateDate.Valid {
		return rates.Conversion{}, fmt.Errorf("%w: %s", ErrLegacyCurrency, c)
	}
	return rates.Conversion{
		Currency:  c,
		Rate:      rate,
		AmountPHP: amountPHP,
		RateDate:  rateDate.Time,
	}, nil
}
