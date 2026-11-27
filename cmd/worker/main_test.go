package main

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	db "github.com/ejsadiarin/corefinance/internal/db/sqlc"
)

// wakeQuerier answers only GetNextWake; any other call panics on the nil
// embedded Querier, which keeps the test honest about what it touches.
type wakeQuerier struct {
	db.Querier
	wake pgtype.Timestamptz
	err  error
}

func (q wakeQuerier) GetNextWake(context.Context, pgtype.Timestamptz) (pgtype.Timestamptz, error) {
	return q.wake, q.err
}

func TestUntilNextWake(t *testing.T) {
	now := time.Date(2026, 11, 27, 9, 0, 0, 0, time.UTC)
	at := func(d time.Duration) pgtype.Timestamptz {
		return pgtype.Timestamptz{Time: now.Add(d), Valid: true}
	}

	cases := []struct {
		name   string
		q      wakeQuerier
		failed bool // the pass before skipped some users
		want   time.Duration
	}{
		{"nothing due sleeps 6 hours", wakeQuerier{}, false, 6 * time.Hour},
		{"a rule due in 3 minutes wakes on time", wakeQuerier{wake: at(3 * time.Minute)}, false, 3 * time.Minute},
		{"a rule due in an hour wakes on time", wakeQuerier{wake: at(time.Hour)}, false, time.Hour},
		{"a rule due in a week still checks after 6 hours", wakeQuerier{wake: at(7 * 24 * time.Hour)}, false, 6 * time.Hour},
		{"an overdue rule waits at least a second", wakeQuerier{wake: at(-time.Minute)}, false, time.Second},
		{"a failed lookup retries after 15 minutes", wakeQuerier{err: errors.New("boom")}, false, 15 * time.Minute},
		{"a pass that skipped users retries after 15 minutes", wakeQuerier{wake: at(7 * 24 * time.Hour)}, true, 15 * time.Minute},
		{"a pass that skipped users still wakes for a rule due sooner", wakeQuerier{wake: at(3 * time.Minute)}, true, 3 * time.Minute},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := untilNextWake(context.Background(), tc.q, now, tc.failed); got != tc.want {
				t.Fatalf("untilNextWake = %v, want %v", got, tc.want)
			}
		})
	}
}
