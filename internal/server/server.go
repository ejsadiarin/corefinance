package server

import (
	"fmt"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/ejsadiarin/corefinance/internal/auth"
	"github.com/ejsadiarin/corefinance/internal/budget"
	"github.com/ejsadiarin/corefinance/internal/category"
	db "github.com/ejsadiarin/corefinance/internal/db/sqlc"
	"github.com/ejsadiarin/corefinance/internal/expense"
	"github.com/ejsadiarin/corefinance/internal/income"
	"github.com/ejsadiarin/corefinance/internal/recurring"
	"github.com/ejsadiarin/corefinance/internal/stats"
	"github.com/ejsadiarin/corefinance/internal/tag"
)

type Server struct {
	port int

	ExpenseHandler   *expense.Handler
	IncomeHandler    *income.Handler
	CategoryHandler  *category.Handler
	TagHandler       *tag.Handler
	StatsHandler     *stats.Handler
	RecurringHandler *recurring.Handler
	BudgetHandler    *budget.Handler
	Verifier         *auth.Verifier
	Queries          *db.Queries
	Pool             *pgxpool.Pool
}

// New builds the server. verifier must be non-nil: without JWT
// verification the service refuses to serve budget routes.
func New(port int, pool *pgxpool.Pool, queries *db.Queries, verifier *auth.Verifier) *http.Server {
	if verifier == nil {
		panic("server: auth verifier is required")
	}
	s := &Server{
		port:     port,
		Pool:     pool,
		Queries:  queries,
		Verifier: verifier,
	}

	s.ExpenseHandler = expense.NewHandler(expense.NewService(queries))
	s.IncomeHandler = income.NewHandler(income.NewService(queries))
	s.CategoryHandler = category.NewHandler(category.NewService(queries))
	s.TagHandler = tag.NewHandler(tag.NewService(queries))
	s.StatsHandler = stats.NewHandler(stats.NewService(queries))
	s.RecurringHandler = recurring.NewHandler(recurring.NewService(queries))
	s.BudgetHandler = budget.NewHandler(budget.NewService(queries, pool))

	return &http.Server{
		Addr:         fmt.Sprintf(":%d", port),
		Handler:      s.RegisterRoutes(),
		IdleTimeout:  time.Minute,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
	}
}
