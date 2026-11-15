// Package materialize implements the recurring-rule worker core (D3/D4/D5):
// due-date computation with month-end clamping, idempotent instance inserts,
// and the one-time legacy inline-rule backfill.
package materialize

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
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

// NextDueDate is the rule's first occurrence after `after`, if it has one
// before its end. It steps straight to the right occurrence instead of
// listing every date since the start, since it runs for every rule a pass
// touches.
func NextDueDate(start time.Time, cadence string, end *time.Time, after time.Time) (time.Time, bool) {
	start, after = truncateDate(start), truncateDate(after)
	var d time.Time
	switch {
	case after.Before(start):
		d = start
	case cadence == "daily":
		d = after.AddDate(0, 0, 1)
	case cadence == "weekly":
		d = start.AddDate(0, 0, (daysBetween(start, after)/7+1)*7)
	case cadence == "monthly":
		i := (after.Year()-start.Year())*12 + int(after.Month()-start.Month())
		for d = addMonthsClamped(start, i); !d.After(after); d = addMonthsClamped(start, i) {
			i++
		}
	case cadence == "yearly":
		i := after.Year() - start.Year()
		for d = addYearsClamped(start, i); !d.After(after); d = addYearsClamped(start, i) {
			i++
		}
	default:
		return time.Time{}, false
	}
	if end != nil && d.After(truncateDate(*end)) {
		return time.Time{}, false
	}
	return d, true
}

// MaxDatesPerRulePass bounds how many dates one pass fills for a single rule.
// A long outage (or a daily rule with a years-old start and no watermark)
// fills in over several passes instead of one huge transaction; the
// watermark records where each pass stopped.
const MaxDatesPerRulePass = 400

// catchUpBound is the last date a rule can be filled through: min(end, today).
func catchUpBound(end pgtype.Date, today time.Time) time.Time {
	bound := truncateDate(today)
	if end.Valid {
		if e := truncateDate(end.Time); e.Before(bound) {
			bound = e
		}
	}
	return bound
}

// PendingDates returns the due dates of a rule after its watermark, up to
// min(end, today), capped at limit, and the date the watermark should move to
// once those dates are inserted: the last returned date when the cap was hit,
// otherwise the bound. An invalid watermark means nothing has been filled yet.
func PendingDates(start time.Time, cadence string, watermark, end pgtype.Date, today time.Time, limit int) (dates []time.Time, through time.Time) {
	bound := catchUpBound(end, today)
	for _, d := range DueDates(start, cadence, pgTimeOrNil(end), today) {
		if watermark.Valid && !d.After(truncateDate(watermark.Time)) {
			continue
		}
		dates = append(dates, d)
		if len(dates) == limit {
			return dates, d
		}
	}
	return dates, bound
}

// MaterializeUser fills one user's rules (#14, ADR-0001): every active rule
// gets its due dates after its watermark (materialized_through) up to
// min(end_date, today), where today is the date at now in the user's zone
// loc, and the watermark moves forward. Missed days while the worker was
// down are filled on the next pass; rules that ended in the meantime still
// get their last entries. The watermark never moves backward, so an entry the user deleted
// is not recreated, and inserts are ON CONFLICT DO NOTHING, so entries the
// user edited or skipped are untouched.
//
// It is the only writer of rule entries: the worker runs it per user, and a
// rule save runs it for the rule's owner. q must be inside a transaction; it
// takes the user's lock for the rest of it. Returns the rows inserted.
func MaterializeUser(ctx context.Context, q db.Querier, userID uuid.UUID, loc *time.Location, now time.Time) (int, error) {
	if err := q.LockUserMaterialize(ctx, userID); err != nil {
		return 0, fmt.Errorf("lock user materialize: %w", err)
	}
	today := dateIn(now, loc)
	rules, err := q.ListUserActiveRules(ctx, userID)
	if err != nil {
		return 0, fmt.Errorf("list active rules: %w", err)
	}
	inserted := 0
	for _, r := range rules {
		dates, through := PendingDates(r.StartDate.Time, r.RecurringType, r.MaterializedThrough, r.EndDate, today, MaxDatesPerRulePass)
		for _, d := range dates {
			n, err := insertRuleEntry(ctx, q, r, d)
			if err != nil {
				return inserted, err
			}
			inserted += int(n)
		}
		filled := r.MaterializedThrough
		if !filled.Valid || through.After(truncateDate(filled.Time)) {
			if err := q.AdvanceRuleWatermark(ctx, db.AdvanceRuleWatermarkParams{ID: r.ID, Through: pgDate(through)}); err != nil {
				return inserted, fmt.Errorf("advance rule watermark: %w", err)
			}
			filled = pgDate(through)
		}
		if err := q.SetRuleNextDue(ctx, db.SetRuleNextDueParams{ID: r.ID, NextDueAt: nextDueAt(r, filled, loc)}); err != nil {
			return inserted, fmt.Errorf("set rule next due: %w", err)
		}
	}
	return inserted, nil
}

// nextDueAt is when the rule's next unfilled occurrence starts in loc, or
// infinity once the rule has no more occurrences.
func nextDueAt(r db.Rule, filled pgtype.Date, loc *time.Location) pgtype.Timestamptz {
	after := truncateDate(r.StartDate.Time).AddDate(0, 0, -1)
	if filled.Valid && truncateDate(filled.Time).After(after) {
		after = truncateDate(filled.Time)
	}
	d, ok := NextDueDate(r.StartDate.Time, r.RecurringType, pgTimeOrNil(r.EndDate), after)
	if !ok {
		return pgtype.Timestamptz{InfinityModifier: pgtype.Infinity, Valid: true}
	}
	y, m, day := d.Date()
	return pgtype.Timestamptz{Time: time.Date(y, m, day, 0, 0, 0, 0, loc), Valid: true}
}

// ErrNothingDue is NextWake's answer when no active rule has an occurrence
// still to come.
var ErrNothingDue = errors.New("no rule is due")

// NextWake is when the earliest active rule's next occurrence starts, after
// now: the worker sleeps until then (#16).
func NextWake(ctx context.Context, q db.Querier, now time.Time) (time.Time, error) {
	t, err := q.GetNextWake(ctx, pgtype.Timestamptz{Time: now, Valid: true})
	if err != nil {
		return time.Time{}, fmt.Errorf("next wake: %w", err)
	}
	if !t.Valid {
		return time.Time{}, ErrNothingDue
	}
	return t.Time, nil
}

// insertRuleEntry creates the rule's entry for date d in the table for its
// kind; an existing entry for that rule and date is left alone.
func insertRuleEntry(ctx context.Context, q db.Querier, r db.Rule, d time.Time) (int64, error) {
	source := pgtype.UUID{Bytes: r.ID, Valid: true}
	switch r.Kind {
	case "expense":
		n, err := q.CreateExpenseInstance(ctx, db.CreateExpenseInstanceParams{
			UserID: r.UserID, CategoryID: r.CategoryID, Amount: r.Amount, Currency: r.Currency,
			Description: r.Description, Notes: r.Notes, ExpenseDate: pgDate(d), Priority: r.Priority,
			IsDebt: false, SourceRuleID: source,
		})
		if err != nil {
			return 0, fmt.Errorf("create expense instance: %w", err)
		}
		return n, nil
	case "income":
		n, err := q.CreateIncomeInstance(ctx, db.CreateIncomeInstanceParams{
			UserID: r.UserID, CategoryID: r.CategoryID, Amount: r.Amount, Currency: r.Currency,
			Description: r.Description, Notes: r.Notes, Date: pgDate(d), Priority: r.Priority,
			SourceRuleID: source,
		})
		if err != nil {
			return 0, fmt.Errorf("create income instance: %w", err)
		}
		return n, nil
	}
	return 0, fmt.Errorf("rule %s has unknown kind %q", r.ID, r.Kind)
}

// materializeHistory inserts one instance row per due date bounded by
// start..min(end, today) for every active rule. Used by the backfill path
// (D5 Phase 1) to converge history; the regular pass uses catchUp.
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
		if err := q.AdvanceExpenseRuleWatermark(ctx, db.AdvanceExpenseRuleWatermarkParams{ID: r.ID, Through: pgDate(catchUpBound(r.EndDate, today))}); err != nil {
			return expenseN, incomeN, fmt.Errorf("advance expense rule watermark: %w", err)
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
				CategoryID:   r.CategoryID,
				Amount:       r.Amount,
				Currency:     r.Currency,
				Description:  r.Description,
				Notes:        r.Notes,
				Date:         pgDate(d),
				Priority:     r.Priority,
				SourceRuleID: pgtype.UUID{Bytes: r.ID, Valid: true},
			})
			if err != nil {
				return expenseN, incomeN, fmt.Errorf("create income instance: %w", err)
			}
			incomeN += int(n)
		}
		if err := q.AdvanceIncomeRuleWatermark(ctx, db.AdvanceIncomeRuleWatermarkParams{ID: r.ID, Through: pgDate(catchUpBound(r.EndDate, today))}); err != nil {
			return expenseN, incomeN, fmt.Errorf("advance income rule watermark: %w", err)
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

// MaterializeDue runs MaterializeUser, in the zone zones gives, for every user
// with a rule due by now (its next_due_at has passed, or isn't computed
// yet), each in its own short transaction, so one user's pass never waits on
// another's and a failure for one user doesn't stop the rest. Instance rows use
// recurring_type='one-time', NULL start/end, source_rule_id set,
// status='posted'. Returns the rows inserted, and every user's error joined.
func MaterializeDue(ctx context.Context, pool *pgxpool.Pool, queries *db.Queries, zones Zones, now time.Time) (int, error) {
	users, err := queries.ListUsersDue(ctx, pgtype.Timestamptz{Time: now, Valid: true})
	if err != nil {
		return 0, fmt.Errorf("list users due: %w", err)
	}
	var errs []error
	inserted := 0
	for _, user := range users {
		n, err := materializeUserTx(ctx, pool, queries, zones, user, now)
		inserted += n
		if err != nil {
			errs = append(errs, fmt.Errorf("user %s: %w", user, err))
		}
	}
	return inserted, errors.Join(errs...)
}

// materializeUserTx reads the user's zone once it holds their lock, so a zone
// recorded meanwhile (which takes the same lock) is committed and seen, and
// its reset of the next due times isn't overwritten with times in the old
// zone. A failed read skips the user rather than guess; the next pass retries.
func materializeUserTx(ctx context.Context, pool *pgxpool.Pool, queries *db.Queries, zones Zones, user uuid.UUID, now time.Time) (int, error) {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("begin: %w", err)
	}
	defer tx.Rollback(ctx)
	q := queries.WithTx(tx)
	if err := q.LockUserMaterialize(ctx, user); err != nil {
		return 0, fmt.Errorf("lock user materialize: %w", err)
	}
	loc, err := zones.Zone(ctx, user)
	if err != nil {
		return 0, fmt.Errorf("timezone: %w", err)
	}
	n, err := MaterializeUser(ctx, q, user, loc, now)
	if err != nil {
		return 0, err
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("commit: %w", err)
	}
	return n, nil
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
			if err := q.AdvanceExpenseRuleWatermark(ctx, db.AdvanceExpenseRuleWatermarkParams{ID: rule.ID, Through: pgDate(catchUpBound(leg.EndDate, today))}); err != nil {
				return fmt.Errorf("advance expense rule watermark: %w", err)
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
			if err := q.AdvanceIncomeRuleWatermark(ctx, db.AdvanceIncomeRuleWatermarkParams{ID: rule.ID, Through: pgDate(catchUpBound(leg.EndDate, today))}); err != nil {
				return fmt.Errorf("advance income rule watermark: %w", err)
			}
			if err := q.FlipLegacyIncomeToOneTime(ctx, leg.ID); err != nil {
				return fmt.Errorf("flip legacy income %s: %w", leg.ID, err)
			}
		}
		return nil
	})
	return res, err
}

// CountLegacyRowsForUser counts legacy inline rows for one user only.
// ponytail: reuses the global advisory lock caller-side via BackfillLegacyForUser;
// the count itself is lock-free and safe to call for dry_run.
func CountLegacyRowsForUser(ctx context.Context, q *db.Queries, userID uuid.UUID) (expenses, incomes int64, err error) {
	expenses, err = q.CountLegacyExpensesByUser(ctx, userID)
	if err != nil {
		return 0, 0, fmt.Errorf("count legacy expenses: %w", err)
	}
	incomes, err = q.CountLegacyIncomesByUser(ctx, userID)
	if err != nil {
		return 0, 0, fmt.Errorf("count legacy incomes: %w", err)
	}
	return expenses, incomes, nil
}

// materializeHistoryForUser is the per-user variant of materializeHistory:
// only the given user's active rules are converged. Same idempotency
// (ON CONFLICT DO NOTHING), same global advisory lock via the caller.
func materializeHistoryForUser(ctx context.Context, q *db.Queries, userID uuid.UUID, today time.Time) (expenseN, incomeN int, err error) {
	todayPg := pgDate(today)

	expenseRules, err := q.ListActiveExpenseRulesByUser(ctx, db.ListActiveExpenseRulesByUserParams{
		UserID:    userID,
		StartDate: todayPg,
	})
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
		if err := q.AdvanceExpenseRuleWatermark(ctx, db.AdvanceExpenseRuleWatermarkParams{ID: r.ID, Through: pgDate(catchUpBound(r.EndDate, today))}); err != nil {
			return expenseN, incomeN, fmt.Errorf("advance expense rule watermark: %w", err)
		}
	}

	incomeRules, err := q.ListActiveIncomeRulesByUser(ctx, db.ListActiveIncomeRulesByUserParams{
		UserID:    userID,
		StartDate: todayPg,
	})
	if err != nil {
		return expenseN, incomeN, fmt.Errorf("list active income rules: %w", err)
	}
	for _, r := range incomeRules {
		for _, d := range DueDates(r.StartDate.Time, r.RecurringType, pgTimeOrNil(r.EndDate), today) {
			n, err := q.CreateIncomeInstance(ctx, db.CreateIncomeInstanceParams{
				UserID:       r.UserID,
				CategoryID:   r.CategoryID,
				Amount:       r.Amount,
				Currency:     r.Currency,
				Description:  r.Description,
				Notes:        r.Notes,
				Date:         pgDate(d),
				Priority:     r.Priority,
				SourceRuleID: pgtype.UUID{Bytes: r.ID, Valid: true},
			})
			if err != nil {
				return expenseN, incomeN, fmt.Errorf("create income instance: %w", err)
			}
			incomeN += int(n)
		}
		if err := q.AdvanceIncomeRuleWatermark(ctx, db.AdvanceIncomeRuleWatermarkParams{ID: r.ID, Through: pgDate(catchUpBound(r.EndDate, today))}); err != nil {
			return expenseN, incomeN, fmt.Errorf("advance income rule watermark: %w", err)
		}
	}
	return expenseN, incomeN, nil
}

// BackfillLegacyForUser is the per-user variant of BackfillLegacy used by the
// admin HTTP endpoint: phase 1 converges only the user's active rules, phase 2
// converts only the user's legacy rows. Global advisory lock is reused, so
// concurrent CLI and HTTP backfills still serialize.
func BackfillLegacyForUser(ctx context.Context, pool *pgxpool.Pool, queries *db.Queries, userID uuid.UUID, today time.Time) (BackfillResult, error) {
	var res BackfillResult
	err := withMaterializeLock(ctx, pool, queries, func(q *db.Queries) error {
		e, i, err := materializeHistoryForUser(ctx, q, userID, today)
		if err != nil {
			return err
		}
		res.ExpenseInstances += e
		res.IncomeInstances += i

		legacyExpenses, err := q.ListLegacyExpensesByUser(ctx, userID)
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
			if err := q.AdvanceExpenseRuleWatermark(ctx, db.AdvanceExpenseRuleWatermarkParams{ID: rule.ID, Through: pgDate(catchUpBound(leg.EndDate, today))}); err != nil {
				return fmt.Errorf("advance expense rule watermark: %w", err)
			}
			if err := q.FlipLegacyExpenseToOneTime(ctx, leg.ID); err != nil {
				return fmt.Errorf("flip legacy expense %s: %w", leg.ID, err)
			}
		}

		legacyIncomes, err := q.ListLegacyIncomesByUser(ctx, userID)
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
			if err := q.AdvanceIncomeRuleWatermark(ctx, db.AdvanceIncomeRuleWatermarkParams{ID: rule.ID, Through: pgDate(catchUpBound(leg.EndDate, today))}); err != nil {
				return fmt.Errorf("advance income rule watermark: %w", err)
			}
			if err := q.FlipLegacyIncomeToOneTime(ctx, leg.ID); err != nil {
				return fmt.Errorf("flip legacy income %s: %w", leg.ID, err)
			}
		}
		return nil
	})
	return res, err
}
