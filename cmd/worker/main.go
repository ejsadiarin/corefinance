package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	db "github.com/ejsadiarin/corefinance/internal/db/sqlc"
	"github.com/ejsadiarin/corefinance/internal/logger"
	"github.com/ejsadiarin/corefinance/internal/materialize"

	_ "github.com/joho/godotenv/autoload"
)

// Env: DATABASE_URL (or DB_*). Entries are dated in each user's last-seen
// timezone (the zone their last request's token carried; UTC if none), read
// from corefinance's own database.
//
// Flags:
//   --once  run one materialization pass and exit (one-shot jobs, tests)

func dsn() string {
	// NOTE: no search_path handling here by design. Schema scoping lives
	// exclusively in the connecting role's default
	// (ALTER ROLE <app> SET search_path = corefinance), which is the only
	// mechanism that survives connection poolers. Set it in every
	// environment, including local dev.
	if connStr := os.Getenv("DATABASE_URL"); connStr != "" {
		return connStr
	}
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=disable",
		os.Getenv("DB_USERNAME"),
		os.Getenv("DB_PASSWORD"),
		os.Getenv("DB_HOST"),
		os.Getenv("DB_PORT"),
		os.Getenv("DB_DATABASE"),
	)
}

func buildPool() *pgxpool.Pool {
	config, err := pgxpool.ParseConfig(dsn())
	if err != nil {
		slog.Error("main.buildPool: failed to parse pool config", "error", err)
		panic(err)
	}
	pool, err := pgxpool.NewWithConfig(context.Background(), config)
	if err != nil {
		slog.Error("main.buildPool: unable to create pgxpool", "error", err)
		panic(err)
	}
	if err := pool.Ping(context.Background()); err != nil {
		slog.Error("main.buildPool: unable to ping database", "error", err)
		panic(err)
	}
	return pool
}

func runOnce(ctx context.Context, pool *pgxpool.Pool, queries *db.Queries, zones materialize.Zones) int {
	today := time.Now()
	n, err := materialize.MaterializeDue(ctx, pool, queries, zones, today)
	if err != nil {
		// Some users were skipped (a failed zone read, a failed write); the
		// rest were filled, and the skipped ones are retried next pass.
		slog.Warn("main.runOnce: materialize pass skipped some users", "inserted", n, "error", err)
		return 1
	}
	// Passes run at least once a minute; only ones that did something are news.
	if n > 0 {
		slog.Info("main.runOnce: materialize pass complete", "inserted", n)
	} else {
		slog.Debug("main.runOnce: materialize pass complete", "inserted", n)
	}
	return 0
}

func main() {
	once := flag.Bool("once", false, "run one materialization pass and exit")
	flag.Parse()

	logger.Setup()

	pool := buildPool()
	defer pool.Close()
	queries := db.New(pool)

	zones := materialize.SeenZones{Queries: queries}

	if *once {
		os.Exit(runOnce(context.Background(), pool, queries, zones))
	}

	// Long-running mode (#16): a pass, then sleep until the earliest rule's
	// next occurrence starts, checking at least once a minute so rules saved
	// in the meantime (by the API, in another process) are never late by more.
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	slog.Info("main: worker started, filling rules as they fall due")
	for {
		if code := runOnce(ctx, pool, queries, zones); code != 0 {
			slog.Info("main: retrying skipped users at the next wake")
		}
		wait := untilNextWake(ctx, queries, time.Now())
		select {
		case <-ctx.Done():
			slog.Info("main: worker shutting down")
			return
		case <-time.After(wait):
		}
	}
}

// maxSleep caps how long the worker sleeps between passes.
const maxSleep = time.Minute

// untilNextWake is how long to sleep before the next pass: until the earliest
// next occurrence, at least a second and at most maxSleep.
func untilNextWake(ctx context.Context, queries *db.Queries, now time.Time) time.Duration {
	wake, err := materialize.NextWake(ctx, queries, now)
	if err != nil {
		if !errors.Is(err, materialize.ErrNothingDue) {
			slog.Error("main.untilNextWake: next wake lookup failed", "error", err)
		}
		return maxSleep
	}
	return min(max(wake.Sub(now), time.Second), maxSleep)
}
