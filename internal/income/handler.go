package income

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/google/uuid"

	"github.com/ejsadiarin/corefinance/internal/helper"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var req CreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		helper.RespondErrorJSON(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if err := helper.ValidateAmount(req.Amount, "amount"); err != nil {
		helper.RespondErrorJSON(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := helper.ValidatePriority(req.Priority); err != nil {
		helper.RespondErrorJSON(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := helper.ValidateRecurringType(req.RecurringType); err != nil {
		helper.RespondErrorJSON(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := helper.ValidateStatus(req.Status); err != nil {
		helper.RespondErrorJSON(w, http.StatusBadRequest, err.Error())
		return
	}
	userID, ok := helper.GetUserID(w, r)
	if !ok {
		return
	}
	slog.Debug("income.Create: request received", "user_id", userID)
	income, err := h.service.Create(r.Context(), userID, req)
	if err != nil {
		slog.Error("income.Create: failed", "error", err, "user_id", userID)
		helper.RespondErrorJSON(w, http.StatusInternalServerError, err.Error())
		return
	}
	slog.Info("income.Create: income created", "id", income.ID, "user_id", userID)
	helper.RespondJSON(w, http.StatusCreated, income)
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	id, ok := helper.ParseUUID(w, r, "id")
	if !ok {
		return
	}
	userID, ok := helper.GetUserID(w, r)
	if !ok {
		return
	}
	slog.Debug("income.Get: request received", "id", id, "user_id", userID)
	income, err := h.service.Get(r.Context(), id, userID)
	if err != nil {
		slog.Error("income.Get: failed", "error", err, "id", id, "user_id", userID)
		helper.RespondErrorJSON(w, http.StatusNotFound, "income not found")
		return
	}
	slog.Debug("income.Get: success", "id", id, "user_id", userID)
	helper.RespondJSON(w, http.StatusOK, income)
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	userID, ok := helper.GetUserID(w, r)
	if !ok {
		return
	}
	slog.Debug("income.List: request received", "user_id", userID)
	params := ListParams{
		StartDate:     helper.ParseQueryString(r, "start_date"),
		EndDate:       helper.ParseQueryString(r, "end_date"),
		CategoryID:    helper.ParseQueryString(r, "category_id"),
		Status:        helper.ParseQueryString(r, "status"),
		RecurringType: helper.ParseQueryString(r, "recurring_type"),
		Page:          helper.ParseQueryInt(r, "page", 1),
		PageSize:      helper.ParseQueryInt(r, "page_size", 50),
	}
	incomes, err := h.service.List(r.Context(), userID, params)
	if err != nil {
		slog.Error("income.List: failed", "error", err, "user_id", userID)
		helper.RespondErrorJSON(w, http.StatusInternalServerError, err.Error())
		return
	}
	slog.Debug("income.List: success", "user_id", userID, "count", len(incomes))
	helper.RespondJSON(w, http.StatusOK, incomes)
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	id, ok := helper.ParseUUID(w, r, "id")
	if !ok {
		return
	}
	var req UpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		helper.RespondErrorJSON(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if err := helper.ValidateAmount(req.Amount, "amount"); err != nil {
		helper.RespondErrorJSON(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := helper.ValidatePriority(req.Priority); err != nil {
		helper.RespondErrorJSON(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := helper.ValidateRecurringType(req.RecurringType); err != nil {
		helper.RespondErrorJSON(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := helper.ValidateStatus(req.Status); err != nil {
		helper.RespondErrorJSON(w, http.StatusBadRequest, err.Error())
		return
	}
	userID, ok := helper.GetUserID(w, r)
	if !ok {
		return
	}
	slog.Debug("income.Update: request received", "id", id, "user_id", userID)
	income, err := h.service.Update(r.Context(), id, userID, req)
	if err != nil {
		slog.Error("income.Update: failed", "error", err, "id", id, "user_id", userID)
		helper.RespondErrorJSON(w, http.StatusInternalServerError, err.Error())
		return
	}
	slog.Info("income.Update: income updated", "id", id, "user_id", userID)
	helper.RespondJSON(w, http.StatusOK, income)
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	id, ok := helper.ParseUUID(w, r, "id")
	if !ok {
		return
	}
	userID, ok := helper.GetUserID(w, r)
	if !ok {
		return
	}
	slog.Debug("income.Delete: request received", "id", id, "user_id", userID)
	if err := h.service.Delete(r.Context(), id, userID); err != nil {
		slog.Error("income.Delete: failed", "error", err, "id", id, "user_id", userID)
		helper.RespondErrorJSON(w, http.StatusInternalServerError, err.Error())
		return
	}
	slog.Info("income.Delete: income deleted", "id", id, "user_id", userID)
	helper.RespondJSON(w, http.StatusNoContent, nil)
}

func (h *Handler) Skip(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		helper.RespondErrorJSON(w, http.StatusBadRequest, "invalid request body")
		return
	}
	id, err := uuid.Parse(req.ID)
	if err != nil {
		helper.RespondErrorJSON(w, http.StatusBadRequest, "invalid income id")
		return
	}
	userID, ok := helper.GetUserID(w, r)
	if !ok {
		return
	}
	slog.Debug("income.Skip: request received", "id", id, "user_id", userID)
	if err := h.service.Skip(r.Context(), id, userID); err != nil {
		slog.Error("income.Skip: failed", "error", err, "id", id, "user_id", userID)
		helper.RespondErrorJSON(w, http.StatusInternalServerError, err.Error())
		return
	}
	slog.Info("income.Skip: income skipped", "id", id, "user_id", userID)
	helper.RespondJSON(w, http.StatusOK, map[string]string{"status": "skipped"})
}

func (h *Handler) CheckSkipped(w http.ResponseWriter, r *http.Request) {
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
	slog.Debug("income.CheckSkipped: request received", "user_id", userID)
	skipped, err := h.service.CheckSkipped(r.Context(), userID, startDate, endDate)
	if err != nil {
		slog.Error("income.CheckSkipped: failed", "error", err, "user_id", userID)
		helper.RespondErrorJSON(w, http.StatusInternalServerError, err.Error())
		return
	}
	slog.Debug("income.CheckSkipped: success", "user_id", userID, "is_skipped", skipped)
	helper.RespondJSON(w, http.StatusOK, map[string]bool{"is_skipped": skipped})
}

func (h *Handler) GetOccurrences(w http.ResponseWriter, r *http.Request) {
	userID, ok := helper.GetUserID(w, r)
	if !ok {
		return
	}
	startDate := helper.ParseQueryString(r, "start_date")
	endDate := helper.ParseQueryString(r, "end_date")
	slog.Debug("income.GetOccurrences: request received", "user_id", userID)
	occurrences, err := h.service.Occurrences(r.Context(), userID, startDate, endDate)
	if err != nil {
		slog.Error("income.GetOccurrences: failed", "error", err, "user_id", userID)
		helper.RespondErrorJSON(w, http.StatusInternalServerError, err.Error())
		return
	}
	slog.Debug("income.GetOccurrences: success", "user_id", userID, "count", len(occurrences))
	helper.RespondJSON(w, http.StatusOK, occurrences)
}
