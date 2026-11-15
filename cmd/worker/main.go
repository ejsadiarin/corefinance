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
	"github.com/ejsadiarin/corefinance/internal/gateway"
	"github.com/ejsadiarin/corefinance/internal/logger"
	"github.com/ejsadiarin/corefinance/internal/materialize"

	_ "github.com/joho/godotenv/autoload"
)

// Env: DATABASE_URL (or DB_*), and GATEWAY_URL + GATEWAY_API_KEY: an API key
// owned by a gateway admin with the corefinance:admin scope, used to look
// users' timezones up (coregateway ADR-0005). Not needed for --backfill.
//
// Flags:
//   --once      run one materialization pass and exit (one-shot jobs, tests)
//   --backfill  run one pass of materialize + legacy inline-rule conversion
//               (D5) and exit; implies --once. Safe to re-run: instance
//               inserts are ON CONFLICT DO NOTHING and flipped rows no longer
//               match the legacy predicate.

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
		slog.Error("failed to parse pool config", "error", err)
		panic(err)
	}
	pool, err := pgxpool.NewWithConfig(context.Background(), config)
	if err != nil {
		slog.Error("unable to create pgxpool", "error", err)
		panic(err)
	}
	if err := pool.Ping(context.Background()); err != nil {
		slog.Error("unable to ping database", "error", err)
		panic(err)
	}
	return pool
}

func runOnce(ctx context.Context, pool *pgxpool.Pool, queries *db.Queries, zones materialize.Zones, backfill bool) int {
	today := time.Now()
	if backfill {
		res, err := materialize.BackfillLegacy(ctx, pool, queries, today)
		if err != nil {
			slog.Error("backfill failed", "error", err)
			return 1
		}
		legacy, err := materialize.CountLegacyRows(ctx, queries)
		if err != nil {
			slog.Error("legacy verification query failed", "error", err)
			return 1
		}
		slog.Info("backfill complete",
			"expense_rules", res.ExpenseRules,
			"income_rules", res.IncomeRules,
			"expense_instances", res.ExpenseInstances,
			"income_instances", res.IncomeInstances,
			"legacy_rows_remaining", legacy,
		)
		if legacy != 0 {
			slog.Error("legacy rows remain after backfill", "count", legacy)
			return 1
		}
		return 0
	}
	n, err := materialize.MaterializeDue(ctx, pool, queries, zones, today)
	if err != nil {
		slog.Error("materialize failed", "error", err)
		return 1
	}
	// Passes run at least once a minute; only ones that did something are news.
	if n > 0 {
		slog.Info("materialize pass complete", "inserted", n)
	} else {
		slog.Debug("materialize pass complete", "inserted", n)
	}
	return 0
}

func main() {
	once := flag.Bool("once", false, "run one materialization pass and exit")
	backfill := flag.Bool("backfill", false, "run materialize + legacy backfill once and exit (implies --once)")
	flag.Parse()

	logger.Setup()

	pool := buildPool()
	defer pool.Close()
	queries := db.New(pool)

	// The gateway owns users' timezones (ADR-0005); without a way to look them
	// up the worker can't date entries, so it refuses to run rather than guess.
	var zones materialize.Zones
	if !*backfill {
		gatewayURL, gatewayKey := os.Getenv("GATEWAY_URL"), os.Getenv("GATEWAY_API_KEY")
		if gatewayURL == "" || gatewayKey == "" {
			slog.Error("GATEWAY_URL and GATEWAY_API_KEY are required: the worker looks users' timezones up from the gateway")
			os.Exit(1)
		}
		zones = gateway.NewTimezoneClient(gatewayURL, gatewayKey)
	}

	if *backfill {
		os.Exit(runOnce(context.Background(), pool, queries, nil, true))
	}
	if *once {
		os.Exit(runOnce(context.Background(), pool, queries, zones, false))
	}

	// Long-running mode (#16): a pass, then sleep until the earliest rule's
	// next occurrence starts, checking at least once a minute so rules saved
	// in the meantime (by the API, in another process) are never late by more.
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	slog.Info("worker started, filling rules as they fall due")
	for {
		if code := runOnce(ctx, pool, queries, zones, false); code != 0 {
			slog.Error("materialize pass failed, retrying at the next wake")
		}
		wait := untilNextWake(ctx, queries, time.Now())
		select {
		case <-ctx.Done():
			slog.Info("worker shutting down")
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
			slog.Error("next wake lookup failed", "error", err)
		}
		return maxSleep
	}
	return min(max(wake.Sub(now), time.Second), maxSleep)
}
