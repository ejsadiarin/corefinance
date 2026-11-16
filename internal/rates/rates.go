// Package rates turns an amount in one currency into the PHP amount reports
// use, at a rate fixed for its date (ADR-0003).
package rates

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/shopspring/decimal"
	"golang.org/x/sync/singleflight"
)

// DefaultURL is the ECB-backed provider.
const DefaultURL = "https://api.frankfurter.dev/v1"

// FromEnv builds the production provider from RATE_URL.
func FromEnv() *Frankfurter {
	base := os.Getenv("RATE_URL")
	if base == "" {
		base = DefaultURL
	}
	return NewFrankfurter(base, nil)
}

var (
	// ErrUnsupportedCurrency means the provider does not publish the code.
	ErrUnsupportedCurrency = errors.New("unsupported currency")
	// ErrRateUnavailable means the provider could not answer.
	ErrRateUnavailable = errors.New("exchange rate unavailable")
)

// Rate is the PHP value of one unit of a currency on Date.
type Rate struct {
	Value decimal.Decimal
	Date  time.Time
}

// Provider answers the PHP rate for a currency on a date.
type Provider interface {
	Rate(ctx context.Context, currency string, date time.Time) (Rate, error)
}

// CurrencyChecker is an optional provider capability: validate a code without
// fetching a rate. A provider that cannot check skips the check; the write
// still fails when it needs the rate.
type CurrencyChecker interface {
	Supports(ctx context.Context, currency string) error
}

// Supports validates a currency when the provider knows how. PHP always
// passes without a call.
func Supports(ctx context.Context, provider Provider, currency string) error {
	c := Normalize(currency)
	if c == "PHP" {
		return nil
	}
	if checker, ok := provider.(CurrencyChecker); ok {
		return checker.Supports(ctx, c)
	}
	return nil
}

// PHPOnly is the safe default: it answers PHP at 1 and refuses every other
// currency rather than guess.
type PHPOnly struct{}

func (PHPOnly) Rate(_ context.Context, currency string, date time.Time) (Rate, error) {
	if Normalize(currency) != "PHP" {
		return Rate{}, fmt.Errorf("%w: %s", ErrUnsupportedCurrency, currency)
	}
	return Rate{Value: decimal.NewFromInt(1), Date: DateOnly(date)}, nil
}

func (PHPOnly) Supports(_ context.Context, currency string) error {
	if Normalize(currency) != "PHP" {
		return fmt.Errorf("%w: %s", ErrUnsupportedCurrency, currency)
	}
	return nil
}

// Normalize is the canonical currency code: trimmed, uppercase, PHP when
// omitted.
func Normalize(currency string) string {
	c := strings.ToUpper(strings.TrimSpace(currency))
	if c == "" {
		return "PHP"
	}
	return c
}

// DateOnly drops the time of day, keeping the calendar date in UTC.
func DateOnly(t time.Time) time.Time {
	y, m, d := t.UTC().Date()
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}

// Conversion is what a budget record stores: the original currency, the rate,
// the PHP amount rounded to centavos, and the rate's published date.
type Conversion struct {
	Currency  string
	Rate      decimal.Decimal
	AmountPHP decimal.Decimal
	RateDate  time.Time
}

// Existing is a saved record's conversion, used to keep its fixed rate on an
// edit.
type Existing struct {
	Currency string
	Date     time.Time
	Rate     decimal.Decimal
	RateDate time.Time
}

// ConvertForUpdate keeps the saved rate unless the currency or date moved
// (ADR-0003). An amount-only edit reuses it; other field edits never reach
// this function.
func ConvertForUpdate(ctx context.Context, provider Provider, existing Existing, currency string, date time.Time, amount decimal.Decimal) (Conversion, error) {
	c := Normalize(currency)
	if c != "PHP" &&
		c == Normalize(existing.Currency) &&
		DateOnly(existing.Date).Equal(DateOnly(date)) &&
		existing.Rate.IsPositive() {
		rateDate := DateOnly(date)
		if !existing.RateDate.IsZero() {
			rateDate = DateOnly(existing.RateDate)
		}
		return Conversion{
			Currency:  c,
			Rate:      existing.Rate,
			AmountPHP: amount.Mul(existing.Rate).Round(2),
			RateDate:  rateDate,
		}, nil
	}
	return Convert(ctx, provider, c, date, amount)
}

// Convert fixes the rate and PHP amount for an amount on a date. PHP needs no
// provider call.
func Convert(ctx context.Context, provider Provider, currency string, date time.Time, amount decimal.Decimal) (Conversion, error) {
	c := Normalize(currency)
	d := DateOnly(date)
	if c == "PHP" {
		return Conversion{
			Currency:  c,
			Rate:      decimal.NewFromInt(1),
			AmountPHP: amount.Round(2),
			RateDate:  d,
		}, nil
	}
	if provider == nil {
		return Conversion{}, fmt.Errorf("%w: no rate provider", ErrRateUnavailable)
	}
	r, err := provider.Rate(ctx, c, d)
	if err != nil {
		return Conversion{}, err
	}
	if !r.Value.IsPositive() {
		return Conversion{}, fmt.Errorf("%w: %s returned a non-positive rate", ErrRateUnavailable, c)
	}
	return Conversion{
		Currency:  c,
		Rate:      r.Value,
		AmountPHP: amount.Mul(r.Value).Round(2),
		RateDate:  DateOnly(r.Date),
	}, nil
}

// Frankfurter talks to frankfurter.dev's ECB-backed v1 API.
type Frankfurter struct {
	baseURL string
	client  *http.Client

	mu           sync.Mutex
	rates        map[string]Rate
	currencies   map[string]struct{}
	currenciesAt time.Time
	group        singleflight.Group
}

const (
	currencyTTL = 24 * time.Hour
	// requestTimeout keeps a slow provider from holding a save open.
	requestTimeout = 10 * time.Second
)

// NewFrankfurter builds a client. A nil HTTP client gets a 10s timeout.
func NewFrankfurter(baseURL string, client *http.Client) *Frankfurter {
	if client == nil {
		client = &http.Client{Timeout: requestTimeout}
	}
	return &Frankfurter{
		baseURL: strings.TrimRight(baseURL, "/"),
		client:  client,
		rates:   map[string]Rate{},
	}
}

// Supports validates a currency against the provider's published list.
func (f *Frankfurter) Supports(ctx context.Context, currency string) error {
	return f.ensureSupported(ctx, Normalize(currency))
}

// Rate returns the PHP rate for currency on date. The provider answers a
// weekend or holiday with the previous published business day, and a date it
// has not reached yet with its latest published rate; both are returned with
// their actual date. Past lookups are cached; today's and future ones are
// not, so a newly published rate is not hidden by yesterday's answer.
func (f *Frankfurter) Rate(ctx context.Context, currency string, date time.Time) (Rate, error) {
	c := Normalize(currency)
	d := DateOnly(date)
	if c == "PHP" {
		return Rate{Value: decimal.NewFromInt(1), Date: d}, nil
	}
	if err := f.ensureSupported(ctx, c); err != nil {
		return Rate{}, err
	}

	today := DateOnly(time.Now())
	key := c + ":" + d.Format("2006-01-02")
	cacheable := d.Before(today)
	if cacheable {
		f.mu.Lock()
		cached, ok := f.rates[key]
		f.mu.Unlock()
		if ok {
			return cached, nil
		}
		value, err, _ := f.group.Do(key, func() (any, error) {
			f.mu.Lock()
			if cached, ok := f.rates[key]; ok {
				f.mu.Unlock()
				return cached, nil
			}
			f.mu.Unlock()
			got, err := f.fetchRate(ctx, c, d)
			if err != nil {
				return Rate{}, err
			}
			f.mu.Lock()
			f.rates[key] = got
			f.mu.Unlock()
			return got, nil
		})
		if err != nil {
			return Rate{}, err
		}
		return value.(Rate), nil
	}
	return f.fetchRate(ctx, c, d)
}

func (f *Frankfurter) fetchRate(ctx context.Context, currency string, date time.Time) (Rate, error) {
	var payload struct {
		Date  string                 `json:"date"`
		Rates map[string]json.Number `json:"rates"`
	}
	path := fmt.Sprintf("%s/%s?base=%s&symbols=PHP", f.baseURL, date.Format("2006-01-02"), currency)
	if err := f.get(ctx, path, &payload); err != nil {
		return Rate{}, err
	}
	raw, ok := payload.Rates["PHP"]
	if !ok {
		return Rate{}, fmt.Errorf("%w: %s has no PHP rate", ErrRateUnavailable, currency)
	}
	value, err := decimal.NewFromString(raw.String())
	if err != nil || !value.IsPositive() {
		return Rate{}, fmt.Errorf("%w: invalid PHP rate for %s", ErrRateUnavailable, currency)
	}
	rateDate, err := time.Parse("2006-01-02", payload.Date)
	if err != nil {
		return Rate{}, fmt.Errorf("%w: invalid rate date for %s", ErrRateUnavailable, currency)
	}
	return Rate{Value: value, Date: DateOnly(rateDate)}, nil
}

func (f *Frankfurter) ensureSupported(ctx context.Context, currency string) error {
	f.mu.Lock()
	if f.currencies != nil && time.Since(f.currenciesAt) < currencyTTL {
		_, ok := f.currencies[currency]
		f.mu.Unlock()
		if !ok {
			return fmt.Errorf("%w: %s", ErrUnsupportedCurrency, currency)
		}
		return nil
	}
	cached := f.currencies
	f.mu.Unlock()

	var payload map[string]string
	if err := f.get(ctx, f.baseURL+"/currencies", &payload); err != nil {
		if cached != nil {
			// The rate endpoint may still answer; don't block on a stale list.
			_, ok := cached[currency]
			if !ok {
				return fmt.Errorf("%w: %s", ErrUnsupportedCurrency, currency)
			}
			return nil
		}
		return err
	}
	next := make(map[string]struct{}, len(payload))
	for code := range payload {
		next[Normalize(code)] = struct{}{}
	}
	f.mu.Lock()
	f.currencies = next
	f.currenciesAt = time.Now()
	f.mu.Unlock()

	if _, ok := next[currency]; !ok {
		return fmt.Errorf("%w: %s", ErrUnsupportedCurrency, currency)
	}
	return nil
}

func (f *Frankfurter) get(ctx context.Context, url string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrRateUnavailable, err)
	}
	resp, err := f.client.Do(req)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrRateUnavailable, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%w: provider returned %d", ErrRateUnavailable, resp.StatusCode)
	}
	dec := json.NewDecoder(resp.Body)
	dec.UseNumber()
	if err := dec.Decode(out); err != nil {
		return fmt.Errorf("%w: %v", ErrRateUnavailable, err)
	}
	return nil
}
