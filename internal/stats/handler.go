package stats

import (
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
	summary, err := h.service.Summary(r.Context(), userID, startDate, endDate)
	if err != nil {
		helper.RespondErrorJSON(w, http.StatusInternalServerError, err.Error())
		return
	}
	helper.RespondJSON(w, http.StatusOK, summary)
}

func (h *Handler) Trends(w http.ResponseWriter, r *http.Request) {
	userID, ok := helper.GetUserID(w, r)
	if !ok {
		return
	}
	startDate := helper.ParseQueryString(r, "start_date")
	endDate := helper.ParseQueryString(r, "end_date")
	trends, err := h.service.Trends(r.Context(), userID, startDate, endDate)
	if err != nil {
		helper.RespondErrorJSON(w, http.StatusInternalServerError, err.Error())
		return
	}
	helper.RespondJSON(w, http.StatusOK, trends)
}

func (h *Handler) CategoryBreakdown(w http.ResponseWriter, r *http.Request) {
	userID, ok := helper.GetUserID(w, r)
	if !ok {
		return
	}
	startDate := helper.ParseQueryString(r, "start_date")
	endDate := helper.ParseQueryString(r, "end_date")
	breakdown, err := h.service.CategoryBreakdown(r.Context(), userID, startDate, endDate)
	if err != nil {
		helper.RespondErrorJSON(w, http.StatusInternalServerError, err.Error())
		return
	}
	helper.RespondJSON(w, http.StatusOK, breakdown)
}

func (h *Handler) SavingsRate(w http.ResponseWriter, r *http.Request) {
	userID, ok := helper.GetUserID(w, r)
	if !ok {
		return
	}
	startDate := helper.ParseQueryString(r, "start_date")
	endDate := helper.ParseQueryString(r, "end_date")
	rate, err := h.service.SavingsRate(r.Context(), userID, startDate, endDate)
	if err != nil {
		helper.RespondErrorJSON(w, http.StatusInternalServerError, err.Error())
		return
	}
	helper.RespondJSON(w, http.StatusOK, rate)
}

func (h *Handler) FiftyThirtyTwenty(w http.ResponseWriter, r *http.Request) {
	userID, ok := helper.GetUserID(w, r)
	if !ok {
		return
	}
	startDate := helper.ParseQueryString(r, "start_date")
	endDate := helper.ParseQueryString(r, "end_date")
	result, err := h.service.FiftyThirtyTwenty(r.Context(), userID, startDate, endDate)
	if err != nil {
		helper.RespondErrorJSON(w, http.StatusInternalServerError, err.Error())
		return
	}
	helper.RespondJSON(w, http.StatusOK, result)
}

func (h *Handler) UpcomingBills(w http.ResponseWriter, r *http.Request) {
	userID, ok := helper.GetUserID(w, r)
	if !ok {
		return
	}
	bills, err := h.service.UpcomingBills(r.Context(), userID)
	if err != nil {
		helper.RespondErrorJSON(w, http.StatusInternalServerError, err.Error())
		return
	}
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
	velocity, err := h.service.SpendingVelocity(r.Context(), userID, startDate, endDate)
	if err != nil {
		helper.RespondErrorJSON(w, http.StatusInternalServerError, err.Error())
		return
	}
	helper.RespondJSON(w, http.StatusOK, velocity)
}

func (h *Handler) CurrentTotalMoney(w http.ResponseWriter, r *http.Request) {
	userID, ok := helper.GetUserID(w, r)
	if !ok {
		return
	}
	startDate := helper.ParseQueryString(r, "start_date")
	endDate := helper.ParseQueryString(r, "end_date")
	total, err := h.service.CurrentTotalMoney(r.Context(), userID, startDate, endDate)
	if err != nil {
		helper.RespondErrorJSON(w, http.StatusInternalServerError, err.Error())
		return
	}
	helper.RespondJSON(w, http.StatusOK, total)
}

func (h *Handler) MonthOverMonthTrends(w http.ResponseWriter, r *http.Request) {
	userID, ok := helper.GetUserID(w, r)
	if !ok {
		return
	}
	startDate := helper.ParseQueryString(r, "start_date")
	endDate := helper.ParseQueryString(r, "end_date")
	trends, err := h.service.MonthOverMonthTrends(r.Context(), userID, startDate, endDate)
	if err != nil {
		helper.RespondErrorJSON(w, http.StatusInternalServerError, err.Error())
		return
	}
	helper.RespondJSON(w, http.StatusOK, trends)
}
