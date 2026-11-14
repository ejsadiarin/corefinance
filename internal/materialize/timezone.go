package materialize

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	db "github.com/ejsadiarin/corefinance/internal/db/sqlc"
)

// latestZone is the furthest-ahead UTC offset in use (Pacific/Kiritimati):
// no user's date is later than now's date here, so it bounds which rules a
// pass needs to look at before each user's own date is known.
var latestZone = time.FixedZone("UTC+14", 14*60*60)

// userDate is the calendar date at now in the IANA zone tz, as UTC midnight
// (the form rule dates come back from Postgres in). An empty or unknown zone
// keeps the server clock's date.
func userDate(now time.Time, tz string) time.Time {
	if tz != "" {
		if loc, err := time.LoadLocation(tz); err == nil {
			now = now.In(loc)
		}
	}
	y, m, d := now.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}

// UserToday is the user's date at now: in their saved timezone, or the
// server clock's date when they haven't saved one.
func UserToday(ctx context.Context, q db.Querier, userID uuid.UUID, now time.Time) (time.Time, error) {
	s, err := q.GetUserSettings(ctx, userID)
	if errors.Is(err, pgx.ErrNoRows) {
		return userDate(now, ""), nil
	}
	if err != nil {
		return time.Time{}, fmt.Errorf("get user settings: %w", err)
	}
	return userDate(now, s.Timezone), nil
}

// userTimezones maps every user who saved a timezone to it.
func userTimezones(ctx context.Context, q db.Querier) (map[uuid.UUID]string, error) {
	rows, err := q.ListUserTimezones(ctx)
	if err != nil {
		return nil, fmt.Errorf("list user timezones: %w", err)
	}
	out := make(map[uuid.UUID]string, len(rows))
	for _, r := range rows {
		out[r.UserID] = r.Timezone
	}
	return out, nil
}
