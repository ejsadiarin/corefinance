package main

import (
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/pressly/goose/v3"

	"github.com/ejsadiarin/corefinance/internal/db/migrations"
	_ "github.com/jackc/pgx/v5/stdlib"
	_ "github.com/joho/godotenv/autoload"
)

func dsn() string {
	if connStr := os.Getenv("DATABASE_URL"); connStr != "" {
		// The owner role carries no search_path default, so scope the
		// session explicitly here (migrate is the one binary allowed to
		// do this: it always runs on a direct connection, never pooled).
		// Use the options=-c form (not a bare search_path param): libpq
		// rejects unknown URI params outright and some proxies poolers
		// drop them, while options is forwarded everywhere on direct
		// connections.
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
		"postgres://%s:%s@%s:%s/%s?sslmode=disable&options=-c%20search_path%3Dcorefinance",
		os.Getenv("DB_USERNAME"),
		os.Getenv("DB_PASSWORD"),
		os.Getenv("DB_HOST"),
		os.Getenv("DB_PORT"),
		os.Getenv("DB_DATABASE"),
	)
}

func main() {
	// Migrations must run on a direct (unpooled) connection: poolers
	// reject search_path scoping, which this binary needs because the
	// baseline migration is intentionally schema-unqualified and the
	// owner role carries no search_path default. Fail fast instead of
	// migrating the wrong schema.
	if strings.Contains(os.Getenv("DATABASE_URL"), "-pooler") {
		slog.Error("migrate requires a direct (unpooled) DATABASE_URL; refusing to run through a pooler")
		os.Exit(1)
	}

	db, err := sql.Open("pgx", dsn())
	if err != nil {
		slog.Error("failed to open database", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		slog.Error("unable to ping database", "error", err)
		os.Exit(1)
	}

	goose.SetBaseFS(migrations.FS)
	if err := goose.SetDialect("postgres"); err != nil {
		slog.Error("failed to set dialect", "error", err)
		os.Exit(1)
	}

	// The baseline creates the schema too, but goose bookkeeping
	// (version table) runs before any migration — and with search_path
	// pointing at a not-yet-existing schema every statement fails.
	// Ensure it here; the baseline's own CREATE SCHEMA stays as the
	// migration-native record and is a no-op afterwards.
	if _, err := db.Exec(`CREATE SCHEMA IF NOT EXISTS corefinance`); err != nil {
		slog.Error("failed to ensure corefinance schema", "error", err)
		os.Exit(1)
	}

	if err := goose.Up(db, "."); err != nil {
		slog.Error("goose up failed", "error", err)
		os.Exit(1)
	}

	slog.Info("migrations applied successfully")
}
