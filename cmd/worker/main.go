package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	db "github.com/ejsadiarin/corefinance/internal/db/sqlc"
	"github.com/ejsadiarin/corefinance/internal/logger"
	"github.com/ejsadiarin/corefinance/internal/materialize"

	_ "github.com/joho/godotenv/autoload"
)

// Flags:
//   --once      run one materialization pass and exit (one-shot jobs, tests)
//   --backfill  run one pass of materialize + legacy inline-rule conversion
//               (D5) and exit; implies --once. Safe to re-run: instance
//               inserts are ON CONFLICT DO NOTHING and flipped rows no longer
//               match the legacy predicate.

func dsn() string {
	if connStr := os.Getenv("DATABASE_URL"); connStr != "" {
		// Unlike cmd/api (AfterConnect hook), this binary sets no session
		// defaults, so guarantee the corefinance schema is in scope.
		// Use the options=-c form (not a bare search_path param): libpq
		// rejects unknown URI params outright and some proxies poolers
		// drop them, while options is forwarded everywhere.
		if !strings.Contains(connStr, "search_path") {
			sep := "?"
			if strings.Contains(connStr, "?") {
				sep = "&"
			}
			connStr += sep + "options=-c%20search_path%3Dcorefinance"
		}
		return connStr
	}
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=disable&search_path=corefinance",
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

func runOnce(ctx context.Context, pool *pgxpool.Pool, queries *db.Queries, backfill bool) int {
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
	n, err := materialize.MaterializeDue(ctx, pool, queries, today)
	if err != nil {
		slog.Error("materialize failed", "error", err)
		return 1
	}
	slog.Info("materialize pass complete", "inserted", n)
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

	if *backfill {
		os.Exit(runOnce(context.Background(), pool, queries, true))
	}
	if *once {
		os.Exit(runOnce(context.Background(), pool, queries, false))
	}

	// Long-running mode: one pass at startup, then every 24h on a stdlib ticker.
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if code := runOnce(ctx, pool, queries, false); code != 0 {
		slog.Error("initial materialize pass failed, continuing with schedule")
	}

	ticker := time.NewTicker(24 * time.Hour)
	defer ticker.Stop()
	slog.Info("worker started, daily materialization scheduled")
	for {
		select {
		case <-ctx.Done():
			slog.Info("worker shutting down")
			return
		case t := <-ticker.C:
			n, err := materialize.MaterializeDue(ctx, pool, queries, t)
			if err != nil {
				slog.Error("scheduled materialize failed", "error", err)
				continue
			}
			slog.Info("scheduled materialize pass complete", "inserted", n)
		}
	}
}
