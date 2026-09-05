package category

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

// --- Expense Categories ---

func (h *Handler) CreateExpense(w http.ResponseWriter, r *http.Request) {
	var req CreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		helper.RespondErrorJSON(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if err := helper.ValidateRequired(req.Name, "name"); err != nil {
		helper.RespondErrorJSON(w, http.StatusBadRequest, err.Error())
		return
	}
	userID, ok := helper.GetUserID(w, r)
	if !ok {
		return
	}
	category, err := h.service.CreateExpense(r.Context(), userID, req)
	if err != nil {
		helper.RespondErrorJSON(w, http.StatusInternalServerError, err.Error())
		return
	}
	helper.RespondJSON(w, http.StatusCreated, category)
}

func (h *Handler) ListExpense(w http.ResponseWriter, r *http.Request) {
	userID, ok := helper.GetUserID(w, r)
	if !ok {
		return
	}
	categories, err := h.service.ListExpense(r.Context(), userID)
	if err != nil {
		helper.RespondErrorJSON(w, http.StatusInternalServerError, err.Error())
		return
	}
	helper.RespondJSON(w, http.StatusOK, categories)
}

func (h *Handler) UpdateExpense(w http.ResponseWriter, r *http.Request) {
	id, ok := helper.ParseUUID(w, r, "id")
	if !ok {
		return
	}
	var req UpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		helper.RespondErrorJSON(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if err := helper.ValidateRequired(req.Name, "name"); err != nil {
		helper.RespondErrorJSON(w, http.StatusBadRequest, err.Error())
		return
	}
	userID, ok := helper.GetUserID(w, r)
	if !ok {
		return
	}
	category, err := h.service.UpdateExpense(r.Context(), id, userID, req)
	if err != nil {
		helper.RespondErrorJSON(w, http.StatusInternalServerError, err.Error())
		return
	}
	helper.RespondJSON(w, http.StatusOK, category)
}

func (h *Handler) DeleteExpense(w http.ResponseWriter, r *http.Request) {
	id, ok := helper.ParseUUID(w, r, "id")
	if !ok {
		return
	}
	userID, ok := helper.GetUserID(w, r)
	if !ok {
		return
	}
	if err := h.service.DeleteExpense(r.Context(), id, userID); err != nil {
		helper.RespondErrorJSON(w, http.StatusInternalServerError, err.Error())
		return
	}
	helper.RespondJSON(w, http.StatusNoContent, nil)
}

// --- Income Categories ---

func (h *Handler) CreateIncome(w http.ResponseWriter, r *http.Request) {
	var req CreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		helper.RespondErrorJSON(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if err := helper.ValidateRequired(req.Name, "name"); err != nil {
		helper.RespondErrorJSON(w, http.StatusBadRequest, err.Error())
		return
	}
	userID, ok := helper.GetUserID(w, r)
	if !ok {
		return
	}
	category, err := h.service.CreateIncome(r.Context(), userID, req)
	if err != nil {
		helper.RespondErrorJSON(w, http.StatusInternalServerError, err.Error())
		return
	}
	helper.RespondJSON(w, http.StatusCreated, category)
}

func (h *Handler) ListIncome(w http.ResponseWriter, r *http.Request) {
	userID, ok := helper.GetUserID(w, r)
	if !ok {
		return
	}
	categories, err := h.service.ListIncome(r.Context(), userID)
	if err != nil {
		helper.RespondErrorJSON(w, http.StatusInternalServerError, err.Error())
		return
	}
	helper.RespondJSON(w, http.StatusOK, categories)
}

func (h *Handler) UpdateIncome(w http.ResponseWriter, r *http.Request) {
	id, ok := helper.ParseUUID(w, r, "id")
	if !ok {
		return
	}
	var req UpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		helper.RespondErrorJSON(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if err := helper.ValidateRequired(req.Name, "name"); err != nil {
		helper.RespondErrorJSON(w, http.StatusBadRequest, err.Error())
		return
	}
	userID, ok := helper.GetUserID(w, r)
	if !ok {
		return
	}
	category, err := h.service.UpdateIncome(r.Context(), id, userID, req)
	if err != nil {
		helper.RespondErrorJSON(w, http.StatusInternalServerError, err.Error())
		return
	}
	helper.RespondJSON(w, http.StatusOK, category)
}

func (h *Handler) DeleteIncome(w http.ResponseWriter, r *http.Request) {
	id, ok := helper.ParseUUID(w, r, "id")
	if !ok {
		return
	}
	userID, ok := helper.GetUserID(w, r)
	if !ok {
		return
	}
	if err := h.service.DeleteIncome(r.Context(), id, userID); err != nil {
		helper.RespondErrorJSON(w, http.StatusInternalServerError, err.Error())
		return
	}
	helper.RespondJSON(w, http.StatusNoContent, nil)
}
