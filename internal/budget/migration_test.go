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

// Migration 00016 backfills PHP rows at rate 1 and converts the one known
// 2026-07-12 USD gift at the 2026-07-10 rate (ADR-0003).
func TestMultiCurrencyMigration_Integration(t *testing.T) {
	ctx := context.Background()
	pool := testutil.SetupTestDBThrough(t, "00015")
	user := uuid.New()

	_, err := pool.Exec(ctx, `INSERT INTO expenses
		(user_id, amount, currency, description, expense_date)
		VALUES ($1, 100, 'PHP', 'lunch', '2026-07-12')`, user)
	require.NoError(t, err)

	_, err = pool.Exec(ctx, `INSERT INTO incomes
		(user_id, amount, currency, description, notes, date)
		VALUES ($1, 100, 'PHP', 'cash gift', 'received USD 100 cash', '2026-07-12')`, user)
	require.NoError(t, err)

	_, err = pool.Exec(ctx, `INSERT INTO incomes
		(user_id, amount, currency, description, date)
		VALUES ($1, 50, 'php', 'lowercase currency', '2026-07-13')`, user)
	require.NoError(t, err)

	testutil.ApplyMigration(t, pool, "00016")

	var expenseRate, expensePHP decimal.Decimal
	var expenseRateDate string
	require.NoError(t, pool.QueryRow(ctx,
		`SELECT exchange_rate, amount_php, rate_date::text FROM expenses WHERE user_id = $1`, user).
		Scan(&expenseRate, &expensePHP, &expenseRateDate))
	assert.True(t, decimal.NewFromInt(1).Equal(expenseRate))
	assert.True(t, decimal.NewFromInt(100).Equal(expensePHP))
	assert.Equal(t, "2026-07-12", expenseRateDate)

	var currency string
	var amount, rate, amountPHP decimal.Decimal
	var rateDate string
	require.NoError(t, pool.QueryRow(ctx,
		`SELECT currency, amount, exchange_rate, amount_php, rate_date::text FROM incomes WHERE user_id = $1`, user).
		Scan(&currency, &amount, &rate, &amountPHP, &rateDate))
	assert.Equal(t, "USD", currency)
	assert.True(t, decimal.NewFromInt(100).Equal(amount))
	assert.True(t, decimal.RequireFromString("61.573").Equal(rate))
	assert.True(t, decimal.RequireFromString("6157.30").Equal(amountPHP))
	assert.Equal(t, "2026-07-10", rateDate)

	var lowerCurrency string
	var lowerRate, lowerPHP decimal.Decimal
	require.NoError(t, pool.QueryRow(ctx,
		`SELECT currency, exchange_rate, amount_php FROM incomes WHERE user_id = $1 AND description = 'lowercase currency'`, user).
		Scan(&lowerCurrency, &lowerRate, &lowerPHP))
	assert.Equal(t, "PHP", lowerCurrency)
	assert.True(t, decimal.NewFromInt(1).Equal(lowerRate))
	assert.True(t, decimal.NewFromInt(50).Equal(lowerPHP))

	// Down restores the gift, so a later Up does not trip its non-PHP guard.
	testutil.ApplyMigrationDown(t, pool, "00016")
	var restoredCurrency string
	var restoredAmount decimal.Decimal
	require.NoError(t, pool.QueryRow(ctx,
		`SELECT currency, amount FROM incomes WHERE user_id = $1 AND description = 'cash gift'`, user).
		Scan(&restoredCurrency, &restoredAmount))
	assert.Equal(t, "PHP", restoredCurrency)
	assert.True(t, decimal.NewFromInt(100).Equal(restoredAmount))

	testutil.ApplyMigration(t, pool, "00016")
	require.NoError(t, pool.QueryRow(ctx,
		`SELECT currency, amount FROM incomes WHERE user_id = $1 AND description = 'cash gift'`, user).
		Scan(&restoredCurrency, &restoredAmount))
	assert.Equal(t, "USD", restoredCurrency)
	assert.True(t, decimal.NewFromInt(100).Equal(restoredAmount))
}

// The migration refuses a non-PHP row it has no historical rate for.
func TestMultiCurrencyMigration_RejectsOtherNonPHP_Integration(t *testing.T) {
	ctx := context.Background()
	pool := testutil.SetupTestDBThrough(t, "00015")

	_, err := pool.Exec(ctx, `INSERT INTO incomes
		(user_id, amount, currency, description, date)
		VALUES ($1, 10, 'USD', 'already dollars', '2026-01-01')`, uuid.New())
	require.NoError(t, err)

	err = testutil.ApplyMigrationErr(t, pool, "00016")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unexpected non-PHP")
}

// Import trusts a file's conversion fields instead of looking up a new rate,
// so past totals cannot move (ADR-0003).
func TestImportKeepsConvertedFields_Integration(t *testing.T) {
	ctx := context.Background()
	pool := testutil.SetupTestDB(t)
	q := db.New(pool)
	svc := NewServiceWithRates(q, pool, importRates{})
	source, target := uuid.New(), uuid.New()
	date := pgtype.Date{Time: time.Date(2026, 7, 12, 0, 0, 0, 0, time.UTC), Valid: true}

	_, err := q.CreateIncome(ctx, db.CreateIncomeParams{
		UserID:       source,
		Amount:       decimal.NewFromInt(100),
		Currency:     "USD",
		ExchangeRate: decimal.RequireFromString("61.573"),
		AmountPhp:    decimal.RequireFromString("6157.30"),
		RateDate:     pgtype.Date{Time: time.Date(2026, 7, 10, 0, 0, 0, 0, time.UTC), Valid: true},
		Description:  "cash gift",
		Date:         date,
		Priority:     "want",
		Status:       "posted",
	})
	require.NoError(t, err)

	export, err := svc.Export(ctx, source)
	require.NoError(t, err)
	data, version := writeAndRead(t, export)
	clearOwner(t, pool, source)
	importOK(t, svc, target, data, version)

	got, err := q.ListAllIncomesByUser(ctx, target)
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, "USD", got[0].Currency)
	assert.True(t, decimal.NewFromInt(100).Equal(got[0].Amount))
	assert.True(t, decimal.RequireFromString("61.573").Equal(got[0].ExchangeRate))
	assert.True(t, decimal.RequireFromString("6157.30").Equal(got[0].AmountPhp))
	assert.Equal(t, "2026-07-10", got[0].RateDate.Time.Format("2006-01-02"))
}
