package expense

import (
	"encoding/json"
	"net/http"

	"github.com/ejsadiarin/corefinance/internal/helper"
	"github.com/google/uuid"
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
		helper.RespondError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if err := helper.ValidateAmount(req.Amount, "amount"); err != nil {
		helper.RespondError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := helper.ValidatePriority(req.Priority); err != nil {
		helper.RespondError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := helper.ValidateRecurringType(req.RecurringType); err != nil {
		helper.RespondError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := helper.ValidateStatus(req.Status); err != nil {
		helper.RespondError(w, http.StatusBadRequest, err.Error())
		return
	}
	userID, ok := helper.GetUserID(w, r)
	if !ok {
		return
	}
	expense, err := h.service.Create(r.Context(), userID, req)
	if err != nil {
		helper.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	helper.Respond(w, http.StatusCreated, expense)
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
	expense, err := h.service.Get(r.Context(), id, userID)
	if err != nil {
		helper.RespondError(w, http.StatusNotFound, "expense not found")
		return
	}
	helper.Respond(w, http.StatusOK, expense)
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	userID, ok := helper.GetUserID(w, r)
	if !ok {
		return
	}
	params := ListParams{
		StartDate:     helper.ParseQueryString(r, "start_date"),
		EndDate:       helper.ParseQueryString(r, "end_date"),
		CategoryID:    helper.ParseQueryString(r, "category_id"),
		Priority:      helper.ParseQueryString(r, "priority"),
		Status:        helper.ParseQueryString(r, "status"),
		RecurringType: helper.ParseQueryString(r, "recurring_type"),
		Page:          helper.ParseQueryInt(r, "page", 1),
		PageSize:      helper.ParseQueryInt(r, "page_size", 50),
	}
	expenses, err := h.service.List(r.Context(), userID, params)
	if err != nil {
		helper.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	helper.Respond(w, http.StatusOK, expenses)
}

func (h *Handler) Search(w http.ResponseWriter, r *http.Request) {
	userID, ok := helper.GetUserID(w, r)
	if !ok {
		return
	}
	query := r.URL.Query().Get("q")
	if query == "" {
		helper.RespondError(w, http.StatusBadRequest, "query parameter 'q' is required")
		return
	}
	params := SearchParams{
		Query:    query,
		Page:     helper.ParseQueryInt(r, "page", 1),
		PageSize: helper.ParseQueryInt(r, "page_size", 50),
	}
	expenses, err := h.service.Search(r.Context(), userID, params)
	if err != nil {
		helper.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	helper.Respond(w, http.StatusOK, expenses)
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	id, ok := helper.ParseUUID(w, r, "id")
	if !ok {
		return
	}
	var req UpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		helper.RespondError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if err := helper.ValidateAmount(req.Amount, "amount"); err != nil {
		helper.RespondError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := helper.ValidatePriority(req.Priority); err != nil {
		helper.RespondError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := helper.ValidateRecurringType(req.RecurringType); err != nil {
		helper.RespondError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := helper.ValidateStatus(req.Status); err != nil {
		helper.RespondError(w, http.StatusBadRequest, err.Error())
		return
	}
	userID, ok := helper.GetUserID(w, r)
	if !ok {
		return
	}
	expense, err := h.service.Update(r.Context(), id, userID, req)
	if err != nil {
		helper.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	helper.Respond(w, http.StatusOK, expense)
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
	if err := h.service.Delete(r.Context(), id, userID); err != nil {
		helper.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	helper.Respond(w, http.StatusNoContent, nil)
}

func (h *Handler) Skip(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		helper.RespondError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	id, err := uuid.Parse(req.ID)
	if err != nil {
		helper.RespondError(w, http.StatusBadRequest, "invalid expense id")
		return
	}
	userID, ok := helper.GetUserID(w, r)
	if !ok {
		return
	}
	if err := h.service.Skip(r.Context(), id, userID); err != nil {
		helper.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	helper.Respond(w, http.StatusOK, map[string]string{"status": "skipped"})
}

func (h *Handler) CheckSkipped(w http.ResponseWriter, r *http.Request) {
	userID, ok := helper.GetUserID(w, r)
	if !ok {
		return
	}
	startDate := r.URL.Query().Get("start_date")
	endDate := r.URL.Query().Get("end_date")
	if startDate == "" || endDate == "" {
		helper.RespondError(w, http.StatusBadRequest, "start_date and end_date are required")
		return
	}
	skipped, err := h.service.CheckSkipped(r.Context(), userID, startDate, endDate)
	if err != nil {
		helper.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	helper.Respond(w, http.StatusOK, map[string]bool{"is_skipped": skipped})
}
