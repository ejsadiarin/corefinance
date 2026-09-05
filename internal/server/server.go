package server

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/ejsadiarin/corefinance/internal/category"
	db "github.com/ejsadiarin/corefinance/internal/db/sqlc"
	"github.com/ejsadiarin/corefinance/internal/database"
	"github.com/ejsadiarin/corefinance/internal/expense"
	"github.com/ejsadiarin/corefinance/internal/income"
	"github.com/ejsadiarin/corefinance/internal/recurring"
	"github.com/ejsadiarin/corefinance/internal/stats"
	"github.com/ejsadiarin/corefinance/internal/tag"

	_ "github.com/joho/godotenv/autoload"
)

type Server struct {
	port int

	db database.Service

	ExpenseHandler   *expense.Handler
	IncomeHandler    *income.Handler
	CategoryHandler  *category.Handler
	TagHandler       *tag.Handler
	StatsHandler     *stats.Handler
	RecurringHandler *recurring.Handler
	Queries          *db.Queries
	Pool             *pgxpool.Pool
}

func NewServer() *http.Server {
	port, _ := strconv.Atoi(os.Getenv("PORT"))
	NewServer := &Server{
		port: port,
		db:   database.New(),
	}

	// Create pgxpool connection for sqlc queries
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, buildConnString())
	if err != nil {
		log.Printf("unable to create pgxpool: %v", err)
	}

	// Wire up handlers (db.Queries implements all service Querier interfaces)
	queries := db.New(pool)
	NewServer.ExpenseHandler = expense.NewHandler(expense.NewService(queries))
	NewServer.IncomeHandler = income.NewHandler(income.NewService(queries))
	NewServer.CategoryHandler = category.NewHandler(category.NewService(queries))
	NewServer.TagHandler = tag.NewHandler(tag.NewService(queries))
	NewServer.StatsHandler = stats.NewHandler(stats.NewService(queries))
	NewServer.RecurringHandler = recurring.NewHandler(recurring.NewService(queries))
	NewServer.Queries = queries
	NewServer.Pool = pool

	// Declare Server config
	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", NewServer.port),
		Handler:      NewServer.RegisterRoutes(),
		IdleTimeout:  time.Minute,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	return server
}

func buildConnString() string {
	database := os.Getenv("BLUEPRINT_DB_DATABASE")
	password := os.Getenv("BLUEPRINT_DB_PASSWORD")
	username := os.Getenv("BLUEPRINT_DB_USERNAME")
	port := os.Getenv("BLUEPRINT_DB_PORT")
	host := os.Getenv("BLUEPRINT_DB_HOST")
	schema := os.Getenv("BLUEPRINT_DB_SCHEMA")
	return fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable&search_path=%s", username, password, host, port, database, schema)
}
