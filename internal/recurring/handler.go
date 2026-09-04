package recurring

import (
	"encoding/json"
	"net/http"

	"github.com/ejsadiarin/corefinance/internal/helper"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// --- Income Rules ---

func (h *Handler) CreateIncomeRule(w http.ResponseWriter, r *http.Request) {
	var req CreateIncomeRuleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		helper.RespondError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	userID, ok := helper.GetUserID(w, r)
	if !ok {
		return
	}
	rule, err := h.service.CreateIncomeRule(r.Context(), userID, req)
	if err != nil {
		helper.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	helper.Respond(w, http.StatusCreated, rule)
}

func (h *Handler) ListIncomeRules(w http.ResponseWriter, r *http.Request) {
	userID, ok := helper.GetUserID(w, r)
	if !ok {
		return
	}
	rules, err := h.service.ListIncomeRules(r.Context(), userID)
	if err != nil {
		helper.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	helper.Respond(w, http.StatusOK, rules)
}

func (h *Handler) GetIncomeRule(w http.ResponseWriter, r *http.Request) {
	id, ok := helper.ParseUUID(w, r, "id")
	if !ok {
		return
	}
	userID, ok := helper.GetUserID(w, r)
	if !ok {
		return
	}
	rule, err := h.service.GetIncomeRule(r.Context(), id, userID)
	if err != nil {
		helper.RespondError(w, http.StatusNotFound, "rule not found")
		return
	}
	helper.Respond(w, http.StatusOK, rule)
}

func (h *Handler) UpdateIncomeRule(w http.ResponseWriter, r *http.Request) {
	id, ok := helper.ParseUUID(w, r, "id")
	if !ok {
		return
	}
	var req UpdateIncomeRuleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		helper.RespondError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	userID, ok := helper.GetUserID(w, r)
	if !ok {
		return
	}
	rule, err := h.service.UpdateIncomeRule(r.Context(), id, userID, req)
	if err != nil {
		helper.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	helper.Respond(w, http.StatusOK, rule)
}

func (h *Handler) DeleteIncomeRule(w http.ResponseWriter, r *http.Request) {
	id, ok := helper.ParseUUID(w, r, "id")
	if !ok {
		return
	}
	userID, ok := helper.GetUserID(w, r)
	if !ok {
		return
	}
	if err := h.service.DeleteIncomeRule(r.Context(), id, userID); err != nil {
		helper.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	helper.Respond(w, http.StatusNoContent, nil)
}

// --- Expense Rules ---

func (h *Handler) CreateExpenseRule(w http.ResponseWriter, r *http.Request) {
	var req CreateExpenseRuleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		helper.RespondError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	userID, ok := helper.GetUserID(w, r)
	if !ok {
		return
	}
	rule, err := h.service.CreateExpenseRule(r.Context(), userID, req)
	if err != nil {
		helper.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	helper.Respond(w, http.StatusCreated, rule)
}

func (h *Handler) ListExpenseRules(w http.ResponseWriter, r *http.Request) {
	userID, ok := helper.GetUserID(w, r)
	if !ok {
		return
	}
	rules, err := h.service.ListExpenseRules(r.Context(), userID)
	if err != nil {
		helper.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	helper.Respond(w, http.StatusOK, rules)
}

func (h *Handler) GetExpenseRule(w http.ResponseWriter, r *http.Request) {
	id, ok := helper.ParseUUID(w, r, "id")
	if !ok {
		return
	}
	userID, ok := helper.GetUserID(w, r)
	if !ok {
		return
	}
	rule, err := h.service.GetExpenseRule(r.Context(), id, userID)
	if err != nil {
		helper.RespondError(w, http.StatusNotFound, "rule not found")
		return
	}
	helper.Respond(w, http.StatusOK, rule)
}

func (h *Handler) UpdateExpenseRule(w http.ResponseWriter, r *http.Request) {
	id, ok := helper.ParseUUID(w, r, "id")
	if !ok {
		return
	}
	var req UpdateExpenseRuleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		helper.RespondError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	userID, ok := helper.GetUserID(w, r)
	if !ok {
		return
	}
	rule, err := h.service.UpdateExpenseRule(r.Context(), id, userID, req)
	if err != nil {
		helper.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	helper.Respond(w, http.StatusOK, rule)
}

func (h *Handler) DeleteExpenseRule(w http.ResponseWriter, r *http.Request) {
	id, ok := helper.ParseUUID(w, r, "id")
	if !ok {
		return
	}
	userID, ok := helper.GetUserID(w, r)
	if !ok {
		return
	}
	if err := h.service.DeleteExpenseRule(r.Context(), id, userID); err != nil {
		helper.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	helper.Respond(w, http.StatusNoContent, nil)
}
