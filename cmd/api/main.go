package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	db "github.com/ejsadiarin/corefinance/internal/db/sqlc"
	"github.com/ejsadiarin/corefinance/internal/logger"
	"github.com/ejsadiarin/corefinance/internal/server"

	_ "github.com/joho/godotenv/autoload"
)

func gracefulShutdown(apiServer *http.Server, pool *pgxpool.Pool, done chan bool) {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	<-ctx.Done()

	slog.Info("shutting down gracefully, press Ctrl+C again to force")
	stop()

	// Give in-flight requests 5s to complete
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := apiServer.Shutdown(ctx); err != nil {
		slog.Error("server forced to shutdown", "error", err)
	}

	pool.Close()
	slog.Info("database pool closed")
	slog.Info("server exiting")

	done <- true
}

func buildPool() *pgxpool.Pool {
	connStr := os.Getenv("DATABASE_URL")
	if connStr == "" {
		connStr = fmt.Sprintf(
			"postgres://%s:%s@%s:%s/%s?sslmode=disable&search_path=corefinance",
			os.Getenv("DB_USERNAME"),
			os.Getenv("DB_PASSWORD"),
			os.Getenv("DB_HOST"),
			os.Getenv("DB_PORT"),
			os.Getenv("DB_DATABASE"),
		)
	}

	config, err := pgxpool.ParseConfig(connStr)
	if err != nil {
		slog.Error("failed to parse pool config", "error", err)
		panic(err)
	}

	config.MaxConns = 20
	config.MinConns = 2
	config.MaxConnLifetime = time.Hour
	config.MaxConnIdleTime = 30 * time.Minute
	config.HealthCheckPeriod = time.Minute

	config.AfterConnect = func(ctx context.Context, conn *pgx.Conn) error {
		_, err := conn.Exec(ctx, "SET search_path TO corefinance")
		return err
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

	slog.Info("database pool connected")
	return pool
}

func main() {
	logger.Setup()

	pool := buildPool()
	defer pool.Close()

	queries := db.New(pool)

	port, _ := strconv.Atoi(os.Getenv("PORT"))
	if port == 0 {
		port = 6969
	}

	srv := server.New(port, pool, queries)

	done := make(chan bool, 1)
	go gracefulShutdown(srv, pool, done)

	slog.Info("server starting", "port", port)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		slog.Error("server error", "error", err)
	}

	<-done
	slog.Info("graceful shutdown complete")
}
