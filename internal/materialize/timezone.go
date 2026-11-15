package materialize

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
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
// seen at now. When it differs from the last one corefinance saw, the user's
// next due times are cleared: each is midnight in the old zone, and the next
// pass recomputes them (#16).
func ObserveTimezone(ctx context.Context, q db.Querier, userID uuid.UUID, tz string, now time.Time) error {
	changed, err := q.ObserveUserTimezone(ctx, db.ObserveUserTimezoneParams{
		UserID: userID, Timezone: tz, SeenAt: pgtype.Timestamptz{Time: now, Valid: true},
	})
	if err != nil {
		return fmt.Errorf("observe user timezone: %w", err)
	}
	if len(changed) == 0 {
		return nil
	}
	return clearNextDue(ctx, q, userID)
}

// ZoneRefreshEvery is how often the worker asks the gateway again for the
// zone of a user with active rules, so a change there reaches their schedule
// even if they make no request to corefinance.
const ZoneRefreshEvery = 10 * time.Minute

// refreshZones asks zones again for every user with an active rule whose
// zone wasn't checked in the last ZoneRefreshEvery, and clears the next due
// times of those whose zone changed. A failed lookup is reported and the
// user's schedule is left as it is until a later pass.
func refreshZones(ctx context.Context, q db.Querier, zones Zones, now time.Time) error {
	users, err := q.ListUsersToRefreshZone(ctx, pgtype.Timestamptz{Time: now.Add(-ZoneRefreshEvery), Valid: true})
	if err != nil {
		return fmt.Errorf("list users to refresh zone: %w", err)
	}
	var errs []error
	for _, user := range users {
		loc, err := zones.Zone(ctx, user)
		if err != nil {
			errs = append(errs, fmt.Errorf("user %s: timezone: %w", user, err))
			continue
		}
		prev, err := q.RefreshUserTimezone(ctx, db.RefreshUserTimezoneParams{
			UserID: user, Timezone: loc.String(), SeenAt: pgtype.Timestamptz{Time: now, Valid: true},
		})
		if err != nil {
			errs = append(errs, fmt.Errorf("user %s: refresh timezone: %w", user, err))
			continue
		}
		if prev != loc.String() {
			if err := clearNextDue(ctx, q, user); err != nil {
				errs = append(errs, fmt.Errorf("user %s: %w", user, err))
			}
		}
	}
	return errors.Join(errs...)
}

func clearNextDue(ctx context.Context, q db.Querier, userID uuid.UUID) error {
	if err := q.ClearUserNextDue(ctx, userID); err != nil {
		return fmt.Errorf("clear next due: %w", err)
	}
	return nil
}
