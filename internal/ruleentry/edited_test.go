package ruleentry

import (
	"context"
	"net/url"
	"testing"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ejsadiarin/corefinance/internal/expense"
	"github.com/ejsadiarin/corefinance/internal/income"
)

// Rule entries remember whether the user edited them (#12): editing or
// skipping one through the API marks it; the worker creating it does not.
func TestRuleEntries_EditedMark_Integration(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()
	user := uuid.New()
	rent := f.expenseRule(t, user, "monthly", "2026-09-01")
	salary := f.incomeRule(t, user, "monthly", "2026-09-05")
	f.pass(t, "2026-10-20")

	byDate := func() map[string]Entry {
		out := map[string]Entry{}
		for _, e := range f.list(t, user, url.Values{}) {
			out[string(e.Kind)+" "+e.Date] = e
		}
		return out
	}
	before := byDate()
	require.Len(t, before, 4)
	for k, e := range before {
		assert.Nil(t, e.UserEditedAt, "%s: the worker's entries start unmarked", k)
		assert.True(t, decimal.NewFromInt(1).Equal(e.ExchangeRate), "%s: PHP rate", k)
		assert.True(t, e.Amount.Equal(e.AmountPHP), "%s: PHP amount", k)
		assert.Equal(t, e.Date, e.RateDate, "%s: PHP rate date", k)
	}

	edited := before["expense 2026-09-01"]
	_, err := expense.NewService(f.q).Update(ctx, uuid.MustParse(edited.ID), user, expense.UpdateRequest{
		Amount: 120, Currency: "PHP", Description: "rent (raised)", ExpenseDate: "2026-09-01",
		Priority: "need", Status: "posted", SourceRuleID: ptr(rent.String()),
	})
	require.NoError(t, err)
	require.NoError(t, expense.NewService(f.q).Skip(ctx, uuid.MustParse(before["expense 2026-10-01"].ID), user))
	require.NoError(t, income.NewService(f.q).Skip(ctx, uuid.MustParse(before["income 2026-10-05"].ID), user))

	after := byDate()
	assert.NotNil(t, after["expense 2026-09-01"].UserEditedAt, "edited")
	assert.NotNil(t, after["expense 2026-10-01"].UserEditedAt, "skipped expense")
	assert.NotNil(t, after["income 2026-10-05"].UserEditedAt, "skipped income")
	assert.Nil(t, after["income 2026-09-05"].UserEditedAt, "untouched")
	_ = salary
}

func ptr[T any](v T) *T { return &v }
