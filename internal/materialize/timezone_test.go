package materialize

import (
	"context"
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

func setTimezone(t *testing.T, ctx context.Context, q *db.Queries, userID uuid.UUID, tz string) {
	t.Helper()
	_, err := q.PutUserSettings(ctx, db.PutUserSettingsParams{UserID: userID, Timezone: tz})
	require.NoError(t, err)
}

func TestCatchUp_ManilaUserGetsTuesdayEntryAtMondayEveningUTC_Integration(t *testing.T) {
	pool, q := newMaterializeTest(t)
	ctx := context.Background()
	manila := uuid.New()
	setTimezone(t, ctx, q, manila, "Asia/Manila")
	rule := seedExpenseRule(t, ctx, q, manila, "weekly", "2026-11-10", nil) // a Tuesday

	n, err := MaterializeDue(ctx, pool, q, mondayEveningUTC)
	require.NoError(t, err)
	assert.Equal(t, 1, n)
	assert.Equal(t, []string{"2026-11-10"}, ruleExpenseDates(t, ctx, pool, rule.ID))
}

func TestCatchUp_UserWithoutTimezoneKeepsServerDate_Integration(t *testing.T) {
	pool, q := newMaterializeTest(t)
	ctx := context.Background()
	rule := seedExpenseRule(t, ctx, q, uuid.New(), "weekly", "2026-11-10", nil)

	n, err := MaterializeDue(ctx, pool, q, mondayEveningUTC)
	require.NoError(t, err)
	assert.Equal(t, 0, n)
	assert.Empty(t, ruleExpenseDates(t, ctx, pool, rule.ID))
}

// Resuming a paused rule: the watermark is the day before the user's date,
// so the catch-up pass fills the user's today onward.
func TestResumeThrough_UsesUserDate_Integration(t *testing.T) {
	_, q := newMaterializeTest(t)
	ctx := context.Background()
	manila := uuid.New()
	setTimezone(t, ctx, q, manila, "Asia/Manila")

	got, err := ResumeThrough(ctx, q, manila, mondayEveningUTC)
	require.NoError(t, err)
	assert.Equal(t, "2026-11-09", got.Time.Format("2006-01-02"))

	got, err = ResumeThrough(ctx, q, uuid.New(), mondayEveningUTC)
	require.NoError(t, err)
	assert.Equal(t, "2026-11-08", got.Time.Format("2006-01-02"))
}
