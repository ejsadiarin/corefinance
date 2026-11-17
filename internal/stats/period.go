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

// periodRange is the day, week (Monday to Sunday) or month containing anchor.
func periodRange(kind string, anchor time.Time) (dateRange, error) {
	anchor = dateOnly(anchor)
	switch kind {
	case "day":
		return dateRange{start: anchor, end: anchor}, nil
	case "week":
		offset := (int(anchor.Weekday()) + 6) % 7 // Monday is 0
		start := anchor.AddDate(0, 0, -offset)
		return dateRange{start: start, end: start.AddDate(0, 0, 6)}, nil
	case "month":
		start := time.Date(anchor.Year(), anchor.Month(), 1, 0, 0, 0, 0, time.UTC)
		return dateRange{start: start, end: start.AddDate(0, 1, -1)}, nil
	}
	return dateRange{}, fmt.Errorf("period must be day, week or month")
}

// previousRange is the period immediately before r.
func previousRange(kind string, r dateRange) dateRange {
	switch kind {
	case "day":
		day := r.start.AddDate(0, 0, -1)
		return dateRange{start: day, end: day}
	case "week":
		return dateRange{start: r.start.AddDate(0, 0, -7), end: r.start.AddDate(0, 0, -1)}
	default: // month
		start := time.Date(r.start.Year(), r.start.Month(), 1, 0, 0, 0, 0, time.UTC).AddDate(0, -1, 0)
		return dateRange{start: start, end: r.start.AddDate(0, 0, -1)}
	}
}

// daysInRange counts both ends of a closed range.
func daysInRange(r dateRange) int {
	return int(r.end.Sub(r.start).Hours()/24) + 1
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
