package budget

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	db "github.com/ejsadiarin/corefinance/internal/db/sqlc"
	"github.com/ejsadiarin/corefinance/internal/testutil"
)

// A record's time survives an export file and its import; a record without
// one, or from a file written before times existed, imports without one.
func TestExportImport_KeepsTheTime_Integration(t *testing.T) {
	ctx := context.Background()
	pool := testutil.SetupTestDB(t)
	q := db.New(pool)
	svc := NewService(q, pool)
	source, target := uuid.New(), uuid.New()
	day := pgtype.Date{Time: time.Date(2026, 11, 24, 0, 0, 0, 0, time.UTC), Valid: true}
	at := time.Date(2026, 11, 24, 15, 30, 0, 0, time.UTC)
	php := func(n int64) (decimal.Decimal, decimal.Decimal, decimal.Decimal) {
		return decimal.NewFromInt(n), decimal.NewFromInt(1), decimal.NewFromInt(n)
	}

	amount, rate, amountPHP := php(250)
	_, err := q.CreateExpense(ctx, db.CreateExpenseParams{
		UserID: source, Amount: amount, Currency: "PHP", ExchangeRate: rate, AmountPhp: amountPHP,
		RateDate: day, Description: "dinner", ExpenseDate: day, Priority: "want", Status: "posted",
		OccurredAt: pgtype.Timestamptz{Time: at, Valid: true},
	})
	require.NoError(t, err)
	_, err = q.CreateExpense(ctx, db.CreateExpenseParams{
		UserID: source, Amount: amount, Currency: "PHP", ExchangeRate: rate, AmountPhp: amountPHP,
		RateDate: day, Description: "snack", ExpenseDate: day, Priority: "want", Status: "posted",
	})
	require.NoError(t, err)
	amount, rate, amountPHP = php(1000)
	_, err = q.CreateIncome(ctx, db.CreateIncomeParams{
		UserID: source, Amount: amount, Currency: "PHP", ExchangeRate: rate, AmountPhp: amountPHP,
		RateDate: day, Description: "pay", Date: day, Priority: "need", Status: "posted",
		OccurredAt: pgtype.Timestamptz{Time: at, Valid: true},
	})
	require.NoError(t, err)

	export, err := svc.Export(ctx, source)
	require.NoError(t, err)
	data, version := writeAndRead(t, export)
	clearOwner(t, pool, source)
	importOK(t, svc, target, data, version)

	expenses, err := q.ListAllExpensesByUser(ctx, target)
	require.NoError(t, err)
	require.Len(t, expenses, 2)
	times := map[string]pgtype.Timestamptz{}
	for _, e := range expenses {
		times[e.Description] = e.OccurredAt
	}
	assert.True(t, times["dinner"].Valid)
	assert.True(t, at.Equal(times["dinner"].Time), "dinner at %v", times["dinner"].Time)
	assert.False(t, times["snack"].Valid)

	incomes, err := q.ListAllIncomesByUser(ctx, target)
	require.NoError(t, err)
	require.Len(t, incomes, 1)
	assert.True(t, at.Equal(incomes[0].OccurredAt.Time))

	// A file from before #28 has no occurred_at at all.
	old := []byte(`{"expenses": [{"id": "` + uuid.NewString() + `", "amount": "5", "currency": "PHP",
		"exchange_rate": "1", "amount_php": "5", "rate_date": "2026-11-20", "description": "old",
		"expense_date": "2026-11-20", "priority": "want", "status": "posted"}]}`)
	legacy, v0, err := ReadPart(old)
	require.NoError(t, err)
	require.Equal(t, 0, v0)
	other := uuid.New()
	importOK(t, svc, other, legacy, v0)
	got, err := q.ListAllExpensesByUser(ctx, other)
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.False(t, got[0].OccurredAt.Valid)
}
