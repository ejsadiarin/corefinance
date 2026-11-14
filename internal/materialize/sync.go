package materialize

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	db "github.com/ejsadiarin/corefinance/internal/db/sqlc"
)

// ResumeThrough is the watermark a user's rule gets when it is resumed from
// paused at now: the day before the user's date, so the catch-up pass fills
// today onward and not the stretch the rule was paused. Pass it as
// ResumeThrough on every rule update; the update query only applies it when
// is_active flips false->true.
func ResumeThrough(ctx context.Context, q db.Querier, userID uuid.UUID, now time.Time) (pgtype.Date, error) {
	today, err := UserToday(ctx, q, userID, now)
	if err != nil {
		return pgtype.Date{}, err
	}
	return pgDate(today.AddDate(0, 0, -1)), nil
}
