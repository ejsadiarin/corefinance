package server

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"

	"github.com/coder/websocket"
)

func (s *Server) RegisterRoutes() http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.Logger)

	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"https://*", "http://*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS", "PATCH"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	r.Get("/", s.HelloWorldHandler)

	r.Get("/health", s.healthHandler)

	r.Get("/websocket", s.websocketHandler)

	// budget tracking routes
	r.Route("/budget", func(r chi.Router) {
		// priority groups (reference data)
		r.Get("/priority-groups", s.BudgetHandler.GetPriorityGroups)

		// categories
		r.Route("/categories", func(r chi.Router) {
			r.Post("/", s.BudgetHandler.CreateCategory)
			r.Get("/", s.BudgetHandler.ListCategories)
			r.Put("/{id}", s.BudgetHandler.UpdateCategory)
			r.Delete("/{id}", s.BudgetHandler.DeleteCategory)
		})

		// tags
		r.Route("/tags", func(r chi.Router) {
			r.Post("/", s.BudgetHandler.CreateTag)
			r.Get("/", s.BudgetHandler.ListTags)
			r.Put("/{id}", s.BudgetHandler.UpdateTag)
			r.Delete("/{id}", s.BudgetHandler.DeleteTag)
		})

		// expenses
		r.Route("/expenses", func(r chi.Router) {
			r.Post("/", s.BudgetHandler.CreateExpense)
			r.Post("/skip", s.BudgetHandler.SkipExpense)
			r.Get("/", s.BudgetHandler.ListExpenses)
			r.Get("/search", s.BudgetHandler.SearchExpenses)
			r.Get("/check-skipped", s.BudgetHandler.CheckSkippedExpense)
			r.Get("/{id}", s.BudgetHandler.GetExpense)
			r.Put("/{id}", s.BudgetHandler.UpdateExpense)
			r.Delete("/{id}", s.BudgetHandler.DeleteExpense)
		})

		// incomes
		r.Route("/incomes", func(r chi.Router) {
			r.Post("/", s.BudgetHandler.CreateIncome)
			r.Post("/skip", s.BudgetHandler.SkipIncome)
			r.Get("/", s.BudgetHandler.ListIncomes)
			r.Get("/check-skipped", s.BudgetHandler.CheckSkippedIncome)
			r.Get("/occurrences", s.BudgetHandler.GetIncomeOccurrences)
			r.Get("/{id}", s.BudgetHandler.GetIncome)
			r.Put("/{id}", s.BudgetHandler.UpdateIncome)
			r.Delete("/{id}", s.BudgetHandler.DeleteIncome)
		})

		// budget remaining
		r.Get("/remaining", s.BudgetHandler.GetBudgetRemaining)
		r.Get("/export", s.BudgetHandler.ExportBudgetJSON)
		r.Post("/import", s.BudgetHandler.ImportBudgetJSON)

		// stats
		r.Route("/stats", func(r chi.Router) {
			r.Get("/summary", s.BudgetHandler.GetSummary)
			r.Get("/trends", s.BudgetHandler.GetTrends)
			r.Get("/category-breakdown", s.BudgetHandler.GetCategoryBreakdown)
			r.Get("/savings-rate", s.BudgetHandler.GetSavingsRate)
			r.Get("/health-score", s.BudgetHandler.GetHealthScore)
		})

		// budget analytics endpoints
		r.Get("/velocity", s.BudgetHandler.GetSpendingVelocity)
		r.Get("/forecast/upcoming", s.BudgetHandler.GetUpcomingBills)
		r.Get("/current-total-money", s.BudgetHandler.GetCurrentTotalMoney)

		// financial health analysis
		r.Get("/analysis/503020", s.BudgetHandler.GetFiftyThirtyTwenty)
		r.Get("/analysis/weekday-pattern", s.BudgetHandler.GetWeekdayPattern)
		r.Get("/analysis/merchants", s.BudgetHandler.GetMerchantAnalysis)
		r.Get("/trends/month-over-month", s.BudgetHandler.GetMonthOverMonthTrends)

		// subscriptions
		r.Get("/subscriptions", s.BudgetHandler.GetSubscriptions)

		// recurring incomes
		r.Get("/recurring-incomes", s.BudgetHandler.GetRecurringIncomes)

		// category budgets
		r.Route("/category-budgets", func(r chi.Router) {
			r.Post("/", s.BudgetHandler.CreateCategoryBudget)
			r.Get("/", s.BudgetHandler.GetCategoryBudgets)
			r.Put("/{id}", s.BudgetHandler.UpdateCategoryBudget)
			r.Delete("/{id}", s.BudgetHandler.DeleteCategoryBudget)
		})
	})

	return r
}

func (s *Server) HelloWorldHandler(w http.ResponseWriter, r *http.Request) {
	resp := make(map[string]string)
	resp["message"] = "Hello World"

	jsonResp, err := json.Marshal(resp)
	if err != nil {
		log.Fatalf("error handling JSON marshal. Err: %v", err)
	}

	_, _ = w.Write(jsonResp)
}

func (s *Server) healthHandler(w http.ResponseWriter, r *http.Request) {
	jsonResp, _ := json.Marshal(s.db.Health())
	_, _ = w.Write(jsonResp)
}

func (s *Server) websocketHandler(w http.ResponseWriter, r *http.Request) {
	socket, err := websocket.Accept(w, r, nil)
	if err != nil {
		log.Printf("could not open websocket: %v", err)
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
