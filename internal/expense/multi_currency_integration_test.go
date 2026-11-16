package expense

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	db "github.com/ejsadiarin/corefinance/internal/db/sqlc"
	"github.com/ejsadiarin/corefinance/internal/rates"
	"github.com/ejsadiarin/corefinance/internal/testutil"
)

type failingRates struct{}

func (failingRates) Rate(context.Context, string, time.Time) (rates.Rate, error) {
	return rates.Rate{}, rates.ErrRateUnavailable
}

// A provider failure refuses the save before any row is written (ADR-0003).
func TestCreate_ProviderFailureWritesNothing_Integration(t *testing.T) {
	pool := testutil.SetupTestDB(t)
	ctx := context.Background()
	q := db.New(pool)
	user := uuid.New()

	_, err := NewServiceWithRates(q, failingRates{}).Create(ctx, user, CreateRequest{
		Amount: 100, Currency: "USD", Description: "gift", ExpenseDate: "2026-07-12",
		Priority: "want", Status: "posted",
	})
	require.ErrorIs(t, err, rates.ErrRateUnavailable)

	var count int
	require.NoError(t, pool.QueryRow(ctx, `SELECT count(*) FROM expenses WHERE user_id = $1`, user).Scan(&count))
	assert.Equal(t, 0, count)
}

// The saved record keeps the fake provider's fixed rate and PHP amount.
func TestCreate_StoresFixedRate_Integration(t *testing.T) {
	pool := testutil.SetupTestDB(t)
	ctx := context.Background()
	q := db.New(pool)
	user := uuid.New()

	created, err := NewServiceWithRates(q, tripRates{}).Create(ctx, user, CreateRequest{
		Amount: 100, Currency: "USD", Description: "gift", ExpenseDate: "2026-07-12",
		Priority: "want", Status: "posted",
	})
	require.NoError(t, err)
	assert.Equal(t, "USD", created.Currency)
	assert.True(t, decimal.RequireFromString("61.573").Equal(created.ExchangeRate))
	assert.True(t, decimal.RequireFromString("6157.30").Equal(created.AmountPhp))
	assert.Equal(t, "2026-07-10", created.RateDate.Time.Format("2006-01-02"))
}

type tripRates struct{}

func (tripRates) Rate(_ context.Context, _ string, date time.Time) (rates.Rate, error) {
	if date.Format("2006-01-02") == "2026-07-12" {
		return rates.Rate{Value: decimal.RequireFromString("61.573"), Date: date.AddDate(0, 0, -2)}, nil
	}
	return rates.Rate{}, rates.ErrRateUnavailable
}
