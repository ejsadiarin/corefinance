package materialize

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	db "github.com/ejsadiarin/corefinance/internal/db/sqlc"
)

// The user's timezone decides which date "today" is for their rules (#9).
// The gateway owns it (coregateway ADR-0005): API requests carry it in the
// internal token's tz claim, and the worker looks it up from the gateway.
// corefinance stores only the last zone it saw per user, to notice changes.

// Zones looks up a user's timezone. An error means the zone is unknown right
// now: the caller skips that user rather than guess (ADR-0005).
type Zones interface {
	Zone(ctx context.Context, userID uuid.UUID) (*time.Location, error)
}

// LoadZone is the IANA zone tz, or UTC when tz is empty or unknown (a token
// minted before the gateway sent tz, or a zone this build's tzdata lacks).
func LoadZone(tz string) *time.Location {
	if tz == "" {
		return time.UTC
	}
	loc, err := time.LoadLocation(tz)
	if err != nil {
		return time.UTC
	}
	return loc
}

// FixedZones is a Zones from a map of user to IANA zone; users not in it are
// on UTC. For tests and one-off runs.
type FixedZones map[uuid.UUID]string

func (z FixedZones) Zone(_ context.Context, userID uuid.UUID) (*time.Location, error) {
	return LoadZone(z[userID]), nil
}

// dateIn is the calendar date at now in loc, as UTC midnight (the form rule
// dates come back from Postgres in).
func dateIn(now time.Time, loc *time.Location) time.Time {
	y, m, d := now.In(loc).Date()
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}

// ObserveTimezone records the zone a request's token carried for the user,
// seen at now: when it differs from the last one corefinance saw, the user's
// next due times are reset in the same statement (#16). The common case, an
// unchanged zone, is a read and no write.
func ObserveTimezone(ctx context.Context, q db.Querier, userID uuid.UUID, loc *time.Location, now time.Time) error {
	seen, err := q.GetSeenTimezone(ctx, userID)
	if err == nil && seen == loc.String() {
		return nil
	}
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("get seen timezone: %w", err)
	}
	return recordZone(ctx, q, userID, loc, now)
}

// recordZone records loc as the user's zone checked at now, resetting their
// next due times if it changed.
func recordZone(ctx context.Context, q db.Querier, userID uuid.UUID, loc *time.Location, now time.Time) error {
	if _, err := q.RecordUserTimezone(ctx, db.RecordUserTimezoneParams{
		Owner: userID, Timezone: loc.String(), SeenAt: pgtype.Timestamptz{Time: now, Valid: true},
	}); err != nil {
		return fmt.Errorf("record user timezone: %w", err)
	}
	return nil
}

// ZoneRefreshEvery is how often the worker asks the gateway again for the
// zone of a user with active rules, so a change there reaches their schedule
// even if they make no request to corefinance.
const ZoneRefreshEvery = 10 * time.Minute

// zoneCheck is what refreshZones learned this pass: the zones it looked up,
// and the users whose lookup failed (skipped for the rest of the pass).
type zoneCheck struct {
	zones  map[uuid.UUID]*time.Location
	failed map[uuid.UUID]bool
}

// refreshZones asks zones again for every user with an active rule whose
// zone wasn't checked in the last ZoneRefreshEvery, and records each answer
// (resetting the schedule of users whose zone changed). A failed lookup is
// reported, and the user's schedule is left alone until a later pass.
func refreshZones(ctx context.Context, q db.Querier, zones Zones, now time.Time) (zoneCheck, error) {
	check := zoneCheck{zones: map[uuid.UUID]*time.Location{}, failed: map[uuid.UUID]bool{}}
	users, err := q.ListUsersToRefreshZone(ctx, pgtype.Timestamptz{Time: now.Add(-ZoneRefreshEvery), Valid: true})
	if err != nil {
		return check, fmt.Errorf("list users to refresh zone: %w", err)
	}
	var errs []error
	for _, user := range users {
		loc, err := zones.Zone(ctx, user)
		if err != nil {
			check.failed[user] = true
			errs = append(errs, fmt.Errorf("user %s: timezone: %w", user, err))
			continue
		}
		check.zones[user] = loc
		if err := recordZone(ctx, q, user, loc, now); err != nil {
			errs = append(errs, fmt.Errorf("user %s: %w", user, err))
		}
	}
	return check, errors.Join(errs...)
}

// zoneFor is the user's zone for this pass: the one refreshZones just looked
// up, or a fresh lookup. ok is false when the lookup failed (now or earlier
// in the pass); the error is reported once.
func (c zoneCheck) zoneFor(ctx context.Context, zones Zones, user uuid.UUID) (loc *time.Location, ok bool, err error) {
	if c.failed[user] {
		return nil, false, nil
	}
	if loc, ok := c.zones[user]; ok {
		return loc, true, nil
	}
	loc, err = zones.Zone(ctx, user)
	if err != nil {
		return nil, false, fmt.Errorf("user %s: timezone: %w", user, err)
	}
	return loc, true, nil
}
