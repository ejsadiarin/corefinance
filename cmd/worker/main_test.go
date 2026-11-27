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
		name string
		q    wakeQuerier
		want time.Duration
	}{
		{"nothing due sleeps 15 minutes", wakeQuerier{}, 15 * time.Minute},
		{"a rule due in 3 minutes wakes on time", wakeQuerier{wake: at(3 * time.Minute)}, 3 * time.Minute},
		{"a rule due in an hour still checks after 15 minutes", wakeQuerier{wake: at(time.Hour)}, 15 * time.Minute},
		{"an overdue rule waits at least a second", wakeQuerier{wake: at(-time.Minute)}, time.Second},
		{"a failed lookup sleeps 15 minutes", wakeQuerier{err: errors.New("boom")}, 15 * time.Minute},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := untilNextWake(context.Background(), tc.q, now); got != tc.want {
				t.Fatalf("untilNextWake = %v, want %v", got, tc.want)
			}
		})
	}
}
