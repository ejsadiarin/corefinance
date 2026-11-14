package recurring

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	db "github.com/ejsadiarin/corefinance/internal/db/sqlc"
	"github.com/ejsadiarin/corefinance/internal/materialize"
)

// Reconciling a rule's entries after a schedule change (#15, ADR-0001). When
// an update changes a rule's start date, cadence or end date, the entries the
// rule already created are compared with the new schedule. Entries it no
// longer produces ("stale") are skipped if the user never edited them,
// unless the caller keeps them; entries the user edited are kept and
// reported for review. Nothing is deleted, and a schedule change adds no
// entries before today: the writer carries on from where it is, so entries
// the user deleted are never recreated. Ending a rule in the past (#5) is
// one case of this.

// What happens to untouched stale entries.
const (
	StaleSkip = "skip" // the default
	StaleKeep = "keep"
)

// ErrInvalidStaleEntries is returned for a stale_entries other than "skip",
// "keep" or empty.
var ErrInvalidStaleEntries = errors.New(`stale_entries must be "skip" or "keep"`)

// StaleEntry is a rule entry the new schedule no longer produces, with its
// status after the update. NeedsReview marks one the user had edited: it is
// kept as it is, for them to decide.
type StaleEntry struct {
	ID          string `json:"id"`
	Date        string `json:"date"`
	Status      string `json:"status"`
	NeedsReview bool   `json:"needs_review"`
}

// ExpenseRuleUpdate is the rule as updated plus its stale entries (empty
// unless the schedule changed).
type ExpenseRuleUpdate struct {
	db.RecurringExpenseRule
	StaleEntries []StaleEntry `json:"stale_entries"`
}

// IncomeRuleUpdate is the income-rule counterpart of ExpenseRuleUpdate.
type IncomeRuleUpdate struct {
	db.RecurringIncomeRule
	StaleEntries []StaleEntry `json:"stale_entries"`
}

func validStaleChoice(choice string) error {
	switch choice {
	case "", StaleSkip, StaleKeep:
		return nil
	}
	return ErrInvalidStaleEntries
}

// schedule is the part of a rule that decides which dates it produces.
type schedule struct {
	start   pgtype.Date
	cadence string
	end     pgtype.Date
}

func sameDate(a, b pgtype.Date) bool {
	return a.Valid == b.Valid && (!a.Valid || a.Time.Equal(b.Time))
}

func (s schedule) equal(o schedule) bool {
	return s.cadence == o.cadence && sameDate(s.start, o.start) && sameDate(s.end, o.end)
}

// reconcile skips, keeps or reports the rule's entries that schedule `now`
// no longer produces, if it differs from `was`. It runs in the update's
// transaction.
func reconcile(ctx context.Context, q db.Querier, userID, ruleID uuid.UUID, was, now schedule, choice string) ([]StaleEntry, error) {
	stale := []StaleEntry{}
	if was.equal(now) {
		return stale, nil
	}
	rows, err := q.ListRuleEntriesToReconcile(ctx, db.ListRuleEntriesToReconcileParams{Owner: userID, RuleID: ruleID})
	if err != nil {
		return nil, fmt.Errorf("list rule entries: %w", err)
	}
	if len(rows) == 0 {
		return stale, nil
	}
	var end *time.Time
	if now.end.Valid {
		end = &now.end.Time
	}
	due := map[string]bool{}
	for _, d := range materialize.DueDates(now.start.Time, now.cadence, end, rows[len(rows)-1].Date.Time) {
		due[d.Format(time.DateOnly)] = true
	}

	var toSkip, toKeep []uuid.UUID
	for _, r := range rows {
		date := r.Date.Time.Format(time.DateOnly)
		if due[date] {
			continue
		}
		e := StaleEntry{ID: r.ID.String(), Date: date, Status: r.Status}
		switch {
		case r.UserEditedAt.Valid:
			e.NeedsReview = true
		case choice == StaleKeep:
			toKeep = append(toKeep, r.ID)
		default:
			toSkip = append(toSkip, r.ID)
			e.Status = "skipped"
		}
		stale = append(stale, e)
	}
	if len(toSkip) > 0 {
		if err := q.SkipExpenseEntries(ctx, db.SkipExpenseEntriesParams{UserID: userID, Ids: toSkip}); err != nil {
			return nil, fmt.Errorf("skip stale expenses: %w", err)
		}
		if err := q.SkipIncomeEntries(ctx, db.SkipIncomeEntriesParams{UserID: userID, Ids: toSkip}); err != nil {
			return nil, fmt.Errorf("skip stale incomes: %w", err)
		}
	}
	if len(toKeep) > 0 {
		if err := q.MarkExpenseEntriesKept(ctx, db.MarkExpenseEntriesKeptParams{UserID: userID, Ids: toKeep}); err != nil {
			return nil, fmt.Errorf("keep stale expenses: %w", err)
		}
		if err := q.MarkIncomeEntriesKept(ctx, db.MarkIncomeEntriesKeptParams{UserID: userID, Ids: toKeep}); err != nil {
			return nil, fmt.Errorf("keep stale incomes: %w", err)
		}
	}
	return stale, nil
}
