package materialize

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	db "github.com/ejsadiarin/corefinance/internal/db/sqlc"
)

// MaterializeExpenseRuleToday inserts today's instance for a single expense
// rule when it is due today. It mirrors the guards of the daily pass
// (active, start/end window, cadence) and reuses the idempotent instance
// insert, so overlapping calls and worker passes are safe no-ops.
// Returns the number of rows inserted (0 or 1).
func MaterializeExpenseRuleToday(ctx context.Context, q db.Querier, rule db.RecurringExpenseRule, today time.Time) (int64, error) {
	if !rule.IsActive || !rule.StartDate.Valid {
		return 0, nil
	}
	if rule.EndDate.Valid && truncateDate(today).After(truncateDate(rule.EndDate.Time)) {
		return 0, nil
	}
	if !IsDue(rule.StartDate.Time, rule.RecurringType, today) {
		return 0, nil
	}
	n, err := q.CreateExpenseInstance(ctx, db.CreateExpenseInstanceParams{
		UserID:       rule.UserID,
		CategoryID:   rule.CategoryID,
		Amount:       rule.Amount,
		Currency:     rule.Currency,
		Description:  rule.Description,
		Notes:        rule.Notes,
		ExpenseDate:  pgDate(today),
		Priority:     rule.Priority,
		IsDebt:       false,
		SourceRuleID: pgtype.UUID{Bytes: rule.ID, Valid: true},
	})
	if err != nil {
		return 0, fmt.Errorf("create expense instance for rule %s: %w", rule.ID, err)
	}
	return n, nil
}

// MaterializeIncomeRuleToday inserts today's instance for a single income
// rule when it is due today. Same idempotency and guard semantics as
// MaterializeExpenseRuleToday. Returns the number of rows inserted (0 or 1).
func MaterializeIncomeRuleToday(ctx context.Context, q db.Querier, rule db.RecurringIncomeRule, today time.Time) (int64, error) {
	if !rule.IsActive || !rule.StartDate.Valid {
		return 0, nil
	}
	if rule.EndDate.Valid && truncateDate(today).After(truncateDate(rule.EndDate.Time)) {
		return 0, nil
	}
	if !IsDue(rule.StartDate.Time, rule.RecurringType, today) {
		return 0, nil
	}
	n, err := q.CreateIncomeInstance(ctx, db.CreateIncomeInstanceParams{
		UserID:       rule.UserID,
		CategoryID:   pgtype.UUID{},
		Amount:       rule.Amount,
		Currency:     rule.Currency,
		Description:  rule.Description,
		Notes:        pgtype.Text{},
		Date:         pgDate(today),
		Priority:     "want",
		SourceRuleID: pgtype.UUID{Bytes: rule.ID, Valid: true},
	})
	if err != nil {
		return 0, fmt.Errorf("create income instance for rule %s: %w", rule.ID, err)
	}
	return n, nil
}
