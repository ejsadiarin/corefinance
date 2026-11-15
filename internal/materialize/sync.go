package materialize

import (
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

// ResumeThrough is the watermark a rule gets when it is resumed from paused
// at now, for an owner in loc: the day before the owner's date, so the
// catch-up pass fills today onward and not the stretch the rule was paused.
// Pass it as ResumeThrough on every rule update; the update query only
// applies it when is_active flips false->true.
func ResumeThrough(loc *time.Location, now time.Time) pgtype.Date {
	return pgDate(dateIn(now, loc).AddDate(0, 0, -1))
}
