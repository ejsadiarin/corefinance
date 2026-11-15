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

	"github.com/ejsadiarin/corefinance/internal/auth"
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

// fill runs the owner's materialize pass in q's transaction (#14, ADR-0001):
// the one writer of rule entries, the same code the worker runs. Rule saves
// call lockOwner first, so they take the owner's lock before any rule row,
// in the same order the worker does.
func fill(ctx context.Context, q db.Querier, userID uuid.UUID, loc *time.Location) error {
	n, err := materialize.MaterializeUser(ctx, q, userID, loc, time.Now())
	if err != nil {
		return fmt.Errorf("fill rule entries: %w", err)
	}
	if n > 0 {
		slog.Info("recurring.fill: filled rule entries", "user_id", userID, "inserted", n)
	}
	return nil
}

// ownerZone is the rule owner's timezone (coregateway ADR-0005): the zone in
// the request's internal token (rule writes are always the caller's own), or,
// for a token without one, the zone last seen for the owner (UTC if none).
func ownerZone(ctx context.Context, q db.Querier, userID uuid.UUID) (*time.Location, error) {
	if tz := auth.TimezoneFromContext(ctx); tz != "" {
		return materialize.LoadZone(tz), nil
	}
	loc, err := materialize.SeenZones{Queries: q}.Zone(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("owner timezone: %w", err)
	}
	return loc, nil
}

// lockOwnerInZone takes the owner's lock and then reads their zone, so a zone
// recorded concurrently is either already committed or waits for this save.
func lockOwnerInZone(ctx context.Context, q db.Querier, userID uuid.UUID) (*time.Location, error) {
	if err := lockOwner(ctx, q, userID); err != nil {
		return nil, err
	}
	return ownerZone(ctx, q, userID)
}

func lockOwner(ctx context.Context, q db.Querier, userID uuid.UUID) error {
	if err := q.LockUserMaterialize(ctx, userID); err != nil {
		return fmt.Errorf("lock rule owner: %w", err)
	}
	return nil
}

// --- Income Rules ---

func (s *Service) CreateIncomeRule(ctx context.Context, userID uuid.UUID, req CreateIncomeRuleRequest) (db.RecurringIncomeRule, error) {
	slog.Debug("recurring.Service.CreateIncomeRule", "user_id", userID)
	var rule db.RecurringIncomeRule
	err := s.inTx(ctx, func(q db.Querier) error {
		loc, err := lockOwnerInZone(ctx, q, userID)
		if err != nil {
			return err
		}
		r, err := q.CreateRecurringIncomeRule(ctx, db.CreateRecurringIncomeRuleParams{
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
			return err
		}
		rule = r
		return fill(ctx, q, userID, loc)
	})
	if err != nil {
		return db.RecurringIncomeRule{}, err
	}
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

// UpdateIncomeRule updates the rule and, when its schedule changes,
// reconciles the entries it created against the new schedule (#15): stale
// untouched entries are skipped unless req says "keep", edited ones are
// reported for review.
func (s *Service) UpdateIncomeRule(ctx context.Context, id uuid.UUID, userID uuid.UUID, req UpdateIncomeRuleRequest) (IncomeRuleUpdate, error) {
	slog.Debug("recurring.Service.UpdateIncomeRule", "id", id, "user_id", userID)
	choice := staleChoice(req.StaleEntries, req.EntriesAfterEnd)
	if err := validStaleChoice(choice); err != nil {
		return IncomeRuleUpdate{}, err
	}
	var out IncomeRuleUpdate
	// The rule and its entries change together: if reconciling fails, the
	// schedule change isn't saved either, so a retry still sees it (#15).
	err := s.inTx(ctx, func(q db.Querier) error {
		loc, err := lockOwnerInZone(ctx, q, userID)
		if err != nil {
			return err
		}
		old, err := q.GetRecurringIncomeRule(ctx, db.GetRecurringIncomeRuleParams{ID: id, UserID: userID})
		if err != nil {
			return err
		}
		// The day before the owner's today: where a resumed rule, or one
		// whose schedule changed, carries on from.
		yesterday := materialize.ResumeThrough(loc, time.Now())
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
			ResumeThrough: yesterday,
		})
		if err != nil {
			return err
		}
		was := schedule{start: old.StartDate, cadence: old.RecurringType, end: old.EndDate}
		now := schedule{start: rule.StartDate, cadence: rule.RecurringType, end: rule.EndDate}
		stale, err := reconcile(ctx, q, userID, rule.ID, was, now, choice)
		if err != nil {
			return err
		}
		if err := noBackfill(ctx, q, rule.ID, was, now, yesterday); err != nil {
			return err
		}
		out = IncomeRuleUpdate{RecurringIncomeRule: rule, StaleEntries: stale}
		return fill(ctx, q, userID, loc)
	})
	if err != nil {
		return IncomeRuleUpdate{}, err
	}
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
	var rule db.RecurringExpenseRule
	err := s.inTx(ctx, func(q db.Querier) error {
		loc, err := lockOwnerInZone(ctx, q, userID)
		if err != nil {
			return err
		}
		r, err := q.CreateRecurringExpenseRule(ctx, db.CreateRecurringExpenseRuleParams{
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
			return err
		}
		rule = r
		return fill(ctx, q, userID, loc)
	})
	if err != nil {
		return db.RecurringExpenseRule{}, err
	}
	return rule, nil
}

func (s *Service) GetExpenseRule(ctx context.Context, id uuid.UUID, userID uuid.UUID) (db.GetRecurringExpenseRuleRow, error) {
	slog.Debug("recurring.Service.GetExpenseRule", "id", id, "user_id", userID)
	return s.queries.GetRecurringExpenseRule(ctx, db.GetRecurringExpenseRuleParams{
		ID:     id,
		UserID: userID,
	})
}

// UpdateExpenseRule updates the rule and, when its schedule changes,
// reconciles the entries it created against the new schedule (#15): stale
// untouched entries are skipped unless req says "keep", edited ones are
// reported for review.
func (s *Service) UpdateExpenseRule(ctx context.Context, id uuid.UUID, userID uuid.UUID, req UpdateExpenseRuleRequest) (ExpenseRuleUpdate, error) {
	slog.Debug("recurring.Service.UpdateExpenseRule", "id", id, "user_id", userID)
	choice := staleChoice(req.StaleEntries, req.EntriesAfterEnd)
	if err := validStaleChoice(choice); err != nil {
		return ExpenseRuleUpdate{}, err
	}
	var out ExpenseRuleUpdate
	// The rule and its entries change together: if reconciling fails, the
	// schedule change isn't saved either, so a retry still sees it (#15).
	err := s.inTx(ctx, func(q db.Querier) error {
		loc, err := lockOwnerInZone(ctx, q, userID)
		if err != nil {
			return err
		}
		old, err := q.GetRecurringExpenseRule(ctx, db.GetRecurringExpenseRuleParams{ID: id, UserID: userID})
		if err != nil {
			return err
		}
		// The day before the owner's today: where a resumed rule, or one
		// whose schedule changed, carries on from.
		yesterday := materialize.ResumeThrough(loc, time.Now())
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
			ResumeThrough: yesterday,
		})
		if err != nil {
			return err
		}
		was := schedule{start: old.StartDate, cadence: old.RecurringType, end: old.EndDate}
		now := schedule{start: rule.StartDate, cadence: rule.RecurringType, end: rule.EndDate}
		stale, err := reconcile(ctx, q, userID, rule.ID, was, now, choice)
		if err != nil {
			return err
		}
		if err := noBackfill(ctx, q, rule.ID, was, now, yesterday); err != nil {
			return err
		}
		out = ExpenseRuleUpdate{RecurringExpenseRule: rule, StaleEntries: stale}
		return fill(ctx, q, userID, loc)
	})
	if err != nil {
		return ExpenseRuleUpdate{}, err
	}
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
