package stats

import (
	"log/slog"
	"net/http"
	"strconv"

	"github.com/ejsadiarin/corefinance/internal/helper"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) Summary(w http.ResponseWriter, r *http.Request) {
	userID, ok := helper.ResolveTargetUser(w, r)
	if !ok {
		return
	}
	startDate := helper.ParseQueryString(r, "start_date")
	endDate := helper.ParseQueryString(r, "end_date")
	slog.Debug("stats.Summary: request received", "user_id", userID)
	summary, err := h.service.Summary(r.Context(), userID, startDate, endDate)
	if err != nil {
		slog.Error("stats.Summary: failed", "error", err, "user_id", userID)
		helper.RespondErrorJSON(w, http.StatusInternalServerError, err.Error())
		return
	}
	slog.Debug("stats.Summary: success", "user_id", userID)
	helper.RespondJSON(w, http.StatusOK, summary)
}

func (h *Handler) Trends(w http.ResponseWriter, r *http.Request) {
	userID, ok := helper.ResolveTargetUser(w, r)
	if !ok {
		return
	}
	startDate := helper.ParseQueryString(r, "start_date")
	endDate := helper.ParseQueryString(r, "end_date")
	slog.Debug("stats.Trends: request received", "user_id", userID)
	trends, err := h.service.Trends(r.Context(), userID, startDate, endDate)
	if err != nil {
		slog.Error("stats.Trends: failed", "error", err, "user_id", userID)
		helper.RespondErrorJSON(w, http.StatusInternalServerError, err.Error())
		return
	}
	slog.Debug("stats.Trends: success", "user_id", userID, "count", len(trends))
	helper.RespondJSON(w, http.StatusOK, trends)
}

func (h *Handler) CategoryBreakdown(w http.ResponseWriter, r *http.Request) {
	userID, ok := helper.ResolveTargetUser(w, r)
	if !ok {
		return
	}
	startDate := helper.ParseQueryString(r, "start_date")
	endDate := helper.ParseQueryString(r, "end_date")
	slog.Debug("stats.CategoryBreakdown: request received", "user_id", userID)
	breakdown, err := h.service.CategoryBreakdown(r.Context(), userID, startDate, endDate)
	if err != nil {
		slog.Error("stats.CategoryBreakdown: failed", "error", err, "user_id", userID)
		helper.RespondErrorJSON(w, http.StatusInternalServerError, err.Error())
		return
	}
	slog.Debug("stats.CategoryBreakdown: success", "user_id", userID, "count", len(breakdown))
	helper.RespondJSON(w, http.StatusOK, breakdown)
}

func (h *Handler) SavingsRate(w http.ResponseWriter, r *http.Request) {
	userID, ok := helper.ResolveTargetUser(w, r)
	if !ok {
		return
	}
	startDate := helper.ParseQueryString(r, "start_date")
	endDate := helper.ParseQueryString(r, "end_date")
	slog.Debug("stats.SavingsRate: request received", "user_id", userID)
	rate, err := h.service.SavingsRate(r.Context(), userID, startDate, endDate)
	if err != nil {
		slog.Error("stats.SavingsRate: failed", "error", err, "user_id", userID)
		helper.RespondErrorJSON(w, http.StatusInternalServerError, err.Error())
		return
	}
	slog.Debug("stats.SavingsRate: success", "user_id", userID)
	helper.RespondJSON(w, http.StatusOK, rate)
}

func (h *Handler) FiftyThirtyTwenty(w http.ResponseWriter, r *http.Request) {
	userID, ok := helper.ResolveTargetUser(w, r)
	if !ok {
		return
	}
	startDate := helper.ParseQueryString(r, "start_date")
	endDate := helper.ParseQueryString(r, "end_date")
	slog.Debug("stats.FiftyThirtyTwenty: request received", "user_id", userID)
	result, err := h.service.FiftyThirtyTwenty(r.Context(), userID, startDate, endDate)
	if err != nil {
		slog.Error("stats.FiftyThirtyTwenty: failed", "error", err, "user_id", userID)
		helper.RespondErrorJSON(w, http.StatusInternalServerError, err.Error())
		return
	}
	slog.Debug("stats.FiftyThirtyTwenty: success", "user_id", userID)
	helper.RespondJSON(w, http.StatusOK, result)
}

func (h *Handler) UpcomingBills(w http.ResponseWriter, r *http.Request) {
	userID, ok := helper.ResolveTargetUser(w, r)
	if !ok {
		return
	}
	slog.Debug("stats.UpcomingBills: request received", "user_id", userID)
	bills, err := h.service.UpcomingBills(r.Context(), userID)
	if err != nil {
		slog.Error("stats.UpcomingBills: failed", "error", err, "user_id", userID)
		helper.RespondErrorJSON(w, http.StatusInternalServerError, err.Error())
		return
	}
	slog.Debug("stats.UpcomingBills: success", "user_id", userID, "count", len(bills))
	helper.RespondJSON(w, http.StatusOK, bills)
}

func (h *Handler) SpendingVelocity(w http.ResponseWriter, r *http.Request) {
	userID, ok := helper.ResolveTargetUser(w, r)
	if !ok {
		return
	}
	startDate := r.URL.Query().Get("start_date")
	endDate := r.URL.Query().Get("end_date")
	if startDate == "" || endDate == "" {
		helper.RespondErrorJSON(w, http.StatusBadRequest, "start_date and end_date are required")
		return
	}
	slog.Debug("stats.SpendingVelocity: request received", "user_id", userID)
	velocity, err := h.service.SpendingVelocity(r.Context(), userID, startDate, endDate)
	if err != nil {
		slog.Error("stats.SpendingVelocity: failed", "error", err, "user_id", userID)
		helper.RespondErrorJSON(w, http.StatusInternalServerError, err.Error())
		return
	}
	slog.Debug("stats.SpendingVelocity: success", "user_id", userID)
	helper.RespondJSON(w, http.StatusOK, velocity)
}

func (h *Handler) CurrentTotalMoney(w http.ResponseWriter, r *http.Request) {
	userID, ok := helper.ResolveTargetUser(w, r)
	if !ok {
		return
	}
	slog.Debug("stats.CurrentTotalMoney: request received", "user_id", userID)
	total, err := h.service.CurrentTotalMoney(r.Context(), userID)
	if err != nil {
		slog.Error("stats.CurrentTotalMoney: failed", "error", err, "user_id", userID)
		helper.RespondErrorJSON(w, http.StatusInternalServerError, err.Error())
		return
	}
	slog.Debug("stats.CurrentTotalMoney: success", "user_id", userID)
	helper.RespondJSON(w, http.StatusOK, total)
}

func (h *Handler) PeriodBudget(w http.ResponseWriter, r *http.Request) {
	period := r.URL.Query().Get("period")
	if !validPeriodKind(period) {
		helper.RespondErrorJSON(w, http.StatusBadRequest, "period must be day, week or month")
		return
	}
	anchor := r.URL.Query().Get("date")
	if anchor != "" {
		if err := helper.ValidateDate(anchor, "date"); err != nil {
			helper.RespondErrorJSON(w, http.StatusBadRequest, err.Error())
			return
		}
	}
	userID, ok := helper.ResolveTargetUser(w, r)
	if !ok {
		return
	}
	slog.Debug("stats.PeriodBudget: request received", "user_id", userID, "period", period, "date", anchor)
	budget, err := h.service.PeriodBudget(r.Context(), userID, period, anchor)
	if err != nil {
		slog.Error("stats.PeriodBudget: failed", "error", err, "user_id", userID)
		helper.RespondErrorJSON(w, http.StatusInternalServerError, err.Error())
		return
	}
	slog.Debug("stats.PeriodBudget: success", "user_id", userID)
	helper.RespondJSON(w, http.StatusOK, budget)
}

func (h *Handler) CashFlow(w http.ResponseWriter, r *http.Request) {
	bucket := r.URL.Query().Get("bucket")
	if !validPeriodKind(bucket) {
		helper.RespondErrorJSON(w, http.StatusBadRequest, "bucket must be day, week or month")
		return
	}
	count, err := strconv.Atoi(r.URL.Query().Get("count"))
	if err != nil || count < 1 || count > MaxCashFlowBuckets {
		helper.RespondErrorJSON(w, http.StatusBadRequest, "count must be between 1 and 62")
		return
	}
	userID, ok := helper.ResolveTargetUser(w, r)
	if !ok {
		return
	}
	slog.Debug("stats.CashFlow: request received", "user_id", userID, "bucket", bucket, "count", count)
	series, err := h.service.CashFlow(r.Context(), userID, bucket, count)
	if err != nil {
		slog.Error("stats.CashFlow: failed", "error", err, "user_id", userID)
		helper.RespondErrorJSON(w, http.StatusInternalServerError, err.Error())
		return
	}
	slog.Debug("stats.CashFlow: success", "user_id", userID)
	helper.RespondJSON(w, http.StatusOK, series)
}

func (h *Handler) MonthOverMonthTrends(w http.ResponseWriter, r *http.Request) {
	userID, ok := helper.ResolveTargetUser(w, r)
	if !ok {
		return
	}
	startDate := helper.ParseQueryString(r, "start_date")
	endDate := helper.ParseQueryString(r, "end_date")
	slog.Debug("stats.MonthOverMonthTrends: request received", "user_id", userID)
	trends, err := h.service.MonthOverMonthTrends(r.Context(), userID, startDate, endDate)
	if err != nil {
		slog.Error("stats.MonthOverMonthTrends: failed", "error", err, "user_id", userID)
		helper.RespondErrorJSON(w, http.StatusInternalServerError, err.Error())
		return
	}
	slog.Debug("stats.MonthOverMonthTrends: success", "user_id", userID, "count", len(trends))
	helper.RespondJSON(w, http.StatusOK, trends)
}
