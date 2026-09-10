// Package materialize implements the recurring-rule worker core (D3/D4/D5):
// due-date computation with month-end clamping, idempotent instance inserts,
// and the one-time legacy inline-rule backfill.
package materialize

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	db "github.com/ejsadiarin/corefinance/internal/db/sqlc"
)

// advisoryLockKey serializes materialization passes across worker replicas.
// pg_advisory_xact_lock is transaction-scoped, so every pass runs inside a
// single transaction holding this lock for its whole duration.
const advisoryLockKey = "corefinance-materialize"

// truncateDate drops any sub-day component, preserving location.
func truncateDate(t time.Time) time.Time {
	y, m, d := t.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, t.Location())
}

func pgDate(t time.Time) pgtype.Date {
	return pgtype.Date{Time: truncateDate(t), Valid: true}
}

func pgDateOrNull(d pgtype.Date) pgtype.Date {
	if !d.Valid {
		return pgtype.Date{}
	}
	return pgDate(d.Time)
}

func pgTimeOrNil(d pgtype.Date) *time.Time {
	if !d.Valid {
		return nil
	}
	t := truncateDate(d.Time)
	return &t
}

func daysIn(year int, month time.Month) int {
	return time.Date(year, month+1, 0, 0, 0, 0, 0, time.UTC).Day()
}

// addMonthsClamped steps n months from t keeping the original day-of-month,
// clamped to the target month length (Jan 31 + 1 month = Feb 28, then Mar 31).
func addMonthsClamped(t time.Time, n int) time.Time {
	t = truncateDate(t)
	y, m, _ := t.Date()
	target := time.Date(y, m, 1, 0, 0, 0, 0, t.Location()).AddDate(0, n, 0)
	day := t.Day()
	if last := daysIn(target.Year(), target.Month()); day > last {
		day = last
	}
	return time.Date(target.Year(), target.Month(), day, 0, 0, 0, 0, t.Location())
}

// addYearsClamped steps n years from t, clamping Feb 29 to Feb 28 in
// non-leap years.
func addYearsClamped(t time.Time, n int) time.Time {
	t = truncateDate(t)
	y, m, d := t.Date()
	if last := daysIn(y+n, m); d > last {
		d = last
	}
	return time.Date(y+n, m, d, 0, 0, 0, 0, t.Location())
}

// daysBetween returns whole calendar days from a to b (b >= a), DST-safe via
// UTC midnights.
func daysBetween(a, b time.Time) int {
	a, b = truncateDate(a.UTC()), truncateDate(b.UTC())
	return int(b.Sub(a).Hours() / 24)
}

// IsDue reports whether today is an occurrence of a rule starting at start
// with the given cadence (daily/weekly/monthly/yearly stepping from start).
// Monthly uses same day-of-month clamped to month end (Jan 31 -> Feb 28);
// yearly clamps Feb 29 to Feb 28 in non-leap years. Unknown cadences are
// never due.
func IsDue(start time.Time, cadence string, today time.Time) bool {
	start, today = truncateDate(start), truncateDate(today)
	if today.Before(start) {
		return false
	}
	switch cadence {
	case "daily":
		return true
	case "weekly":
		return daysBetween(start, today)%7 == 0
	case "monthly":
		y, m, _ := today.Date()
		day := start.Day()
		if last := daysIn(y, m); day > last {
			day = last
		}
		return today.Day() == day
	case "yearly":
		if today.Month() != start.Month() {
			return false
		}
		day := start.Day()
		if last := daysIn(today.Year(), start.Month()); day > last {
			day = last
		}
		return today.Day() == day
	}
	return false
}

// DueDates returns every occurrence date of a rule with the given start,
// cadence (daily/weekly/monthly/yearly), optional end, bounded above by
// today: start..min(end, today). Used for history backfill; the daily pass
// uses IsDue instead. Unknown cadences yield no dates.
func DueDates(start time.Time, cadence string, end *time.Time, today time.Time) []time.Time {
	start = truncateDate(start)
	bound := truncateDate(today)
	if end != nil {
		if e := truncateDate(*end); e.Before(bound) {
			bound = e
		}
	}
	if start.After(bound) {
		return nil
	}
	var out []time.Time
	switch cadence {
	case "daily":
		for d := start; !d.After(bound); d = d.AddDate(0, 0, 1) {
			out = append(out, d)
		}
	case "weekly":
		for d := start; !d.After(bound); d = d.AddDate(0, 0, 7) {
			out = append(out, d)
		}
	case "monthly":
		for i := 0; i < 1200; i++ {
			d := addMonthsClamped(start, i)
			if d.After(bound) {
				break
			}
			out = append(out, d)
		}
	case "yearly":
		for i := 0; i < 200; i++ {
			d := addYearsClamped(start, i)
			if d.After(bound) {
				break
			}
			out = append(out, d)
		}
	}
	return out
}

// materializeDueToday inserts today's instance for each active rule due
// today. Inserts are ON CONFLICT DO NOTHING against the partial unique
// indexes, so re-runs are no-ops. Returns inserted counts.
func materializeDueToday(ctx context.Context, q *db.Queries, today time.Time) (expenseN, incomeN int, err error) {
	todayPg := pgDate(today)

	expenseRules, err := q.ListActiveExpenseRules(ctx, todayPg)
	if err != nil {
		return 0, 0, fmt.Errorf("list active expense rules: %w", err)
	}
	for _, r := range expenseRules {
		if !IsDue(r.StartDate.Time, r.RecurringType, today) {
			continue
		}
		n, err := q.CreateExpenseInstance(ctx, db.CreateExpenseInstanceParams{
			UserID:       r.UserID,
			CategoryID:   r.CategoryID,
			Amount:       r.Amount,
			Currency:     r.Currency,
			Description:  r.Description,
			Notes:        r.Notes,
			ExpenseDate:  todayPg,
			Priority:     r.Priority,
			IsDebt:       false,
			SourceRuleID: pgtype.UUID{Bytes: r.ID, Valid: true},
		})
		if err != nil {
			return expenseN, incomeN, fmt.Errorf("create expense instance: %w", err)
		}
		expenseN += int(n)
	}

	incomeRules, err := q.ListActiveIncomeRules(ctx, todayPg)
	if err != nil {
		return expenseN, incomeN, fmt.Errorf("list active income rules: %w", err)
	}
	for _, r := range incomeRules {
		if !IsDue(r.StartDate.Time, r.RecurringType, today) {
			continue
		}
		n, err := q.CreateIncomeInstance(ctx, db.CreateIncomeInstanceParams{
			UserID:       r.UserID,
			CategoryID:   pgtype.UUID{},
			Amount:       r.Amount,
			Currency:     r.Currency,
			Description:  r.Description,
			Notes:        pgtype.Text{},
			Date:         todayPg,
			Priority:     "want",
			SourceRuleID: pgtype.UUID{Bytes: r.ID, Valid: true},
		})
		if err != nil {
			return expenseN, incomeN, fmt.Errorf("create income instance: %w", err)
		}
		incomeN += int(n)
	}
	return expenseN, incomeN, nil
}

// materializeHistory inserts one instance row per due date bounded by
// start..min(end, today) for every active rule. Used by the backfill path
// (D5 Phase 1) to converge history; the daily pass uses materializeDueToday.
func materializeHistory(ctx context.Context, q *db.Queries, today time.Time) (expenseN, incomeN int, err error) {
	todayPg := pgDate(today)

	expenseRules, err := q.ListActiveExpenseRules(ctx, todayPg)
	if err != nil {
		return 0, 0, fmt.Errorf("list active expense rules: %w", err)
	}
	for _, r := range expenseRules {
		for _, d := range DueDates(r.StartDate.Time, r.RecurringType, pgTimeOrNil(r.EndDate), today) {
			n, err := q.CreateExpenseInstance(ctx, db.CreateExpenseInstanceParams{
				UserID:       r.UserID,
				CategoryID:   r.CategoryID,
				Amount:       r.Amount,
				Currency:     r.Currency,
				Description:  r.Description,
				Notes:        r.Notes,
				ExpenseDate:  pgDate(d),
				Priority:     r.Priority,
				IsDebt:       false,
				SourceRuleID: pgtype.UUID{Bytes: r.ID, Valid: true},
			})
			if err != nil {
				return expenseN, incomeN, fmt.Errorf("create expense instance: %w", err)
			}
			expenseN += int(n)
		}
	}

	incomeRules, err := q.ListActiveIncomeRules(ctx, todayPg)
	if err != nil {
		return expenseN, incomeN, fmt.Errorf("list active income rules: %w", err)
	}
	for _, r := range incomeRules {
		for _, d := range DueDates(r.StartDate.Time, r.RecurringType, pgTimeOrNil(r.EndDate), today) {
			n, err := q.CreateIncomeInstance(ctx, db.CreateIncomeInstanceParams{
				UserID:       r.UserID,
				CategoryID:   pgtype.UUID{},
				Amount:       r.Amount,
				Currency:     r.Currency,
				Description:  r.Description,
				Notes:        pgtype.Text{},
				Date:         pgDate(d),
				Priority:     "want",
				SourceRuleID: pgtype.UUID{Bytes: r.ID, Valid: true},
			})
			if err != nil {
				return expenseN, incomeN, fmt.Errorf("create income instance: %w", err)
			}
			incomeN += int(n)
		}
	}
	return expenseN, incomeN, nil
}

// withMaterializeLock runs fn inside a transaction holding the
// pg_advisory_xact_lock for the whole pass.
func withMaterializeLock(ctx context.Context, pool *pgxpool.Pool, queries *db.Queries, fn func(q *db.Queries) error) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin materialize tx: %w", err)
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, "SELECT pg_advisory_xact_lock(hashtext('"+advisoryLockKey+"'))"); err != nil {
		return fmt.Errorf("acquire materialize advisory lock: %w", err)
	}
	if err := fn(queries.WithTx(tx)); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit materialize tx: %w", err)
	}
	return nil
}

// MaterializeDue inserts today's static instance row for each active rule
// due today (start_date <= today, end_date NULL or >= today). Instance rows
// use recurring_type='one-time', NULL start/end, source_rule_id set,
// status='posted'. Ended or not-yet-due rules produce nothing. Returns the
// number of rows inserted.
func MaterializeDue(ctx context.Context, pool *pgxpool.Pool, queries *db.Queries, today time.Time) (int, error) {
	inserted := 0
	err := withMaterializeLock(ctx, pool, queries, func(q *db.Queries) error {
		e, i, err := materializeDueToday(ctx, q, today)
		if err != nil {
			return err
		}
		inserted = e + i
		return nil
	})
	return inserted, err
}

// CountLegacyRows returns the number of rows still matching the legacy inline
// predicate (recurring_type != 'one-time' AND source_rule_id IS NULL). Zero
// means the backfill fully converged.
func CountLegacyRows(ctx context.Context, q *db.Queries) (int64, error) {
	e, err := q.CountLegacyExpenses(ctx)
	if err != nil {
		return 0, fmt.Errorf("count legacy expenses: %w", err)
	}
	i, err := q.CountLegacyIncomes(ctx)
	if err != nil {
		return 0, fmt.Errorf("count legacy incomes: %w", err)
	}
	return e + i, nil
}

// BackfillResult tallies one BackfillLegacy run.
type BackfillResult struct {
	ExpenseRules     int
	IncomeRules      int
	ExpenseInstances int
	IncomeInstances  int
}

// BackfillLegacy converts legacy inline rows (recurring_type != 'one-time'
// AND source_rule_id IS NULL) into real rules plus static history (D5):
// Phase 1 materializes existing rules over start..min(end, today); Phase 2
// creates one real rule per legacy row (copying amount/currency/description/
// category/notes/priority/start/end), materializes its history bounded by
// start..min(end, today) skipping the legacy row's own date (it becomes the
// anchor instance), then flips the legacy row to recurring_type='one-time'.
// Income rules have no category/notes/priority columns, so backfilled income
// instances carry the legacy row's category/notes/priority. All instances
// are born 'posted'.
func BackfillLegacy(ctx context.Context, pool *pgxpool.Pool, queries *db.Queries, today time.Time) (BackfillResult, error) {
	var res BackfillResult
	err := withMaterializeLock(ctx, pool, queries, func(q *db.Queries) error {
		e, i, err := materializeHistory(ctx, q, today)
		if err != nil {
			return err
		}
		res.ExpenseInstances += e
		res.IncomeInstances += i

		legacyExpenses, err := q.ListLegacyExpenses(ctx)
		if err != nil {
			return fmt.Errorf("list legacy expenses: %w", err)
		}
		for _, leg := range legacyExpenses {
			start := leg.ExpenseDate.Time
			if leg.StartDate.Valid {
				start = leg.StartDate.Time
			}
			rule, err := q.CreateRecurringExpenseRule(ctx, db.CreateRecurringExpenseRuleParams{
				UserID:        leg.UserID,
				Description:   leg.Description,
				Amount:        leg.Amount,
				Currency:      leg.Currency,
				CategoryID:    leg.CategoryID,
				Notes:         leg.Notes,
				RecurringType: leg.RecurringType,
				StartDate:     pgDate(start),
				EndDate:       pgDateOrNull(leg.EndDate),
				Priority:      leg.Priority,
			})
			if err != nil {
				return fmt.Errorf("create expense rule from legacy %s: %w", leg.ID, err)
			}
			res.ExpenseRules++
			anchor := truncateDate(leg.ExpenseDate.Time)
			for _, d := range DueDates(start, leg.RecurringType, pgTimeOrNil(leg.EndDate), today) {
				if d.Equal(anchor) {
					continue
				}
				n, err := q.CreateExpenseInstance(ctx, db.CreateExpenseInstanceParams{
					UserID:       leg.UserID,
					CategoryID:   leg.CategoryID,
					Amount:       leg.Amount,
					Currency:     leg.Currency,
					Description:  leg.Description,
					Notes:        leg.Notes,
					ExpenseDate:  pgDate(d),
					Priority:     leg.Priority,
					IsDebt:       leg.IsDebt,
					SourceRuleID: pgtype.UUID{Bytes: rule.ID, Valid: true},
				})
				if err != nil {
					return fmt.Errorf("backfill expense instance: %w", err)
				}
				res.ExpenseInstances += int(n)
			}
			if err := q.FlipLegacyExpenseToOneTime(ctx, leg.ID); err != nil {
				return fmt.Errorf("flip legacy expense %s: %w", leg.ID, err)
			}
		}

		legacyIncomes, err := q.ListLegacyIncomes(ctx)
		if err != nil {
			return fmt.Errorf("list legacy incomes: %w", err)
		}
		for _, leg := range legacyIncomes {
			start := leg.Date.Time
			if leg.StartDate.Valid {
				start = leg.StartDate.Time
			}
			rule, err := q.CreateRecurringIncomeRule(ctx, db.CreateRecurringIncomeRuleParams{
				UserID:        leg.UserID,
				Amount:        leg.Amount,
				Currency:      leg.Currency,
				Description:   leg.Description,
				RecurringType: leg.RecurringType,
				StartDate:     pgDate(start),
				EndDate:       pgDateOrNull(leg.EndDate),
			})
			if err != nil {
				return fmt.Errorf("create income rule from legacy %s: %w", leg.ID, err)
			}
			res.IncomeRules++
			anchor := truncateDate(leg.Date.Time)
			for _, d := range DueDates(start, leg.RecurringType, pgTimeOrNil(leg.EndDate), today) {
				if d.Equal(anchor) {
					continue
				}
				n, err := q.CreateIncomeInstance(ctx, db.CreateIncomeInstanceParams{
					UserID:       leg.UserID,
					CategoryID:   leg.CategoryID,
					Amount:       leg.Amount,
					Currency:     leg.Currency,
					Description:  leg.Description,
					Notes:        leg.Notes,
					Date:         pgDate(d),
					Priority:     leg.Priority,
					SourceRuleID: pgtype.UUID{Bytes: rule.ID, Valid: true},
				})
				if err != nil {
					return fmt.Errorf("backfill income instance: %w", err)
				}
				res.IncomeInstances += int(n)
			}
			if err := q.FlipLegacyIncomeToOneTime(ctx, leg.ID); err != nil {
				return fmt.Errorf("flip legacy income %s: %w", leg.ID, err)
			}
		}
		return nil
	})
	return res, err
}
