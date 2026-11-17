package materialize

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	db "github.com/ejsadiarin/corefinance/internal/db/sqlc"
)

// The user's timezone decides which date "today" is for their rules (#9).
// The gateway owns it (coregateway ADR-0005) and sends it in the internal
// token's tz claim. corefinance records the last zone a token carried per
// user (user_timezones); background work, which has no token, uses that
// last-seen zone, and UTC for a user no token has carried one for yet.

// Zones gives a user's timezone. An error means it can't be read right now:
// the caller skips that user rather than guess.
type Zones interface {
	Zone(ctx context.Context, userID uuid.UUID) (*time.Location, error)
}

// LoadZone is the IANA zone tz, or UTC when tz is empty or unknown (a zone
// this build's tzdata lacks).
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

// SeenZones is the Zones the worker uses: each user's last-seen zone, UTC
// when none has been seen.
type SeenZones struct {
	Queries db.Querier
}

func (z SeenZones) Zone(ctx context.Context, userID uuid.UUID) (*time.Location, error) {
	tz, err := z.Queries.GetSeenTimezone(ctx, userID)
	if errors.Is(err, pgx.ErrNoRows) {
		return time.UTC, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get seen timezone: %w", err)
	}
	return LoadZone(tz), nil
}

// FixedZones is a Zones from a map of user to IANA zone; users not in it are
// on UTC. For tests.
type FixedZones map[uuid.UUID]string

func (z FixedZones) Zone(_ context.Context, userID uuid.UUID) (*time.Location, error) {
	return LoadZone(z[userID]), nil
}

// DateIn is the calendar date at now in loc, as UTC midnight (the form dates
// come back from Postgres in).
func DateIn(now time.Time, loc *time.Location) time.Time {
	y, m, d := now.In(loc).Date()
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}

// ObserveTimezone records the zone a request's token carried for the user,
// seen at now. The common case, an unchanged zone, is one read outside any
// transaction. A new zone is recorded in a short transaction that takes the
// user's materialize lock first, since recording it resets the user's next
// due times (#16): a writer that touches the user's rules always takes that
// lock before any rule row, so this can't deadlock with a rule save.
func ObserveTimezone(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID, loc *time.Location, now time.Time) error {
	seen, err := db.New(pool).GetSeenTimezone(ctx, userID)
	if err == nil && seen == loc.String() {
		return nil
	}
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("get seen timezone: %w", err)
	}
	tx, err := pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin record timezone: %w", err)
	}
	defer tx.Rollback(ctx)
	q := db.New(tx)
	if err := q.LockUserMaterialize(ctx, userID); err != nil {
		return fmt.Errorf("lock user materialize: %w", err)
	}
	if _, err := q.RecordUserTimezone(ctx, db.RecordUserTimezoneParams{
		Owner: userID, Timezone: loc.String(), SeenAt: pgtype.Timestamptz{Time: now, Valid: true},
	}); err != nil {
		return fmt.Errorf("record user timezone: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit record timezone: %w", err)
	}
	return nil
}
