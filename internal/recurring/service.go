package recurring

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"

	db "github.com/ejsadiarin/corefinance/internal/db/sqlc"
	"github.com/ejsadiarin/corefinance/internal/helper"
	"github.com/ejsadiarin/corefinance/internal/materialize"
)

type Service struct {
	queries db.Querier
	// pool, when set, runs a rule update and what it does to the rule's
	// entries in one transaction (#5). Without it (unit tests on a mock) the
	// same steps run on queries directly.
	pool *pgxpool.Pool
}

func NewService(queries db.Querier) *Service {
	return &Service{queries: queries}
}

// NewServiceWithPool is NewService with transactions for rule updates.
func NewServiceWithPool(queries db.Querier, pool *pgxpool.Pool) *Service {
	return &Service{queries: queries, pool: pool}
}

// inTx runs fn in one transaction when the service has a pool.
func (s *Service) inTx(ctx context.Context, fn func(q db.Querier) error) error {
	if s.pool == nil {
		return fn(s.queries)
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin rule update: %w", err)
	}
	defer tx.Rollback(ctx)
	if err := fn(db.New(tx)); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// syncExpenseInstance materializes today's instance for a single expense rule
// right after a rule write. Fail-open by design: errors are logged and the
// rule write stays authoritative — the next worker pass converges.
func (s *Service) syncExpenseInstance(ctx context.Context, rule db.RecurringExpenseRule) {
	n, err := materialize.MaterializeExpenseRuleToday(ctx, s.queries, rule, time.Now())
	if err != nil {
		slog.Error("recurring.Service sync expense instance failed", "rule_id", rule.ID, "error", err)
		return
	}
	if n > 0 {
		slog.Info("recurring.Service synced today's expense instance", "rule_id", rule.ID)
	}
}

// syncIncomeInstance is the income-rule counterpart of syncExpenseInstance.
func (s *Service) syncIncomeInstance(ctx context.Context, rule db.RecurringIncomeRule) {
	n, err := materialize.MaterializeIncomeRuleToday(ctx, s.queries, rule, time.Now())
	if err != nil {
		slog.Error("recurring.Service sync income instance failed", "rule_id", rule.ID, "error", err)
		return
	}
	if n > 0 {
		slog.Info("recurring.Service synced today's income instance", "rule_id", rule.ID)
	}
}

// --- Income Rules ---

func (s *Service) CreateIncomeRule(ctx context.Context, userID uuid.UUID, req CreateIncomeRuleRequest) (db.RecurringIncomeRule, error) {
	slog.Debug("recurring.Service.CreateIncomeRule", "user_id", userID)
	rule, err := s.queries.CreateRecurringIncomeRule(ctx, db.CreateRecurringIncomeRuleParams{
		UserID:        userID,
		Amount:        decimal.NewFromFloat(req.Amount),
		Currency:      req.Currency,
		Description:   req.Description,
		RecurringType: req.RecurringType,
		StartDate:     helper.ToPgDate(req.StartDate),
		EndDate:       helper.ToPgDatePtr(req.EndDate),
		CategoryID:    helper.ToPgUUID(req.CategoryID),
		Notes:         helper.ToPgText(req.Notes),
		Priority:      optionalPriority(req.Priority),
	})
	if err != nil {
		return db.RecurringIncomeRule{}, err
	}
	s.syncIncomeInstance(ctx, rule)
	return rule, nil
}

func (s *Service) ListIncomeRules(ctx context.Context, userID uuid.UUID) ([]db.RecurringIncomeRule, error) {
	slog.Debug("recurring.Service.ListIncomeRules", "user_id", userID)
	return s.queries.ListRecurringIncomeRules(ctx, userID)
}

// ListIncomeRulesWithPaused lists every income rule, paused ones too.
func (s *Service) ListIncomeRulesWithPaused(ctx context.Context, userID uuid.UUID) ([]db.RecurringIncomeRule, error) {
	return s.queries.ListRecurringIncomeRulesWithPaused(ctx, userID)
}

func (s *Service) GetIncomeRule(ctx context.Context, id uuid.UUID, userID uuid.UUID) (db.RecurringIncomeRule, error) {
	slog.Debug("recurring.Service.GetIncomeRule", "id", id, "user_id", userID)
	return s.queries.GetRecurringIncomeRule(ctx, db.GetRecurringIncomeRuleParams{
		ID:     id,
		UserID: userID,
	})
}

// UpdateIncomeRule updates the rule and, when its end date moves earlier,
// reports the entries it created after the new end date and skips them unless
// req.EntriesAfterEnd is "keep" (#5).
func (s *Service) UpdateIncomeRule(ctx context.Context, id uuid.UUID, userID uuid.UUID, req UpdateIncomeRuleRequest) (IncomeRuleUpdate, error) {
	slog.Debug("recurring.Service.UpdateIncomeRule", "id", id, "user_id", userID)
	if err := validEntriesAfterEnd(req.EntriesAfterEnd); err != nil {
		return IncomeRuleUpdate{}, err
	}
	var out IncomeRuleUpdate
	// The rule and its entries change together: if skipping fails, the end
	// date isn't saved either, so a retry still sees it move (#5).
	err := s.inTx(ctx, func(q db.Querier) error {
		old, err := q.GetRecurringIncomeRule(ctx, db.GetRecurringIncomeRuleParams{ID: id, UserID: userID})
		if err != nil {
			return err
		}
		resumeThrough, err := materialize.ResumeThrough(ctx, q, userID, time.Now())
		if err != nil {
			return err
		}
		rule, err := q.UpdateRecurringIncomeRule(ctx, db.UpdateRecurringIncomeRuleParams{
			ID:            id,
			UserID:        userID,
			Amount:        decimal.NewFromFloat(req.Amount),
			Currency:      req.Currency,
			Description:   req.Description,
			RecurringType: req.RecurringType,
			StartDate:     helper.ToPgDate(req.StartDate),
			EndDate:       helper.ToPgDatePtr(req.EndDate),
			CategoryID:    helper.ToPgUUID(req.CategoryID),
			Notes:         helper.ToPgText(req.Notes),
			Priority:      optionalPriority(req.Priority),
			IsActive:      helper.ToPgBool(req.IsActive),
			ResumeThrough: resumeThrough,
		})
		if err != nil {
			return err
		}
		entries, err := incomeEntriesAfterEnd(ctx, q, rule, old.EndDate, req.EntriesAfterEnd)
		if err != nil {
			return err
		}
		out = IncomeRuleUpdate{RecurringIncomeRule: rule, EntriesAfterEnd: entries}
		return nil
	})
	if err != nil {
		return IncomeRuleUpdate{}, err
	}
	s.syncIncomeInstance(ctx, out.RecurringIncomeRule)
	return out, nil
}

func (s *Service) DeleteIncomeRule(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	slog.Debug("recurring.Service.DeleteIncomeRule", "id", id, "user_id", userID)
	return s.queries.DeleteRecurringIncomeRule(ctx, db.DeleteRecurringIncomeRuleParams{
		ID:     id,
		UserID: userID,
	})
}

// --- Expense Rules ---

func (s *Service) CreateExpenseRule(ctx context.Context, userID uuid.UUID, req CreateExpenseRuleRequest) (db.RecurringExpenseRule, error) {
	slog.Debug("recurring.Service.CreateExpenseRule", "user_id", userID)
	rule, err := s.queries.CreateRecurringExpenseRule(ctx, db.CreateRecurringExpenseRuleParams{
		UserID:        userID,
		Description:   req.Description,
		Amount:        decimal.NewFromFloat(req.Amount),
		Currency:      req.Currency,
		CategoryID:    helper.ToPgUUID(req.CategoryID),
		Notes:         helper.ToPgText(req.Notes),
		RecurringType: req.RecurringType,
		StartDate:     helper.ToPgDate(req.StartDate),
		EndDate:       helper.ToPgDatePtr(req.EndDate),
		Priority:      req.Priority,
	})
	if err != nil {
		return db.RecurringExpenseRule{}, err
	}
	s.syncExpenseInstance(ctx, rule)
	return rule, nil
}

func (s *Service) GetExpenseRule(ctx context.Context, id uuid.UUID, userID uuid.UUID) (db.GetRecurringExpenseRuleRow, error) {
	slog.Debug("recurring.Service.GetExpenseRule", "id", id, "user_id", userID)
	return s.queries.GetRecurringExpenseRule(ctx, db.GetRecurringExpenseRuleParams{
		ID:     id,
		UserID: userID,
	})
}

// UpdateExpenseRule updates the rule and, when its end date moves earlier,
// reports the entries it created after the new end date and skips them unless
// req.EntriesAfterEnd is "keep" (#5).
func (s *Service) UpdateExpenseRule(ctx context.Context, id uuid.UUID, userID uuid.UUID, req UpdateExpenseRuleRequest) (ExpenseRuleUpdate, error) {
	slog.Debug("recurring.Service.UpdateExpenseRule", "id", id, "user_id", userID)
	if err := validEntriesAfterEnd(req.EntriesAfterEnd); err != nil {
		return ExpenseRuleUpdate{}, err
	}
	var out ExpenseRuleUpdate
	// The rule and its entries change together: if skipping fails, the end
	// date isn't saved either, so a retry still sees it move (#5).
	err := s.inTx(ctx, func(q db.Querier) error {
		old, err := q.GetRecurringExpenseRule(ctx, db.GetRecurringExpenseRuleParams{ID: id, UserID: userID})
		if err != nil {
			return err
		}
		resumeThrough, err := materialize.ResumeThrough(ctx, q, userID, time.Now())
		if err != nil {
			return err
		}
		rule, err := q.UpdateRecurringExpenseRule(ctx, db.UpdateRecurringExpenseRuleParams{
			ID:            id,
			UserID:        userID,
			Description:   req.Description,
			Amount:        decimal.NewFromFloat(req.Amount),
			Currency:      req.Currency,
			CategoryID:    helper.ToPgUUID(req.CategoryID),
			Notes:         helper.ToPgText(req.Notes),
			RecurringType: req.RecurringType,
			StartDate:     helper.ToPgDate(req.StartDate),
			EndDate:       helper.ToPgDatePtr(req.EndDate),
			Priority:      req.Priority,
			IsActive:      helper.ToPgBool(req.IsActive),
			ResumeThrough: resumeThrough,
		})
		if err != nil {
			return err
		}
		entries, err := expenseEntriesAfterEnd(ctx, q, rule, old.EndDate, req.EntriesAfterEnd)
		if err != nil {
			return err
		}
		out = ExpenseRuleUpdate{RecurringExpenseRule: rule, EntriesAfterEnd: entries}
		return nil
	})
	if err != nil {
		return ExpenseRuleUpdate{}, err
	}
	s.syncExpenseInstance(ctx, out.RecurringExpenseRule)
	return out, nil
}

func (s *Service) DeleteExpenseRule(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	slog.Debug("recurring.Service.DeleteExpenseRule", "id", id, "user_id", userID)
	return s.queries.DeleteRecurringExpenseRule(ctx, db.DeleteRecurringExpenseRuleParams{
		ID:     id,
		UserID: userID,
	})
}

func (s *Service) ListExpenseRules(ctx context.Context, userID uuid.UUID) ([]db.ListRecurringExpenseRulesRow, error) {
	slog.Debug("recurring.Service.ListExpenseRules", "user_id", userID)
	return s.queries.ListRecurringExpenseRules(ctx, userID)
}

// ListExpenseRulesWithPaused lists every expense rule, paused ones too.
func (s *Service) ListExpenseRulesWithPaused(ctx context.Context, userID uuid.UUID) ([]db.ListRecurringExpenseRulesWithPausedRow, error) {
	return s.queries.ListRecurringExpenseRulesWithPaused(ctx, userID)
}

// optionalPriority maps an omitted or empty priority to NULL so the query
// keeps the stored value (or the 'want' default on create).
func optionalPriority(p *string) pgtype.Text {
	if p == nil || *p == "" {
		return pgtype.Text{}
	}
	return pgtype.Text{String: *p, Valid: true}
}
