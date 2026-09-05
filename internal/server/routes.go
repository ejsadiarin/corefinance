package server

import (
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
	db "github.com/ejsadiarin/corefinance/internal/db/sqlc"
	"github.com/ejsadiarin/corefinance/internal/helper"
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
		r.Get("/remaining", s.GetBudgetRemaining)
		r.Get("/export", s.ExportBudgetJSON)
		r.Post("/import", s.ImportBudgetJSON)

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

// --- Budget Remaining (income - expenses for current month) ---

func (s *Server) GetBudgetRemaining(w http.ResponseWriter, r *http.Request) {
	userID, ok := helper.GetUserID(w, r)
	if !ok {
		return
	}

	slog.Debug("server.GetBudgetRemaining: request received", "user_id", userID)

	now := time.Now()
	startOfMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	endOfMonth := startOfMonth.AddDate(0, 1, -1)
	startStr := startOfMonth.Format("2006-01-02")
	endStr := endOfMonth.Format("2006-01-02")

	totalIncome, err := s.Queries.GetTotalIncomesByDateRange(r.Context(), db.GetTotalIncomesByDateRangeParams{
		UserID: userID,
		Date:   helper.ToPgDate(startStr),
		Date_2: helper.ToPgDate(endStr),
	})
	if err != nil {
		slog.Error("server.GetBudgetRemaining: failed to get income", "error", err, "user_id", userID)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	totalExpenses, err := s.Queries.GetTotalExpensesByDateRange(r.Context(), db.GetTotalExpensesByDateRangeParams{
		UserID:        userID,
		ExpenseDate:   helper.ToPgDate(startStr),
		ExpenseDate_2: helper.ToPgDate(endStr),
	})
	if err != nil {
		slog.Error("server.GetBudgetRemaining: failed to get expenses", "error", err, "user_id", userID)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	incomeDec := helper.ToDecimal(totalIncome)
	expenseDec := helper.ToDecimal(totalExpenses)
	remaining := incomeDec.Sub(expenseDec)

	slog.Debug("server.GetBudgetRemaining: success", "user_id", userID, "remaining", remaining)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"period_start":  startStr,
		"period_end":    endStr,
		"total_income":  incomeDec,
		"total_expense": expenseDec,
		"remaining":     remaining,
	})
}

// --- Export / Import ---

type BudgetExport struct {
	ExpenseCategories []db.ExpenseCategory             `json:"expense_categories"`
	IncomeCategories  []db.IncomeCategory              `json:"income_categories"`
	Tags              []db.Tag                         `json:"tags"`
	Expenses          []db.Expense                     `json:"expenses"`
	Incomes           []db.Income                      `json:"incomes"`
	RecurringExpenses []db.ListRecurringExpenseRulesRow `json:"recurring_expenses"`
	RecurringIncomes  []db.RecurringIncomeRule         `json:"recurring_incomes"`
	ExpenseTags       []db.ExpenseTag                  `json:"expense_tags"`
	IncomeTags        []db.IncomeTag                   `json:"income_tags"`
}

func (s *Server) ExportBudgetJSON(w http.ResponseWriter, r *http.Request) {
	userID, ok := helper.GetUserID(w, r)
	if !ok {
		return
	}

	slog.Debug("server.ExportBudgetJSON: request received", "user_id", userID)

	ctx := r.Context()

	expenseCategories, err := s.Queries.ListExpenseCategories(ctx, userID)
	if err != nil {
		slog.Error("server.ExportBudgetJSON: failed to list expense categories", "error", err, "user_id", userID)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	incomeCategories, err := s.Queries.ListIncomeCategories(ctx, userID)
	if err != nil {
		slog.Error("server.ExportBudgetJSON: failed to list income categories", "error", err, "user_id", userID)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	tags, err := s.Queries.ListTags(ctx, userID)
	if err != nil {
		slog.Error("server.ExportBudgetJSON: failed to list tags", "error", err, "user_id", userID)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	expenses, err := s.Queries.ListAllExpensesByUser(ctx, userID)
	if err != nil {
		slog.Error("server.ExportBudgetJSON: failed to list expenses", "error", err, "user_id", userID)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	incomes, err := s.Queries.ListAllIncomesByUser(ctx, userID)
	if err != nil {
		slog.Error("server.ExportBudgetJSON: failed to list incomes", "error", err, "user_id", userID)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	recurringExpenses, err := s.Queries.ListRecurringExpenseRules(ctx, userID)
	if err != nil {
		slog.Error("server.ExportBudgetJSON: failed to list recurring expenses", "error", err, "user_id", userID)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	recurringIncomes, err := s.Queries.ListRecurringIncomeRules(ctx, userID)
	if err != nil {
		slog.Error("server.ExportBudgetJSON: failed to list recurring incomes", "error", err, "user_id", userID)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	expenseTags, err := s.Queries.ListAllExpenseTagsByUser(ctx, userID)
	if err != nil {
		slog.Error("server.ExportBudgetJSON: failed to list expense tags", "error", err, "user_id", userID)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	incomeTags, err := s.Queries.ListAllIncomeTagsByUser(ctx, userID)
	if err != nil {
		slog.Error("server.ExportBudgetJSON: failed to list income tags", "error", err, "user_id", userID)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	export := BudgetExport{
		ExpenseCategories: expenseCategories,
		IncomeCategories:  incomeCategories,
		Tags:              tags,
		Expenses:          expenses,
		Incomes:           incomes,
		RecurringExpenses: recurringExpenses,
		RecurringIncomes:  recurringIncomes,
		ExpenseTags:       expenseTags,
		IncomeTags:        incomeTags,
	}

	slog.Info("server.ExportBudgetJSON: export completed", "user_id", userID)

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Disposition", "attachment; filename=budget_export.json")
	json.NewEncoder(w).Encode(export)
}

func (s *Server) ImportBudgetJSON(w http.ResponseWriter, r *http.Request) {
	userID, ok := helper.GetUserID(w, r)
	if !ok {
		return
	}

	slog.Debug("server.ImportBudgetJSON: request received", "user_id", userID)

	var export BudgetExport
	if err := json.NewDecoder(r.Body).Decode(&export); err != nil {
		helper.RespondErrorJSON(w, http.StatusBadRequest, "invalid request body")
		return
	}

	ctx := r.Context()
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		slog.Error("server.ImportBudgetJSON: failed to start transaction", "error", err, "user_id", userID)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "failed to start transaction"})
		return
	}
	defer tx.Rollback(ctx)

	q := s.Queries.WithTx(tx)

	// Track ID mappings: old ID -> new ID
	categoryMap := make(map[uuid.UUID]uuid.UUID)
	tagMap := make(map[uuid.UUID]uuid.UUID)

	// Import expense categories
	for _, cat := range export.ExpenseCategories {
		newCat, err := q.CreateExpenseCategory(ctx, db.CreateExpenseCategoryParams{
			UserID: userID,
			Name:   cat.Name,
			Color:  cat.Color,
			Icon:   cat.Icon,
		})
		if err != nil {
			slog.Error("server.ImportBudgetJSON: failed to import expense categories", "error", err, "user_id", userID)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": "failed to import expense categories: " + err.Error()})
			return
		}
		categoryMap[cat.ID] = newCat.ID
	}

	// Import income categories
	for _, cat := range export.IncomeCategories {
		newCat, err := q.CreateIncomeCategory(ctx, db.CreateIncomeCategoryParams{
			UserID: userID,
			Name:   cat.Name,
			Color:  cat.Color,
			Icon:   cat.Icon,
		})
		if err != nil {
			slog.Error("server.ImportBudgetJSON: failed to import income categories", "error", err, "user_id", userID)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": "failed to import income categories: " + err.Error()})
			return
		}
		categoryMap[cat.ID] = newCat.ID
	}

	// Import tags
	for _, tag := range export.Tags {
		newTag, err := q.CreateTag(ctx, db.CreateTagParams{
			UserID: userID,
			Name:   tag.Name,
			Color:  tag.Color,
		})
		if err != nil {
			slog.Error("server.ImportBudgetJSON: failed to import tags", "error", err, "user_id", userID)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": "failed to import tags: " + err.Error()})
			return
		}
		tagMap[tag.ID] = newTag.ID
	}

	// Track expense/income ID mappings
	expenseMap := make(map[uuid.UUID]uuid.UUID)
	incomeMap := make(map[uuid.UUID]uuid.UUID)

	// Import expenses
	for _, exp := range export.Expenses {
		newCatID := uuid.Nil
		if exp.CategoryID.Valid {
			if mapped, ok := categoryMap[exp.CategoryID.Bytes]; ok {
				newCatID = mapped
			}
		}
		newRuleID := uuid.Nil
		if exp.SourceRuleID.Valid {
			newRuleID = exp.SourceRuleID.Bytes
		}
		newExp, err := q.CreateExpense(ctx, db.CreateExpenseParams{
			UserID:        userID,
			CategoryID:    helper.ToPgUUIDFromUUID(newCatID),
			Amount:        exp.Amount,
			Currency:      exp.Currency,
			Description:   exp.Description,
			Notes:         exp.Notes,
			ExpenseDate:   exp.ExpenseDate,
			RecurringType: exp.RecurringType,
			Priority:      exp.Priority,
			Status:        exp.Status,
			IsDebt:        exp.IsDebt,
			StartDate:     exp.StartDate,
			EndDate:       exp.EndDate,
			SourceRuleID:  helper.ToPgUUIDFromUUID(newRuleID),
		})
		if err != nil {
			slog.Error("server.ImportBudgetJSON: failed to import expenses", "error", err, "user_id", userID)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": "failed to import expenses: " + err.Error()})
			return
		}
		expenseMap[exp.ID] = newExp.ID
	}

	// Import incomes
	for _, inc := range export.Incomes {
		newCatID := uuid.Nil
		if inc.CategoryID.Valid {
			if mapped, ok := categoryMap[inc.CategoryID.Bytes]; ok {
				newCatID = mapped
			}
		}
		newRuleID := uuid.Nil
		if inc.SourceRuleID.Valid {
			newRuleID = inc.SourceRuleID.Bytes
		}
		newInc, err := q.CreateIncome(ctx, db.CreateIncomeParams{
			UserID:        userID,
			CategoryID:    helper.ToPgUUIDFromUUID(newCatID),
			Amount:        inc.Amount,
			Currency:      inc.Currency,
			Description:   inc.Description,
			Notes:         inc.Notes,
			Date:          inc.Date,
			RecurringType: inc.RecurringType,
			Priority:      inc.Priority,
			Status:        inc.Status,
			StartDate:     inc.StartDate,
			EndDate:       inc.EndDate,
			SourceRuleID:  helper.ToPgUUIDFromUUID(newRuleID),
		})
		if err != nil {
			slog.Error("server.ImportBudgetJSON: failed to import incomes", "error", err, "user_id", userID)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": "failed to import incomes: " + err.Error()})
			return
		}
		incomeMap[inc.ID] = newInc.ID
	}

	// Import recurring expense rules
	for _, rule := range export.RecurringExpenses {
		newCatID := uuid.Nil
		if rule.CategoryID.Valid {
			if mapped, ok := categoryMap[rule.CategoryID.Bytes]; ok {
				newCatID = mapped
			}
		}
		_, err := q.CreateRecurringExpenseRule(ctx, db.CreateRecurringExpenseRuleParams{
			UserID:        userID,
			Description:   rule.Description,
			Amount:        rule.Amount,
			Currency:      rule.Currency,
			CategoryID:    helper.ToPgUUIDFromUUID(newCatID),
			Notes:         rule.Notes,
			RecurringType: rule.RecurringType,
			StartDate:     rule.StartDate,
			EndDate:       rule.EndDate,
			Priority:      rule.Priority,
		})
		if err != nil {
			slog.Error("server.ImportBudgetJSON: failed to import recurring expense rules", "error", err, "user_id", userID)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": "failed to import recurring expense rules: " + err.Error()})
			return
		}
	}

	// Import recurring income rules
	for _, rule := range export.RecurringIncomes {
		_, err := q.CreateRecurringIncomeRule(ctx, db.CreateRecurringIncomeRuleParams{
			UserID:        userID,
			Amount:        rule.Amount,
			Currency:      rule.Currency,
			Description:   rule.Description,
			RecurringType: rule.RecurringType,
			StartDate:     rule.StartDate,
			EndDate:       rule.EndDate,
		})
		if err != nil {
			slog.Error("server.ImportBudgetJSON: failed to import recurring income rules", "error", err, "user_id", userID)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": "failed to import recurring income rules: " + err.Error()})
			return
		}
	}

	// Import expense tags
	for _, et := range export.ExpenseTags {
		newExpID, expOk := expenseMap[et.ExpenseID]
		newTagID, tagOk := tagMap[et.TagID]
		if expOk && tagOk {
			q.AddTagToExpense(ctx, db.AddTagToExpenseParams{
				ExpenseID: newExpID,
				TagID:     newTagID,
			})
		}
	}

	// Import income tags
	for _, it := range export.IncomeTags {
		newIncID, incOk := incomeMap[it.IncomeID]
		newTagID, tagOk := tagMap[it.TagID]
		if incOk && tagOk {
			q.AddTagToIncome(ctx, db.AddTagToIncomeParams{
				IncomeID: newIncID,
				TagID:    newTagID,
			})
		}
	}

	if err := tx.Commit(ctx); err != nil {
		slog.Error("server.ImportBudgetJSON: failed to commit transaction", "error", err, "user_id", userID)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "failed to commit transaction"})
		return
	}

	slog.Info("server.ImportBudgetJSON: import completed", "user_id", userID)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "imported"})
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
