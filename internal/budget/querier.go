package budget

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/shopspring/decimal"

	db "github.com/ejsadiarin/corefinance/internal/db/sqlc"
)

type Querier interface {
	// Budget remaining
	GetTotalIncomesByDateRange(ctx context.Context, arg db.GetTotalIncomesByDateRangeParams) (decimal.Decimal, error)
	GetTotalExpensesByDateRange(ctx context.Context, arg db.GetTotalExpensesByDateRangeParams) (decimal.Decimal, error)

	// Export — list all data for user
	ListExpenseCategories(ctx context.Context, userID uuid.UUID) ([]db.ExpenseCategory, error)
	ListIncomeCategories(ctx context.Context, userID uuid.UUID) ([]db.IncomeCategory, error)
	ListTags(ctx context.Context, userID uuid.UUID) ([]db.Tag, error)
	ListAllExpensesByUser(ctx context.Context, userID uuid.UUID) ([]db.Expense, error)
	ListAllIncomesByUser(ctx context.Context, userID uuid.UUID) ([]db.Income, error)
	ListRecurringExpenseRules(ctx context.Context, userID uuid.UUID) ([]db.ListRecurringExpenseRulesRow, error)
	ListRecurringIncomeRules(ctx context.Context, userID uuid.UUID) ([]db.RecurringIncomeRule, error)
	ListAllExpenseTagsByUser(ctx context.Context, userID uuid.UUID) ([]db.ExpenseTag, error)
	ListAllIncomeTagsByUser(ctx context.Context, userID uuid.UUID) ([]db.IncomeTag, error)

	// Import — create entities (used within transaction)
	CreateExpenseCategory(ctx context.Context, arg db.CreateExpenseCategoryParams) (db.ExpenseCategory, error)
	CreateIncomeCategory(ctx context.Context, arg db.CreateIncomeCategoryParams) (db.IncomeCategory, error)
	CreateTag(ctx context.Context, arg db.CreateTagParams) (db.Tag, error)
	CreateExpense(ctx context.Context, arg db.CreateExpenseParams) (db.Expense, error)
	CreateIncome(ctx context.Context, arg db.CreateIncomeParams) (db.Income, error)
	CreateRecurringExpenseRule(ctx context.Context, arg db.CreateRecurringExpenseRuleParams) (db.RecurringExpenseRule, error)
	CreateRecurringIncomeRule(ctx context.Context, arg db.CreateRecurringIncomeRuleParams) (db.RecurringIncomeRule, error)
	AddTagToExpense(ctx context.Context, arg db.AddTagToExpenseParams) error
	AddTagToIncome(ctx context.Context, arg db.AddTagToIncomeParams) error

	// Transaction support
	WithTx(tx pgx.Tx) *db.Queries
}
