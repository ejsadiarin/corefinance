package materialize

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

func TestIsDue_Daily(t *testing.T) {
	assert.True(t, IsDue(d("2026-09-01"), "daily", d("2026-09-10")))
	assert.True(t, IsDue(d("2026-09-10"), "daily", d("2026-09-10")))
	assert.False(t, IsDue(d("2026-09-11"), "daily", d("2026-09-10")))
}

func TestIsDue_Weekly(t *testing.T) {
	start := d("2026-08-03") // Monday
	for _, due := range []string{"2026-08-03", "2026-08-10", "2026-08-17", "2026-08-24"} {
		assert.True(t, IsDue(start, "weekly", d(due)), due)
	}
	for _, notDue := range []string{"2026-08-04", "2026-08-09", "2026-08-11", "2026-08-02"} {
		assert.False(t, IsDue(start, "weekly", d(notDue)), notDue)
	}
}

func TestIsDue_Monthly_Jan31Clamps(t *testing.T) {
	start := d("2026-01-31")
	for _, due := range []string{"2026-01-31", "2026-02-28", "2026-03-31", "2026-04-30"} {
		assert.True(t, IsDue(start, "monthly", d(due)), due)
	}
	for _, notDue := range []string{"2026-02-27", "2026-02-01", "2026-03-30", "2026-01-30"} {
		assert.False(t, IsDue(start, "monthly", d(notDue)), notDue)
	}
}

func TestIsDue_Yearly_Feb29Clamps(t *testing.T) {
	start := d("2024-02-29")
	for _, due := range []string{"2024-02-29", "2025-02-28", "2026-02-28", "2027-02-28", "2028-02-29"} {
		assert.True(t, IsDue(start, "yearly", d(due)), due)
	}
	for _, notDue := range []string{"2025-02-27", "2025-03-01", "2026-02-27", "2028-02-28"} {
		assert.False(t, IsDue(start, "yearly", d(notDue)), notDue)
	}
}

func TestIsDue_UnknownCadence(t *testing.T) {
	assert.False(t, IsDue(d("2026-01-01"), "hourly", d("2026-09-10")))
	assert.False(t, IsDue(d("2026-01-01"), "", d("2026-09-10")))
	assert.False(t, IsDue(d("2026-01-01"), "one-time", d("2026-09-10")))
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

// IsDue(today) must agree with membership of today in the DueDates series:
// the daily pass and the backfill history must converge on the same grid.
func TestIsDue_ConsistentWithDueDates(t *testing.T) {
	starts := []time.Time{d("2024-02-29"), d("2026-01-31"), d("2026-08-03"), d("2026-09-10")}
	cadences := []string{"daily", "weekly", "monthly", "yearly"}
	anchor := d("2024-01-01")
	for _, start := range starts {
		for _, cad := range cadences {
			for i := 0; i < 800; i++ {
				today := anchor.AddDate(0, 0, i)
				series := DueDates(start, cad, nil, today)
				member := len(series) > 0 && series[len(series)-1].Equal(truncateDate(today))
				require.Equal(t, member, IsDue(start, cad, today),
					"cad=%s start=%s today=%s", cad, start.Format("2006-01-02"), today.Format("2006-01-02"))
			}
		}
	}
}
