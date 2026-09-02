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
		r.Get("/priority-groups", s.GetPriorityGroups)

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
		})

		// budget remaining
		r.Get("/remaining", s.GetBudgetRemaining)
		r.Get("/export", s.ExportBudgetJSON)
		r.Post("/import", s.ImportBudgetJSON)

		// stats
		r.Route("/stats", func(r chi.Router) {
			r.Get("/summary", s.GetSummary)
			r.Get("/trends", s.GetTrends)
			r.Get("/category-breakdown", s.GetCategoryBreakdown)
			r.Get("/savings-rate", s.GetSavingsRate)
			r.Get("/health-score", s.GetHealthScore)
		})

		// budget analytics endpoints
		r.Get("/velocity", s.GetSpendingVelocity)
		r.Get("/forecast/upcoming", s.GetUpcomingBills)
		r.Get("/current-total-money", s.GetCurrentTotalMoney)

		// financial health analysis
		r.Get("/analysis/503020", s.GetFiftyThirtyTwenty)
		r.Get("/analysis/weekday-pattern", s.GetWeekdayPattern)
		r.Get("/analysis/merchants", s.GetMerchantAnalysis)
		r.Get("/trends/month-over-month", s.GetMonthOverMonthTrends)

		// subscriptions
		r.Get("/subscriptions", s.GetSubscriptions)

		// recurring incomes
		r.Get("/recurring-incomes", s.GetRecurringIncomes)

		// category budgets
		r.Route("/category-budgets", func(r chi.Router) {
			r.Post("/", s.CreateCategoryBudget)
			r.Get("/", s.GetCategoryBudgets)
			r.Put("/{id}", s.UpdateCategoryBudget)
			r.Delete("/{id}", s.DeleteCategoryBudget)
		})
	})

	return r
}

// --- Reference Data ---

func (s *Server) GetPriorityGroups(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode([]map[string]string{
		{"id": "need", "name": "Need", "description": "Essential expenses"},
		{"id": "want", "name": "Want", "description": "Non-essential expenses"},
		{"id": "savings", "name": "Savings", "description": "Savings and investments"},
	})
}

// --- Stubs for unimplemented endpoints ---

func (s *Server) GetBudgetRemaining(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNotImplemented)
	json.NewEncoder(w).Encode(map[string]string{"error": "not implemented"})
}

func (s *Server) ExportBudgetJSON(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNotImplemented)
	json.NewEncoder(w).Encode(map[string]string{"error": "not implemented"})
}

func (s *Server) ImportBudgetJSON(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNotImplemented)
	json.NewEncoder(w).Encode(map[string]string{"error": "not implemented"})
}

func (s *Server) GetSummary(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNotImplemented)
	json.NewEncoder(w).Encode(map[string]string{"error": "not implemented"})
}

func (s *Server) GetTrends(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNotImplemented)
	json.NewEncoder(w).Encode(map[string]string{"error": "not implemented"})
}

func (s *Server) GetCategoryBreakdown(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNotImplemented)
	json.NewEncoder(w).Encode(map[string]string{"error": "not implemented"})
}

func (s *Server) GetSavingsRate(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNotImplemented)
	json.NewEncoder(w).Encode(map[string]string{"error": "not implemented"})
}

func (s *Server) GetHealthScore(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNotImplemented)
	json.NewEncoder(w).Encode(map[string]string{"error": "not implemented"})
}

func (s *Server) GetSpendingVelocity(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNotImplemented)
	json.NewEncoder(w).Encode(map[string]string{"error": "not implemented"})
}

func (s *Server) GetUpcomingBills(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNotImplemented)
	json.NewEncoder(w).Encode(map[string]string{"error": "not implemented"})
}

func (s *Server) GetCurrentTotalMoney(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNotImplemented)
	json.NewEncoder(w).Encode(map[string]string{"error": "not implemented"})
}

func (s *Server) GetFiftyThirtyTwenty(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNotImplemented)
	json.NewEncoder(w).Encode(map[string]string{"error": "not implemented"})
}

func (s *Server) GetWeekdayPattern(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNotImplemented)
	json.NewEncoder(w).Encode(map[string]string{"error": "not implemented"})
}

func (s *Server) GetMerchantAnalysis(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNotImplemented)
	json.NewEncoder(w).Encode(map[string]string{"error": "not implemented"})
}

func (s *Server) GetMonthOverMonthTrends(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNotImplemented)
	json.NewEncoder(w).Encode(map[string]string{"error": "not implemented"})
}

func (s *Server) GetSubscriptions(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNotImplemented)
	json.NewEncoder(w).Encode(map[string]string{"error": "not implemented"})
}

func (s *Server) GetRecurringIncomes(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNotImplemented)
	json.NewEncoder(w).Encode(map[string]string{"error": "not implemented"})
}

func (s *Server) CreateCategoryBudget(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNotImplemented)
	json.NewEncoder(w).Encode(map[string]string{"error": "not implemented"})
}

func (s *Server) GetCategoryBudgets(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNotImplemented)
	json.NewEncoder(w).Encode(map[string]string{"error": "not implemented"})
}

func (s *Server) UpdateCategoryBudget(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNotImplemented)
	json.NewEncoder(w).Encode(map[string]string{"error": "not implemented"})
}

func (s *Server) DeleteCategoryBudget(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNotImplemented)
	json.NewEncoder(w).Encode(map[string]string{"error": "not implemented"})
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
