// Package occurred turns the time an owner gives a budget record into the
// moment it happened (#28). The record's day stays the budget's; the time,
// when given, is the owner's wall clock on that day, stored as UTC.
package occurred

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/ejsadiarin/corefinance/internal/auth"
	db "github.com/ejsadiarin/corefinance/internal/db/sqlc"
	"github.com/ejsadiarin/corefinance/internal/materialize"
)

// ErrInvalidTime is a time that isn't HH:mm on a 24-hour clock.
var ErrInvalidTime = errors.New("time must be HH:mm (24-hour)")

// ValidateTime accepts no time or an HH:mm one.
func ValidateTime(clock *string) error {
	if clock == nil || *clock == "" {
		return nil
	}
	if _, err := time.Parse("15:04", *clock); err != nil {
		return ErrInvalidTime
	}
	return nil
}

// At is the moment date (YYYY-MM-DD) at clock (HH:mm) in loc; null when no
// time is given.
func At(date string, clock *string, loc *time.Location) (pgtype.Timestamptz, error) {
	if clock == nil || *clock == "" {
		return pgtype.Timestamptz{}, nil
	}
	t, err := time.ParseInLocation("2006-01-02 15:04", date+" "+*clock, loc)
	if err != nil {
		if ValidateTime(clock) != nil {
			return pgtype.Timestamptz{}, ErrInvalidTime
		}
		return pgtype.Timestamptz{}, fmt.Errorf("record date: %w", err)
	}
	return pgtype.Timestamptz{Time: t, Valid: true}, nil
}

// OwnerZone is the zone the owner's times are read in (coregateway
// ADR-0005): the request token's, or the one last seen for the owner, UTC
// if none.
func OwnerZone(ctx context.Context, q db.Querier, owner uuid.UUID) (*time.Location, error) {
	if tz := auth.TimezoneFromContext(ctx); tz != "" {
		return materialize.LoadZone(tz), nil
	}
	loc, err := materialize.SeenZones{Queries: q}.Zone(ctx, owner)
	if err != nil {
		return nil, fmt.Errorf("owner timezone: %w", err)
	}
	return loc, nil
}

// For is the moment a record with date and clock happened for owner, read
// in the owner's zone; null when no time is given (no zone lookup then).
func For(ctx context.Context, q db.Querier, owner uuid.UUID, date string, clock *string) (pgtype.Timestamptz, error) {
	if clock == nil || *clock == "" {
		return pgtype.Timestamptz{}, nil
	}
	if err := ValidateTime(clock); err != nil {
		return pgtype.Timestamptz{}, err
	}
	loc, err := OwnerZone(ctx, q, owner)
	if err != nil {
		return pgtype.Timestamptz{}, err
	}
	return At(date, clock, loc)
}
