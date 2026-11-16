package rates

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func day(s string) time.Time {
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		panic(err)
	}
	return t
}

// testProvider is a minimal provider for Convert's unit tests.
type testProvider struct {
	rate        Rate
	err         error
	supportsErr error
	calls       int
}

func (p *testProvider) Rate(context.Context, string, time.Time) (Rate, error) {
	p.calls++
	return p.rate, p.err
}

func (p *testProvider) Supports(context.Context, string) error {
	return p.supportsErr
}

func TestNormalize(t *testing.T) {
	assert.Equal(t, "PHP", Normalize(""))
	assert.Equal(t, "USD", Normalize(" usd "))
}

func TestConvertRoundsPHPAmount(t *testing.T) {
	p := &testProvider{rate: Rate{Value: decimal.RequireFromString("61.573"), Date: day("2026-07-10")}}
	got, err := Convert(context.Background(), p, "usd", day("2026-07-12"), decimal.NewFromInt(100))
	require.NoError(t, err)
	assert.Equal(t, "USD", got.Currency)
	assert.True(t, decimal.RequireFromString("6157.30").Equal(got.AmountPHP), got.AmountPHP.String())
	assert.True(t, decimal.RequireFromString("61.573").Equal(got.Rate))
	assert.True(t, got.RateDate.Equal(day("2026-07-10")))
	assert.Equal(t, 1, p.calls)
}

func TestSupportsSkipsPHPAndUsesTheChecker(t *testing.T) {
	p := &testProvider{supportsErr: ErrUnsupportedCurrency}
	require.NoError(t, Supports(context.Background(), p, "php"))
	require.ErrorIs(t, Supports(context.Background(), p, "USD"), ErrUnsupportedCurrency)
}

func TestConvertForUpdateReusesTheFixedRate(t *testing.T) {
	p := &testProvider{err: errors.New("must not fetch")}
	existing := Existing{
		Currency: "USD",
		Date:     day("2026-07-12"),
		Rate:     decimal.RequireFromString("61.573"),
		RateDate: day("2026-07-10"),
	}
	got, err := ConvertForUpdate(context.Background(), p, existing, "USD", day("2026-07-12"), decimal.NewFromInt(200))
	require.NoError(t, err)
	assert.Equal(t, 0, p.calls)
	assert.True(t, decimal.RequireFromString("12314.60").Equal(got.AmountPHP))
	assert.True(t, got.RateDate.Equal(day("2026-07-10")))
}

func TestConvertForUpdateFetchesAfterACurrencyOrDateChange(t *testing.T) {
	p := &testProvider{rate: Rate{Value: decimal.RequireFromString("62.1"), Date: day("2026-07-13")}}
	existing := Existing{
		Currency: "USD",
		Date:     day("2026-07-12"),
		Rate:     decimal.RequireFromString("61.573"),
		RateDate: day("2026-07-10"),
	}
	_, err := ConvertForUpdate(context.Background(), p, existing, "USD", day("2026-07-13"), decimal.NewFromInt(100))
	require.NoError(t, err)
	assert.Equal(t, 1, p.calls)
}

func TestConvertPHPDoesNotCallProvider(t *testing.T) {
	p := &testProvider{err: errors.New("must not be called")}
	got, err := Convert(context.Background(), p, "php", day("2026-07-12"), decimal.NewFromInt(100))
	require.NoError(t, err)
	assert.True(t, decimal.NewFromInt(1).Equal(got.Rate))
	assert.True(t, decimal.NewFromInt(100).Equal(got.AmountPHP))
	assert.True(t, got.RateDate.Equal(day("2026-07-12")))
	assert.Equal(t, 0, p.calls)
}

func TestFrankfurterCachesPublishedPastRates(t *testing.T) {
	var rateCalls, currencyCalls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/currencies":
			atomic.AddInt32(&currencyCalls, 1)
			_, _ = w.Write([]byte(`{"USD":"United States Dollar","PHP":"Philippine Peso"}`))
		case "/2020-01-02":
			atomic.AddInt32(&rateCalls, 1)
			_, _ = w.Write([]byte(`{"amount":1,"base":"USD","date":"2020-01-02","rates":{"PHP":50.5}}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	c := NewFrankfurter(srv.URL, srv.Client())
	for i := 0; i < 2; i++ {
		got, err := c.Rate(context.Background(), "usd", day("2020-01-02"))
		require.NoError(t, err)
		assert.True(t, decimal.RequireFromString("50.5").Equal(got.Value))
		assert.True(t, got.Date.Equal(day("2020-01-02")))
	}
	assert.Equal(t, int32(1), atomic.LoadInt32(&rateCalls))
	assert.Equal(t, int32(1), atomic.LoadInt32(&currencyCalls))
}

func TestFrankfurterWeekendReturnsPublishedBusinessDay(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/currencies":
			_, _ = w.Write([]byte(`{"USD":"United States Dollar","PHP":"Philippine Peso"}`))
		case "/2026-07-12":
			_, _ = w.Write([]byte(`{"amount":1,"base":"USD","date":"2026-07-10","rates":{"PHP":61.573}}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	got, err := NewFrankfurter(srv.URL, srv.Client()).Rate(context.Background(), "USD", day("2026-07-12"))
	require.NoError(t, err)
	assert.True(t, got.Date.Equal(day("2026-07-10")))
}

func TestFrankfurterDoesNotCacheFutureLookups(t *testing.T) {
	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/currencies":
			_, _ = w.Write([]byte(`{"USD":"United States Dollar","PHP":"Philippine Peso"}`))
		default:
			atomic.AddInt32(&calls, 1)
			_, _ = w.Write([]byte(`{"amount":1,"base":"USD","date":"2026-11-13","rates":{"PHP":63.813}}`))
		}
	}))
	defer srv.Close()

	c := NewFrankfurter(srv.URL, srv.Client())
	future := time.Now().AddDate(0, 0, 7)
	for i := 0; i < 2; i++ {
		_, err := c.Rate(context.Background(), "USD", future)
		require.NoError(t, err)
	}
	assert.Equal(t, int32(2), atomic.LoadInt32(&calls))
}

func TestFrankfurterRejectsUnsupportedCurrency(t *testing.T) {
	var rateCalls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/currencies" {
			_, _ = w.Write([]byte(`{"USD":"United States Dollar","PHP":"Philippine Peso"}`))
			return
		}
		atomic.AddInt32(&rateCalls, 1)
		http.NotFound(w, r)
	}))
	defer srv.Close()

	_, err := NewFrankfurter(srv.URL, srv.Client()).Rate(context.Background(), "EUR", day("2026-07-12"))
	require.ErrorIs(t, err, ErrUnsupportedCurrency)
	assert.Equal(t, int32(0), atomic.LoadInt32(&rateCalls))
}

func TestFrankfurterProviderFailure(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "down", http.StatusInternalServerError)
	}))
	defer srv.Close()

	_, err := NewFrankfurter(srv.URL, srv.Client()).Rate(context.Background(), "USD", day("2026-07-12"))
	require.ErrorIs(t, err, ErrRateUnavailable)
}

func TestPHPOnlyProvider(t *testing.T) {
	got, err := PHPOnly{}.Rate(context.Background(), "PHP", day("2026-07-12"))
	require.NoError(t, err)
	assert.True(t, decimal.NewFromInt(1).Equal(got.Value))

	_, err = PHPOnly{}.Rate(context.Background(), "USD", day("2026-07-12"))
	require.ErrorIs(t, err, ErrUnsupportedCurrency)
}

// The cache is safe to use from concurrent saves.
func TestFrankfurterCacheConcurrent(t *testing.T) {
	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/currencies":
			_, _ = w.Write([]byte(`{"USD":"United States Dollar","PHP":"Philippine Peso"}`))
		default:
			atomic.AddInt32(&calls, 1)
			time.Sleep(5 * time.Millisecond)
			_, _ = w.Write([]byte(`{"amount":1,"base":"USD","date":"2020-01-02","rates":{"PHP":50.5}}`))
		}
	}))
	defer srv.Close()

	c := NewFrankfurter(srv.URL, srv.Client())
	var wg sync.WaitGroup
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := c.Rate(context.Background(), "USD", day("2020-01-02"))
			assert.NoError(t, err)
		}()
	}
	wg.Wait()
	assert.Equal(t, int32(1), atomic.LoadInt32(&calls))
}
