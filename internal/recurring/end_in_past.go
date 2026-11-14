package recurring

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	db "github.com/ejsadiarin/corefinance/internal/db/sqlc"
)

// Ending a rule in the past (#5). When an update moves a rule's end date
// earlier, the entries the rule already created after the new end date are
// reported back, and skipped unless the caller asks to keep them. Nothing is
// deleted.

// What happens to a rule's entries dated after its new end date.
const (
	EntriesAfterEndSkip = "skip" // the default
	EntriesAfterEndKeep = "keep"
)

// ErrInvalidEntriesAfterEnd is returned for an entries_after_end other than
// "skip", "keep" or empty.
var ErrInvalidEntriesAfterEnd = errors.New(`entries_after_end must be "skip" or "keep"`)

// RuleEntry is one rule-created entry dated after the rule's new end date,
// with its status after the update.
type RuleEntry struct {
	ID     string `json:"id"`
	Date   string `json:"date"`
	Status string `json:"status"`
}

// ExpenseRuleUpdate is the rule as updated plus the entries after its new
// end date (empty unless the end date moved earlier).
type ExpenseRuleUpdate struct {
	db.RecurringExpenseRule
	EntriesAfterEnd []RuleEntry `json:"entries_after_end"`
}

// IncomeRuleUpdate is the income-rule counterpart of ExpenseRuleUpdate.
type IncomeRuleUpdate struct {
	db.RecurringIncomeRule
	EntriesAfterEnd []RuleEntry `json:"entries_after_end"`
}

func validEntriesAfterEnd(choice string) error {
	switch choice {
	case "", EntriesAfterEndSkip, EntriesAfterEndKeep:
		return nil
	}
	return ErrInvalidEntriesAfterEnd
}

// endMovedEarlier reports whether an update from oldEnd to newEnd can leave
// entries after the end date: a new end date that is earlier than the old
// one, or the first end date on an open-ended rule. Only the window between
// the two is revisited, so entries a user kept on an earlier update stay put.
func endMovedEarlier(oldEnd, newEnd pgtype.Date) bool {
	return newEnd.Valid && (!oldEnd.Valid || newEnd.Time.Before(oldEnd.Time))
}

func ruleEntry(id uuid.UUID, date pgtype.Date, status string) RuleEntry {
	return RuleEntry{ID: id.String(), Date: date.Time.Format(time.DateOnly), Status: status}
}

// sortByDate orders entries by date: UPDATE ... RETURNING can't.
func sortByDate(entries []RuleEntry) []RuleEntry {
	sort.Slice(entries, func(i, j int) bool { return entries[i].Date < entries[j].Date })
	return entries
}

// expenseEntriesAfterEnd lists, and unless the choice is "keep" skips, the
// rule's entries between its new end date and oldEnd.
func expenseEntriesAfterEnd(ctx context.Context, q db.Querier, rule db.RecurringExpenseRule, oldEnd pgtype.Date, choice string) ([]RuleEntry, error) {
	entries := []RuleEntry{}
	if !endMovedEarlier(oldEnd, rule.EndDate) {
		return entries, nil
	}
	ruleID := pgtype.UUID{Bytes: rule.ID, Valid: true}
	if choice == EntriesAfterEndKeep {
		rows, err := q.ListExpenseRuleEntriesAfterEnd(ctx, db.ListExpenseRuleEntriesAfterEndParams{
			UserID: rule.UserID, RuleID: ruleID, AfterEnd: rule.EndDate, OldEnd: oldEnd,
		})
		if err != nil {
			return nil, fmt.Errorf("list expense entries after end: %w", err)
		}
		for _, r := range rows {
			entries = append(entries, ruleEntry(r.ID, r.Date, r.Status))
		}
	} else {
		rows, err := q.SkipExpenseRuleEntriesAfterEnd(ctx, db.SkipExpenseRuleEntriesAfterEndParams{
			UserID: rule.UserID, RuleID: ruleID, AfterEnd: rule.EndDate, OldEnd: oldEnd,
		})
		if err != nil {
			return nil, fmt.Errorf("skip expense entries after end: %w", err)
		}
		for _, r := range rows {
			entries = append(entries, ruleEntry(r.ID, r.Date, r.Status))
		}
	}
	return sortByDate(entries), nil
}

// incomeEntriesAfterEnd is the income-rule counterpart of
// expenseEntriesAfterEnd.
func incomeEntriesAfterEnd(ctx context.Context, q db.Querier, rule db.RecurringIncomeRule, oldEnd pgtype.Date, choice string) ([]RuleEntry, error) {
	entries := []RuleEntry{}
	if !endMovedEarlier(oldEnd, rule.EndDate) {
		return entries, nil
	}
	ruleID := pgtype.UUID{Bytes: rule.ID, Valid: true}
	if choice == EntriesAfterEndKeep {
		rows, err := q.ListIncomeRuleEntriesAfterEnd(ctx, db.ListIncomeRuleEntriesAfterEndParams{
			UserID: rule.UserID, RuleID: ruleID, AfterEnd: rule.EndDate, OldEnd: oldEnd,
		})
		if err != nil {
			return nil, fmt.Errorf("list income entries after end: %w", err)
		}
		for _, r := range rows {
			entries = append(entries, ruleEntry(r.ID, r.Date, r.Status))
		}
	} else {
		rows, err := q.SkipIncomeRuleEntriesAfterEnd(ctx, db.SkipIncomeRuleEntriesAfterEndParams{
			UserID: rule.UserID, RuleID: ruleID, AfterEnd: rule.EndDate, OldEnd: oldEnd,
		})
		if err != nil {
			return nil, fmt.Errorf("skip income entries after end: %w", err)
		}
		for _, r := range rows {
			entries = append(entries, ruleEntry(r.ID, r.Date, r.Status))
		}
	}
	return sortByDate(entries), nil
}
