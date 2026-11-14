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

// UserZone is the user's saved timezone, or now's own location (the server
// clock's zone) when they haven't saved one or it doesn't load.
func UserZone(ctx context.Context, q db.Querier, userID uuid.UUID, now time.Time) (*time.Location, error) {
	s, err := q.GetUserSettings(ctx, userID)
	if errors.Is(err, pgx.ErrNoRows) {
		return now.Location(), nil
	}
	if err != nil {
		return nil, fmt.Errorf("get user settings: %w", err)
	}
	loc, err := time.LoadLocation(s.Timezone)
	if err != nil || s.Timezone == "" {
		return now.Location(), nil
	}
	return loc, nil
}
