package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/google/uuid"

	"github.com/coder/websocket"
)

func (s *Server) RegisterRoutes() http.Handler {
	r := chi.NewRouter()

	// Middleware stack
	r.Use(requestIDMiddleware)
	r.Use(middleware.RealIP)
	r.Use(slogMiddleware)
	r.Use(middleware.Recoverer)

	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"https://*", "http://*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS", "PATCH"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-User-ID"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	r.Get("/", s.HelloWorldHandler)

	r.Get("/health", s.healthHandler)

	r.Get("/websocket", s.websocketHandler)

	// budget tracking routes
	r.Route("/budget", func(r chi.Router) {
		// priority groups (reference data)
		r.Get("/priority-groups", s.BudgetHandler.PriorityGroups)

		// expense categories
		r.Route("/expense-categories", func(r chi.Router) {
			r.Post("/", s.CategoryHandler.CreateExpense)
			r.Get("/", s.CategoryHandler.ListExpense)
			r.Put("/{id}", s.CategoryHandler.UpdateExpense)
			r.Delete("/{id}", s.CategoryHandler.DeleteExpense)
		})

		// income categories
		r.Route("/income-categories", func(r chi.Router) {
			r.Post("/", s.CategoryHandler.CreateIncome)
			r.Get("/", s.CategoryHandler.ListIncome)
			r.Put("/{id}", s.CategoryHandler.UpdateIncome)
			r.Delete("/{id}", s.CategoryHandler.DeleteIncome)
		})

		// tags
		r.Route("/tags", func(r chi.Router) {
			r.Post("/", s.TagHandler.Create)
			r.Get("/", s.TagHandler.List)
			r.Put("/{id}", s.TagHandler.Update)
			r.Delete("/{id}", s.TagHandler.Delete)
		})

		// expenses
		r.Route("/expenses", func(r chi.Router) {
			r.Post("/", s.ExpenseHandler.Create)
			r.Post("/skip", s.ExpenseHandler.Skip)
			r.Get("/", s.ExpenseHandler.List)
			r.Get("/search", s.ExpenseHandler.Search)
			r.Get("/check-skipped", s.ExpenseHandler.CheckSkipped)
			r.Get("/{id}", s.ExpenseHandler.Get)
			r.Put("/{id}", s.ExpenseHandler.Update)
			r.Delete("/{id}", s.ExpenseHandler.Delete)

			r.Post("/{id}/tags", s.TagHandler.AddTagToExpense)
			r.Get("/{id}/tags", s.TagHandler.GetTagsByExpenseID)
			r.Delete("/{id}/tags/{tag_id}", s.TagHandler.RemoveTagFromExpense)
		})

		// incomes
		r.Route("/incomes", func(r chi.Router) {
			r.Post("/", s.IncomeHandler.Create)
			r.Post("/skip", s.IncomeHandler.Skip)
			r.Get("/", s.IncomeHandler.List)
			r.Get("/check-skipped", s.IncomeHandler.CheckSkipped)
			r.Get("/occurrences", s.IncomeHandler.GetOccurrences)
			r.Get("/{id}", s.IncomeHandler.Get)
			r.Put("/{id}", s.IncomeHandler.Update)
			r.Delete("/{id}", s.IncomeHandler.Delete)

			r.Post("/{id}/tags", s.TagHandler.AddTagToIncome)
			r.Get("/{id}/tags", s.TagHandler.GetTagsByIncomeID)
			r.Delete("/{id}/tags/{tag_id}", s.TagHandler.RemoveTagFromIncome)
		})

		// budget remaining (income - expenses for current period)
		r.Get("/remaining", s.BudgetHandler.Remaining)
		r.Get("/export", s.BudgetHandler.Export)
		r.Post("/import", s.BudgetHandler.Import)

		// stats
		r.Route("/stats", func(r chi.Router) {
			r.Get("/summary", s.StatsHandler.Summary)
			r.Get("/trends", s.StatsHandler.Trends)
			r.Get("/category-breakdown", s.StatsHandler.CategoryBreakdown)
			r.Get("/savings-rate", s.StatsHandler.SavingsRate)
		})

		// analytics
		r.Get("/analysis/503020", s.StatsHandler.FiftyThirtyTwenty)

		// budget analytics endpoints
		r.Get("/velocity", s.StatsHandler.SpendingVelocity)
		r.Get("/forecast/upcoming", s.StatsHandler.UpcomingBills)
		r.Get("/current-total-money", s.StatsHandler.CurrentTotalMoney)

		// trends
		r.Get("/trends/month-over-month", s.StatsHandler.MonthOverMonthTrends)

		// recurring expense rules (full CRUD)
		r.Route("/recurring-expenses", func(r chi.Router) {
			r.Post("/", s.RecurringHandler.CreateExpenseRule)
			r.Get("/", s.RecurringHandler.ListExpenseRules)
			r.Get("/{id}", s.RecurringHandler.GetExpenseRule)
			r.Put("/{id}", s.RecurringHandler.UpdateExpenseRule)
			r.Delete("/{id}", s.RecurringHandler.DeleteExpenseRule)
		})

		// recurring income rules (full CRUD)
		r.Route("/recurring-incomes", func(r chi.Router) {
			r.Post("/", s.RecurringHandler.CreateIncomeRule)
			r.Get("/", s.RecurringHandler.ListIncomeRules)
			r.Get("/{id}", s.RecurringHandler.GetIncomeRule)
			r.Put("/{id}", s.RecurringHandler.UpdateIncomeRule)
			r.Delete("/{id}", s.RecurringHandler.DeleteIncomeRule)
		})
	})

	return r
}

func (s *Server) HelloWorldHandler(w http.ResponseWriter, r *http.Request) {
	resp := make(map[string]string)
	resp["message"] = "Hello World"

	jsonResp, err := json.Marshal(resp)
	if err != nil {
		slog.Error("error handling JSON marshal", "error", err)
	}

	_, _ = w.Write(jsonResp)
}

func (s *Server) healthHandler(w http.ResponseWriter, r *http.Request) {
	stats := make(map[string]string)

	ctx := r.Context()
	if err := s.Pool.Ping(ctx); err != nil {
		stats["status"] = "down"
		stats["error"] = err.Error()
	} else {
		stats["status"] = "up"
		stats["message"] = "It's healthy"
	}

	poolStats := s.Pool.Stat()
	stats["total_conns"] = fmt.Sprintf("%d", poolStats.TotalConns())
	stats["acquired_conns"] = fmt.Sprintf("%d", poolStats.AcquiredConns())
	stats["idle_conns"] = fmt.Sprintf("%d", poolStats.IdleConns())

	jsonResp, _ := json.Marshal(stats)
	_, _ = w.Write(jsonResp)
}

func (s *Server) websocketHandler(w http.ResponseWriter, r *http.Request) {
	socket, err := websocket.Accept(w, r, nil)
	if err != nil {
		slog.Error("could not open websocket", "error", err)
		_, _ = w.Write([]byte("could not open websocket"))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	defer socket.Close(websocket.StatusGoingAway, "server closing websocket")

	ctx := r.Context()
	socketCtx := socket.CloseRead(ctx)

	for {
		payload := fmt.Sprintf("server timestamp: %d", time.Now().UnixNano())
		err := socket.Write(socketCtx, websocket.MessageText, []byte(payload))
		if err != nil {
			break
		}
		time.Sleep(time.Second * 2)
	}
}

func requestIDMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get("X-Request-ID")
		if id == "" {
			id = r.Header.Get("X-Correlation-ID")
		}
		if id == "" {
			id = uuid.New().String()
		}
		ctx := context.WithValue(r.Context(), middleware.RequestIDKey, id)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func slogMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)

		next.ServeHTTP(ww, r)

		slog.Info("request",
			"method", r.Method,
			"path", r.URL.Path,
			"status", ww.Status(),
			"bytes", ww.BytesWritten(),
			"duration", time.Since(start).String(),
			"request_id", middleware.GetReqID(r.Context()),
		)
	})
}
