package stats

import (
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

// dateRange is a closed range of calendar dates, both at UTC midnight.
type dateRange struct {
	start, end time.Time
}

// periodRanges is the day, week (Monday to Sunday) or month containing
// anchor and the period immediately before it.
func periodRanges(kind string, anchor time.Time) (current, previous dateRange, err error) {
	anchor = dateOnly(anchor)
	switch kind {
	case "day":
		previousDay := anchor.AddDate(0, 0, -1)
		return dateRange{start: anchor, end: anchor},
			dateRange{start: previousDay, end: previousDay}, nil
	case "week":
		offset := (int(anchor.Weekday()) + 6) % 7 // Monday is 0
		start := anchor.AddDate(0, 0, -offset)
		return dateRange{start: start, end: start.AddDate(0, 0, 6)},
			dateRange{start: start.AddDate(0, 0, -7), end: start.AddDate(0, 0, -1)}, nil
	case "month":
		start := time.Date(anchor.Year(), anchor.Month(), 1, 0, 0, 0, 0, time.UTC)
		previousStart := start.AddDate(0, -1, 0)
		return dateRange{start: start, end: start.AddDate(0, 1, -1)},
			dateRange{start: previousStart, end: start.AddDate(0, 0, -1)}, nil
	}
	return dateRange{}, dateRange{}, fmt.Errorf("period must be day, week or month")
}

// daysInRange counts both ends of a closed range.
func daysInRange(r dateRange) int {
	return daysBetween(r.start, r.end)
}

// daysBetween counts both ends of the range from through to.
func daysBetween(from, to time.Time) int {
	return int(dateOnly(to).Sub(dateOnly(from)).Hours()/24) + 1
}

func dateOnly(t time.Time) time.Time {
	y, m, d := t.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}

func formatDate(t time.Time) string {
	return t.Format("2006-01-02")
}

func pgDate(t time.Time) pgtype.Date {
	return pgtype.Date{Time: t, Valid: true}
}
