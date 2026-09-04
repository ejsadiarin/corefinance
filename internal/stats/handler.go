package stats

import (
	"encoding/json"
	"net/http"

	"github.com/ejsadiarin/corefinance/internal/auth"
	"github.com/google/uuid"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) respond(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if data != nil {
		json.NewEncoder(w).Encode(data)
	}
}

func (h *Handler) respondError(w http.ResponseWriter, status int, message string) {
	h.respond(w, status, map[string]string{"error": message})
}

func (h *Handler) getUserID(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	userID, ok := auth.GetUserID(r)
	if !ok {
		h.respondError(w, http.StatusUnauthorized, "X-User-ID header is required")
		return uuid.Nil, false
	}
	return userID, true
}

func (h *Handler) parseQueryString(r *http.Request, key string) *string {
	val := r.URL.Query().Get(key)
	if val == "" {
		return nil
	}
	return &val
}

func (h *Handler) Summary(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.getUserID(w, r)
	if !ok {
		return
	}
	startDate := h.parseQueryString(r, "start_date")
	endDate := h.parseQueryString(r, "end_date")
	summary, err := h.service.Summary(r.Context(), userID, startDate, endDate)
	if err != nil {
		h.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	h.respond(w, http.StatusOK, summary)
}

func (h *Handler) Trends(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.getUserID(w, r)
	if !ok {
		return
	}
	startDate := h.parseQueryString(r, "start_date")
	endDate := h.parseQueryString(r, "end_date")
	trends, err := h.service.Trends(r.Context(), userID, startDate, endDate)
	if err != nil {
		h.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	h.respond(w, http.StatusOK, trends)
}

func (h *Handler) CategoryBreakdown(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.getUserID(w, r)
	if !ok {
		return
	}
	startDate := h.parseQueryString(r, "start_date")
	endDate := h.parseQueryString(r, "end_date")
	breakdown, err := h.service.CategoryBreakdown(r.Context(), userID, startDate, endDate)
	if err != nil {
		h.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	h.respond(w, http.StatusOK, breakdown)
}

func (h *Handler) SavingsRate(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.getUserID(w, r)
	if !ok {
		return
	}
	startDate := h.parseQueryString(r, "start_date")
	endDate := h.parseQueryString(r, "end_date")
	rate, err := h.service.SavingsRate(r.Context(), userID, startDate, endDate)
	if err != nil {
		h.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	h.respond(w, http.StatusOK, rate)
}

func (h *Handler) FiftyThirtyTwenty(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.getUserID(w, r)
	if !ok {
		return
	}
	startDate := h.parseQueryString(r, "start_date")
	endDate := h.parseQueryString(r, "end_date")
	result, err := h.service.FiftyThirtyTwenty(r.Context(), userID, startDate, endDate)
	if err != nil {
		h.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	h.respond(w, http.StatusOK, result)
}

func (h *Handler) UpcomingBills(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.getUserID(w, r)
	if !ok {
		return
	}
	bills, err := h.service.UpcomingBills(r.Context(), userID)
	if err != nil {
		h.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	h.respond(w, http.StatusOK, bills)
}

func (h *Handler) SpendingVelocity(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.getUserID(w, r)
	if !ok {
		return
	}
	startDate := r.URL.Query().Get("start_date")
	endDate := r.URL.Query().Get("end_date")
	if startDate == "" || endDate == "" {
		h.respondError(w, http.StatusBadRequest, "start_date and end_date are required")
		return
	}
	velocity, err := h.service.SpendingVelocity(r.Context(), userID, startDate, endDate)
	if err != nil {
		h.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	h.respond(w, http.StatusOK, velocity)
}

func (h *Handler) CurrentTotalMoney(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.getUserID(w, r)
	if !ok {
		return
	}
	startDate := h.parseQueryString(r, "start_date")
	endDate := h.parseQueryString(r, "end_date")
	total, err := h.service.CurrentTotalMoney(r.Context(), userID, startDate, endDate)
	if err != nil {
		h.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	h.respond(w, http.StatusOK, total)
}

func (h *Handler) MonthOverMonthTrends(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.getUserID(w, r)
	if !ok {
		return
	}
	startDate := h.parseQueryString(r, "start_date")
	endDate := h.parseQueryString(r, "end_date")
	trends, err := h.service.MonthOverMonthTrends(r.Context(), userID, startDate, endDate)
	if err != nil {
		h.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	h.respond(w, http.StatusOK, trends)
}
