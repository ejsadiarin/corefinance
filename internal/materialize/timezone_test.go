package materialize

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	db "github.com/ejsadiarin/corefinance/internal/db/sqlc"
)

// The user's "today" (#9): rule entries land on the user's date, not the
// server clock's.

// Monday 16:30 UTC is Tuesday 00:30 in Manila.
var mondayEveningUTC = time.Date(2026, 11, 9, 16, 30, 0, 0, time.UTC)

// testZones is the gateway as the tests see it: users not in it are on UTC.
var testZones = FixedZones{}

func setTimezone(t *testing.T, _ context.Context, _ *db.Queries, userID uuid.UUID, tz string) {
	t.Helper()
	testZones[userID] = tz
}

func TestCatchUp_ManilaUserGetsTuesdayEntryAtMondayEveningUTC_Integration(t *testing.T) {
	pool, q := newMaterializeTest(t)
	ctx := context.Background()
	manila := uuid.New()
	setTimezone(t, ctx, q, manila, "Asia/Manila")
	rule := seedExpenseRule(t, ctx, q, manila, "weekly", "2026-11-10", nil) // a Tuesday

	n, err := MaterializeDue(ctx, pool, q, testZones, mondayEveningUTC)
	require.NoError(t, err)
	assert.Equal(t, 1, n)
	assert.Equal(t, []string{"2026-11-10"}, ruleExpenseDates(t, ctx, pool, rule.ID))
}

func TestCatchUp_UserWithoutTimezoneKeepsServerDate_Integration(t *testing.T) {
	pool, q := newMaterializeTest(t)
	ctx := context.Background()
	rule := seedExpenseRule(t, ctx, q, uuid.New(), "weekly", "2026-11-10", nil)

	n, err := MaterializeDue(ctx, pool, q, testZones, mondayEveningUTC)
	require.NoError(t, err)
	assert.Equal(t, 0, n)
	assert.Empty(t, ruleExpenseDates(t, ctx, pool, rule.ID))
}

// Resuming a paused rule: the watermark is the day before the owner's date,
// so the catch-up pass fills the owner's today onward.
func TestResumeThrough_UsesOwnersDate(t *testing.T) {
	manila, err := time.LoadLocation("Asia/Manila")
	require.NoError(t, err)
	assert.Equal(t, "2026-11-09", ResumeThrough(manila, mondayEveningUTC).Time.Format(time.DateOnly))
	assert.Equal(t, "2026-11-08", ResumeThrough(time.UTC, mondayEveningUTC).Time.Format(time.DateOnly))
}

// failingZones fails the lookup for one user, as a gateway outage would.
type failingZones struct{ down uuid.UUID }

func (z failingZones) Zone(ctx context.Context, user uuid.UUID) (*time.Location, error) {
	if user == z.down {
		return nil, errors.New("gateway: 502 Bad Gateway")
	}
	return time.UTC, nil
}

// A failed timezone lookup skips that user's pass rather than guess a zone;
// everyone else is still filled, and the skipped user catches up next time.
func TestCatchUp_FailedZoneLookupSkipsOnlyThatUser_Integration(t *testing.T) {
	pool, q := newMaterializeTest(t)
	ctx := context.Background()
	down, up := uuid.New(), uuid.New()
	downRule := seedExpenseRule(t, ctx, q, down, "daily", "2026-11-08", nil)
	upRule := seedExpenseRule(t, ctx, q, up, "daily", "2026-11-08", nil)

	_, err := MaterializeDue(ctx, pool, q, failingZones{down: down}, mondayEveningUTC)
	require.Error(t, err)
	assert.Empty(t, ruleExpenseDates(t, ctx, pool, downRule.ID), "no guessed zone")
	assert.Equal(t, []string{"2026-11-08", "2026-11-09"}, ruleExpenseDates(t, ctx, pool, upRule.ID))

	_, err = MaterializeDue(ctx, pool, q, FixedZones{}, mondayEveningUTC)
	require.NoError(t, err)
	assert.Equal(t, []string{"2026-11-08", "2026-11-09"}, ruleExpenseDates(t, ctx, pool, downRule.ID))
}
