package budget

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/shopspring/decimal"

	"github.com/ejsadiarin/corefinance/internal/category"
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

// Export fetches all budget data for the user.
func (s *Service) Export(ctx context.Context, userID uuid.UUID) (*BudgetExport, error) {
	slog.Debug("budget.Service.Export: called", "user_id", userID)

	expenseCategories, err := s.queries.ListExpenseCategories(ctx)
	if err != nil {
		return nil, err
	}

	incomeCategories, err := s.queries.ListIncomeCategories(ctx)
	if err != nil {
		return nil, err
	}

	tags, err := s.queries.ListTags(ctx, userID)
	if err != nil {
		return nil, err
	}

	expenses, err := s.queries.ListAllExpensesByUser(ctx, userID)
	if err != nil {
		return nil, err
	}

	incomes, err := s.queries.ListAllIncomesByUser(ctx, userID)
	if err != nil {
		return nil, err
	}

	recurringExpenses, err := s.queries.ListRecurringExpenseRules(ctx, userID)
	if err != nil {
		return nil, err
	}

	recurringIncomes, err := s.queries.ListRecurringIncomeRules(ctx, userID)
	if err != nil {
		return nil, err
	}

	expenseTags, err := s.queries.ListAllExpenseTagsByUser(ctx, userID)
	if err != nil {
		return nil, err
	}

	incomeTags, err := s.queries.ListAllIncomeTagsByUser(ctx, userID)
	if err != nil {
		return nil, err
	}

	slog.Info("budget.Service.Export: completed", "user_id", userID)

	return &BudgetExport{
		ExpenseCategories: expenseCategories,
		IncomeCategories:  incomeCategories,
		Tags:              tags,
		Expenses:          expenses,
		Incomes:           incomes,
		RecurringExpenses: recurringExpenses,
		RecurringIncomes:  recurringIncomes,
		ExpenseTags:       expenseTags,
		IncomeTags:        incomeTags,
	}, nil
}

// Import imports budget data within a transaction, remapping IDs.
func (s *Service) Import(ctx context.Context, userID uuid.UUID, export *BudgetExport) error {
	slog.Debug("budget.Service.Import: called", "user_id", userID)

	tx, err := s.db.Begin(ctx)
	if err != nil {
		slog.Error("budget.Service.Import: failed to start transaction", "error", err, "user_id", userID)
		return err
	}
	defer tx.Rollback(ctx)

	q := db.New(tx)

	// Categories are system-defined (ADR-0002): the file's categories map by
	// name, and one with no match becomes `other` plus a tag of its name.
	var expenseCats, incomeCats []namedCategory
	for _, c := range export.ExpenseCategories {
		expenseCats = append(expenseCats, namedCategory{ID: c.ID, Name: c.Name})
	}
	for _, c := range export.IncomeCategories {
		incomeCats = append(incomeCats, namedCategory{ID: c.ID, Name: c.Name})
	}
	expenseCategory := newCategoryMapper(category.Expenses, category.ExpenseOther, expenseCats)
	incomeCategory := newCategoryMapper(category.Incomes, category.IncomeOther, incomeCats)
	tagMap := make(map[uuid.UUID]uuid.UUID)

	// Import tags
	for _, tag := range export.Tags {
		newTag, err := q.CreateTag(ctx, db.CreateTagParams{
			UserID: userID,
			Name:   tag.Name,
			Color:  tag.Color,
		})
		if err != nil {
			slog.Error("budget.Service.Import: failed to import tags", "error", err, "user_id", userID)
			return err
		}
		tagMap[tag.ID] = newTag.ID
	}

	// The owner's tags by name, for the tags that carry an unmatched
	// category's name: reuse one of that name, or create it.
	ownTags, err := q.ListTags(ctx, userID)
	if err != nil {
		return err
	}
	tagByName := make(map[string]uuid.UUID, len(ownTags))
	for _, t := range ownTags {
		key := strings.ToLower(strings.TrimSpace(t.Name))
		if _, seen := tagByName[key]; !seen {
			tagByName[key] = t.ID
		}
	}
	categoryTag := func(name string) (uuid.UUID, error) {
		key := strings.ToLower(name)
		if id, ok := tagByName[key]; ok {
			return id, nil
		}
		t, err := q.CreateTag(ctx, db.CreateTagParams{UserID: userID, Name: name})
		if err != nil {
			return uuid.Nil, err
		}
		tagByName[key] = t.ID
		return t.ID, nil
	}

	// Track expense/income ID mappings
	expenseMap := make(map[uuid.UUID]uuid.UUID)
	incomeMap := make(map[uuid.UUID]uuid.UUID)

	// Import expenses
	for _, exp := range export.Expenses {
		if err := s.checkImportCurrency(ctx, exp.Currency); err != nil {
			return err
		}
		cat := expenseCategory.target(exp.CategoryID)
		newRuleID := uuid.Nil
		if exp.SourceRuleID.Valid {
			newRuleID = exp.SourceRuleID.Bytes
		}
		conv, err := importedConversion(
			exp.Amount, exp.Currency, exp.ExchangeRate, exp.AmountPhp, exp.RateDate, exp.ExpenseDate,
		)
		if err != nil {
			return err
		}
		newExp, err := q.CreateExpense(ctx, db.CreateExpenseParams{
			UserID:       userID,
			CategoryID:   helper.ToPgUUIDFromUUID(cat.ID),
			Amount:       exp.Amount,
			Currency:     conv.Currency,
			ExchangeRate: conv.Rate,
			AmountPhp:    conv.AmountPHP,
			RateDate:     pgtype.Date{Time: conv.RateDate, Valid: true},
			Description:  exp.Description,
			Notes:        exp.Notes,
			ExpenseDate:  exp.ExpenseDate,
			Priority:     exp.Priority,
			Status:       exp.Status,
			IsDebt:       exp.IsDebt,
			SourceRuleID: helper.ToPgUUIDFromUUID(newRuleID),
			OccurredAt:   exp.OccurredAt,
		})
		if err != nil {
			slog.Error("budget.Service.Import: failed to import expenses", "error", err, "user_id", userID)
			return err
		}
		expenseMap[exp.ID] = newExp.ID
		if cat.Tag != "" {
			tagID, err := categoryTag(cat.Tag)
			if err != nil {
				return err
			}
			if err := q.AddTagToExpense(ctx, db.AddTagToExpenseParams{ExpenseID: newExp.ID, TagID: tagID}); err != nil {
				return err
			}
		}
	}

	// Import incomes
	for _, inc := range export.Incomes {
		if err := s.checkImportCurrency(ctx, inc.Currency); err != nil {
			return err
		}
		cat := incomeCategory.target(inc.CategoryID)
		newRuleID := uuid.Nil
		if inc.SourceRuleID.Valid {
			newRuleID = inc.SourceRuleID.Bytes
		}
		conv, err := importedConversion(
			inc.Amount, inc.Currency, inc.ExchangeRate, inc.AmountPhp, inc.RateDate, inc.Date,
		)
		if err != nil {
			return err
		}
		newInc, err := q.CreateIncome(ctx, db.CreateIncomeParams{
			UserID:       userID,
			CategoryID:   helper.ToPgUUIDFromUUID(cat.ID),
			Amount:       inc.Amount,
			Currency:     conv.Currency,
			ExchangeRate: conv.Rate,
			AmountPhp:    conv.AmountPHP,
			RateDate:     pgtype.Date{Time: conv.RateDate, Valid: true},
			Description:  inc.Description,
			Notes:        inc.Notes,
			Date:         inc.Date,
			Priority:     inc.Priority,
			Status:       inc.Status,
			SourceRuleID: helper.ToPgUUIDFromUUID(newRuleID),
			OccurredAt:   inc.OccurredAt,
		})
		if err != nil {
			slog.Error("budget.Service.Import: failed to import incomes", "error", err, "user_id", userID)
			return err
		}
		incomeMap[inc.ID] = newInc.ID
		if cat.Tag != "" {
			tagID, err := categoryTag(cat.Tag)
			if err != nil {
				return err
			}
			if err := q.AddTagToIncome(ctx, db.AddTagToIncomeParams{IncomeID: newInc.ID, TagID: tagID}); err != nil {
				return err
			}
		}
	}

	// Import recurring expense rules
	for _, rule := range export.RecurringExpenses {
		_, err := q.CreateRecurringExpenseRule(ctx, db.CreateRecurringExpenseRuleParams{
			UserID:        userID,
			Description:   rule.Description,
			Amount:        rule.Amount,
			Currency:      rule.Currency,
			CategoryID:    helper.ToPgUUIDFromUUID(expenseCategory.target(rule.CategoryID).ID),
			Notes:         rule.Notes,
			RecurringType: rule.RecurringType,
			StartDate:     rule.StartDate,
			EndDate:       rule.EndDate,
			Priority:      rule.Priority,
		})
		if err != nil {
			slog.Error("budget.Service.Import: failed to import recurring expense rules", "error", err, "user_id", userID)
			return err
		}
	}

	// Import recurring income rules
	for _, rule := range export.RecurringIncomes {
		_, err := q.CreateRecurringIncomeRule(ctx, db.CreateRecurringIncomeRuleParams{
			UserID:        userID,
			Amount:        rule.Amount,
			Currency:      rule.Currency,
			Description:   rule.Description,
			RecurringType: rule.RecurringType,
			StartDate:     rule.StartDate,
			EndDate:       rule.EndDate,
			CategoryID:    helper.ToPgUUIDFromUUID(incomeCategory.target(rule.CategoryID).ID),
		})
		if err != nil {
			slog.Error("budget.Service.Import: failed to import recurring income rules", "error", err, "user_id", userID)
			return err
		}
	}

	// Import expense tags
	for _, et := range export.ExpenseTags {
		newExpID, expOk := expenseMap[et.ExpenseID]
		newTagID, tagOk := tagMap[et.TagID]
		if expOk && tagOk {
			if err := q.AddTagToExpense(ctx, db.AddTagToExpenseParams{
				ExpenseID: newExpID,
				TagID:     newTagID,
			}); err != nil {
				slog.Error("budget.Service.Import: failed to import expense tags", "error", err, "user_id", userID)
				return err
			}
		}
	}

	// Import income tags
	for _, it := range export.IncomeTags {
		newIncID, incOk := incomeMap[it.IncomeID]
		newTagID, tagOk := tagMap[it.TagID]
		if incOk && tagOk {
			if err := q.AddTagToIncome(ctx, db.AddTagToIncomeParams{
				IncomeID: newIncID,
				TagID:    newTagID,
			}); err != nil {
				slog.Error("budget.Service.Import: failed to import income tags", "error", err, "user_id", userID)
				return err
			}
		}
	}

	if err := tx.Commit(ctx); err != nil {
		slog.Error("budget.Service.Import: failed to commit transaction", "error", err, "user_id", userID)
		return err
	}

	slog.Info("budget.Service.Import: completed", "user_id", userID)
	return nil
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
