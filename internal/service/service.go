package service

import (
	"context"

	db "github.com/ejsadiarin/corefinance/internal/db/sqlc"
)

// Service backs scope-gated service endpoints: cross-user reads for
// service identities. There is intentionally no user filter here;
// callers are authorized by scope before reaching this layer.
type Service struct {
	queries db.Querier
}

// NewService builds a Service over sqlc queries.
func NewService(queries db.Querier) *Service {
	return &Service{queries: queries}
}

// ActiveRecurringRules lists active expense and income rules across all
// users, ordered by start date. Reminder/notification services consume
// this to find upcoming obligations without user delegation.
func (s *Service) ActiveRecurringRules(ctx context.Context) ([]db.ListActiveExpenseRulesAllUsersRow, []db.ListActiveIncomeRulesAllUsersRow, error) {
	expenses, err := s.queries.ListActiveExpenseRulesAllUsers(ctx)
	if err != nil {
		return nil, nil, err
	}
	incomes, err := s.queries.ListActiveIncomeRulesAllUsers(ctx)
	if err != nil {
		return nil, nil, err
	}
	return expenses, incomes, nil
}
