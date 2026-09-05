package stats

import (
	"log/slog"
	"net/http"

	"github.com/ejsadiarin/corefinance/internal/helper"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) Summary(w http.ResponseWriter, r *http.Request) {
	userID, ok := helper.GetUserID(w, r)
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
	userID, ok := helper.GetUserID(w, r)
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
	userID, ok := helper.GetUserID(w, r)
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
	userID, ok := helper.GetUserID(w, r)
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
	userID, ok := helper.GetUserID(w, r)
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
	userID, ok := helper.GetUserID(w, r)
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
	userID, ok := helper.GetUserID(w, r)
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
	userID, ok := helper.GetUserID(w, r)
	if !ok {
		return
	}
	startDate := helper.ParseQueryString(r, "start_date")
	endDate := helper.ParseQueryString(r, "end_date")
	slog.Debug("stats.CurrentTotalMoney: request received", "user_id", userID)
	total, err := h.service.CurrentTotalMoney(r.Context(), userID, startDate, endDate)
	if err != nil {
		slog.Error("stats.CurrentTotalMoney: failed", "error", err, "user_id", userID)
		helper.RespondErrorJSON(w, http.StatusInternalServerError, err.Error())
		return
	}
	slog.Debug("stats.CurrentTotalMoney: success", "user_id", userID)
	helper.RespondJSON(w, http.StatusOK, total)
}

func (h *Handler) MonthOverMonthTrends(w http.ResponseWriter, r *http.Request) {
	userID, ok := helper.GetUserID(w, r)
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
