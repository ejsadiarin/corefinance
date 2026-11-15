package category

import (
	"context"

	db "github.com/ejsadiarin/corefinance/internal/db/sqlc"
)

type Service struct {
	queries db.Querier
}

func NewService(queries db.Querier) *Service {
	return &Service{queries: queries}
}

// ListExpense returns the system expense categories, the same for every owner.
func (s *Service) ListExpense(ctx context.Context) ([]db.ExpenseCategory, error) {
	return s.queries.ListExpenseCategories(ctx)
}

// ListIncome returns the system income categories, the same for every owner.
func (s *Service) ListIncome(ctx context.Context) ([]db.IncomeCategory, error) {
	return s.queries.ListIncomeCategories(ctx)
}
