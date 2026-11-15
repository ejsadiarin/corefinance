package materialize

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func d(s string) time.Time {
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		panic(err)
	}
	return t
}

func dates(ds []time.Time) []string {
	out := make([]string, 0, len(ds))
	for _, t := range ds {
		out = append(out, t.Format("2006-01-02"))
	}
	return out
}

func TestDueDates_Monthly_ClampSeries(t *testing.T) {
	got := dates(DueDates(d("2026-01-31"), "monthly", nil, d("2026-04-30")))
	assert.Equal(t, []string{"2026-01-31", "2026-02-28", "2026-03-31", "2026-04-30"}, got)
}

func TestDueDates_Yearly_Feb29Series(t *testing.T) {
	got := dates(DueDates(d("2024-02-29"), "yearly", nil, d("2028-03-01")))
	assert.Equal(t, []string{"2024-02-29", "2025-02-28", "2026-02-28", "2027-02-28", "2028-02-29"}, got)
}

func TestDueDates_BoundedByEnd(t *testing.T) {
	end := d("2026-09-02")
	got := dates(DueDates(d("2026-09-01"), "daily", &end, d("2026-09-10")))
	assert.Equal(t, []string{"2026-09-01", "2026-09-02"}, got)
}

func TestDueDates_WeeklySeries(t *testing.T) {
	got := dates(DueDates(d("2026-08-03"), "weekly", nil, d("2026-08-20")))
	assert.Equal(t, []string{"2026-08-03", "2026-08-10", "2026-08-17"}, got)
}

func TestDueDates_StartAfterToday(t *testing.T) {
	assert.Empty(t, DueDates(d("2026-09-11"), "daily", nil, d("2026-09-10")))
}

func TestDueDates_UnknownCadence(t *testing.T) {
	assert.Empty(t, DueDates(d("2026-01-01"), "hourly", nil, d("2026-09-10")))
}
