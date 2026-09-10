package main

import (
	"database/sql"
	"fmt"
	"log/slog"
	"os"

	"github.com/pressly/goose/v3"

	"github.com/ejsadiarin/corefinance/internal/db/migrations"
	_ "github.com/jackc/pgx/v5/stdlib"
	_ "github.com/joho/godotenv/autoload"
)

func dsn() string {
	if connStr := os.Getenv("DATABASE_URL"); connStr != "" {
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

func main() {
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
		slog.Error("failed to set goose dialect", "error", err)
		os.Exit(1)
	}

	if err := goose.Up(db, "."); err != nil {
		slog.Error("goose up failed", "error", err)
		os.Exit(1)
	}

	slog.Info("migrations applied successfully")
}
