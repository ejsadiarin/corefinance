package budget

import (
	"context"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	db "github.com/ejsadiarin/corefinance/internal/db/sqlc"
	"github.com/ejsadiarin/corefinance/internal/helper"
)

type Service struct {
	queries Querier
	pool    *pgxpool.Pool
}

func NewService(queries Querier, pool *pgxpool.Pool) *Service {
	return &Service{
		queries: queries,
		pool:    pool,
	}
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

	expenseCategories, err := s.queries.ListExpenseCategories(ctx, userID)
	if err != nil {
		return nil, err
	}

	incomeCategories, err := s.queries.ListIncomeCategories(ctx, userID)
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

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		slog.Error("budget.Service.Import: failed to start transaction", "error", err, "user_id", userID)
		return err
	}
	defer tx.Rollback(ctx)

	q := s.queries.WithTx(tx)

	// Track ID mappings: old ID -> new ID
	categoryMap := make(map[uuid.UUID]uuid.UUID)
	tagMap := make(map[uuid.UUID]uuid.UUID)

	// Import expense categories
	for _, cat := range export.ExpenseCategories {
		newCat, err := q.CreateExpenseCategory(ctx, db.CreateExpenseCategoryParams{
			UserID: userID,
			Name:   cat.Name,
			Color:  cat.Color,
			Icon:   cat.Icon,
		})
		if err != nil {
			slog.Error("budget.Service.Import: failed to import expense categories", "error", err, "user_id", userID)
			return err
		}
		categoryMap[cat.ID] = newCat.ID
	}

	// Import income categories
	for _, cat := range export.IncomeCategories {
		newCat, err := q.CreateIncomeCategory(ctx, db.CreateIncomeCategoryParams{
			UserID: userID,
			Name:   cat.Name,
			Color:  cat.Color,
			Icon:   cat.Icon,
		})
		if err != nil {
			slog.Error("budget.Service.Import: failed to import income categories", "error", err, "user_id", userID)
			return err
		}
		categoryMap[cat.ID] = newCat.ID
	}

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

	// Track expense/income ID mappings
	expenseMap := make(map[uuid.UUID]uuid.UUID)
	incomeMap := make(map[uuid.UUID]uuid.UUID)

	// Import expenses
	for _, exp := range export.Expenses {
		newCatID := uuid.Nil
		if exp.CategoryID.Valid {
			if mapped, ok := categoryMap[exp.CategoryID.Bytes]; ok {
				newCatID = mapped
			}
		}
		newRuleID := uuid.Nil
		if exp.SourceRuleID.Valid {
			newRuleID = exp.SourceRuleID.Bytes
		}
		newExp, err := q.CreateExpense(ctx, db.CreateExpenseParams{
			UserID:        userID,
			CategoryID:    helper.ToPgUUIDFromUUID(newCatID),
			Amount:        exp.Amount,
			Currency:      exp.Currency,
			Description:   exp.Description,
			Notes:         exp.Notes,
			ExpenseDate:   exp.ExpenseDate,
			RecurringType: exp.RecurringType,
			Priority:      exp.Priority,
			Status:        exp.Status,
			IsDebt:        exp.IsDebt,
			StartDate:     exp.StartDate,
			EndDate:       exp.EndDate,
			SourceRuleID:  helper.ToPgUUIDFromUUID(newRuleID),
		})
		if err != nil {
			slog.Error("budget.Service.Import: failed to import expenses", "error", err, "user_id", userID)
			return err
		}
		expenseMap[exp.ID] = newExp.ID
	}

	// Import incomes
	for _, inc := range export.Incomes {
		newCatID := uuid.Nil
		if inc.CategoryID.Valid {
			if mapped, ok := categoryMap[inc.CategoryID.Bytes]; ok {
				newCatID = mapped
			}
		}
		newRuleID := uuid.Nil
		if inc.SourceRuleID.Valid {
			newRuleID = inc.SourceRuleID.Bytes
		}
		newInc, err := q.CreateIncome(ctx, db.CreateIncomeParams{
			UserID:        userID,
			CategoryID:    helper.ToPgUUIDFromUUID(newCatID),
			Amount:        inc.Amount,
			Currency:      inc.Currency,
			Description:   inc.Description,
			Notes:         inc.Notes,
			Date:          inc.Date,
			RecurringType: inc.RecurringType,
			Priority:      inc.Priority,
			Status:        inc.Status,
			StartDate:     inc.StartDate,
			EndDate:       inc.EndDate,
			SourceRuleID:  helper.ToPgUUIDFromUUID(newRuleID),
		})
		if err != nil {
			slog.Error("budget.Service.Import: failed to import incomes", "error", err, "user_id", userID)
			return err
		}
		incomeMap[inc.ID] = newInc.ID
	}

	// Import recurring expense rules
	for _, rule := range export.RecurringExpenses {
		newCatID := uuid.Nil
		if rule.CategoryID.Valid {
			if mapped, ok := categoryMap[rule.CategoryID.Bytes]; ok {
				newCatID = mapped
			}
		}
		_, err := q.CreateRecurringExpenseRule(ctx, db.CreateRecurringExpenseRuleParams{
			UserID:        userID,
			Description:   rule.Description,
			Amount:        rule.Amount,
			Currency:      rule.Currency,
			CategoryID:    helper.ToPgUUIDFromUUID(newCatID),
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
			q.AddTagToExpense(ctx, db.AddTagToExpenseParams{
				ExpenseID: newExpID,
				TagID:     newTagID,
			})
		}
	}

	// Import income tags
	for _, it := range export.IncomeTags {
		newIncID, incOk := incomeMap[it.IncomeID]
		newTagID, tagOk := tagMap[it.TagID]
		if incOk && tagOk {
			q.AddTagToIncome(ctx, db.AddTagToIncomeParams{
				IncomeID: newIncID,
				TagID:    newTagID,
			})
		}
	}

	if err := tx.Commit(ctx); err != nil {
		slog.Error("budget.Service.Import: failed to commit transaction", "error", err, "user_id", userID)
		return err
	}

	slog.Info("budget.Service.Import: completed", "user_id", userID)
	return nil
}

// Ensure pgx.Tx is used (for compile check)
var _ pgx.Tx = (pgx.Tx)(nil)
