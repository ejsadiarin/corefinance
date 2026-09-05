package category

import (
	"encoding/json"
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
	slog.Debug("category.CreateExpense: request received", "user_id", userID)
	category, err := h.service.CreateExpense(r.Context(), userID, req)
	if err != nil {
		slog.Error("category.CreateExpense: failed", "error", err, "user_id", userID)
		helper.RespondErrorJSON(w, http.StatusInternalServerError, err.Error())
		return
	}
	slog.Info("category.CreateExpense: category created", "id", category.ID, "user_id", userID)
	helper.RespondJSON(w, http.StatusCreated, category)
}

func (h *Handler) ListExpense(w http.ResponseWriter, r *http.Request) {
	userID, ok := helper.GetUserID(w, r)
	if !ok {
		return
	}
	slog.Debug("category.ListExpense: request received", "user_id", userID)
	categories, err := h.service.ListExpense(r.Context(), userID)
	if err != nil {
		slog.Error("category.ListExpense: failed", "error", err, "user_id", userID)
		helper.RespondErrorJSON(w, http.StatusInternalServerError, err.Error())
		return
	}
	slog.Debug("category.ListExpense: success", "user_id", userID, "count", len(categories))
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
	slog.Debug("category.UpdateExpense: request received", "id", id, "user_id", userID)
	category, err := h.service.UpdateExpense(r.Context(), id, userID, req)
	if err != nil {
		slog.Error("category.UpdateExpense: failed", "error", err, "id", id, "user_id", userID)
		helper.RespondErrorJSON(w, http.StatusInternalServerError, err.Error())
		return
	}
	slog.Info("category.UpdateExpense: category updated", "id", id, "user_id", userID)
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
	slog.Debug("category.DeleteExpense: request received", "id", id, "user_id", userID)
	if err := h.service.DeleteExpense(r.Context(), id, userID); err != nil {
		slog.Error("category.DeleteExpense: failed", "error", err, "id", id, "user_id", userID)
		helper.RespondErrorJSON(w, http.StatusInternalServerError, err.Error())
		return
	}
	slog.Info("category.DeleteExpense: category deleted", "id", id, "user_id", userID)
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
	slog.Debug("category.CreateIncome: request received", "user_id", userID)
	category, err := h.service.CreateIncome(r.Context(), userID, req)
	if err != nil {
		slog.Error("category.CreateIncome: failed", "error", err, "user_id", userID)
		helper.RespondErrorJSON(w, http.StatusInternalServerError, err.Error())
		return
	}
	slog.Info("category.CreateIncome: category created", "id", category.ID, "user_id", userID)
	helper.RespondJSON(w, http.StatusCreated, category)
}

func (h *Handler) ListIncome(w http.ResponseWriter, r *http.Request) {
	userID, ok := helper.GetUserID(w, r)
	if !ok {
		return
	}
	slog.Debug("category.ListIncome: request received", "user_id", userID)
	categories, err := h.service.ListIncome(r.Context(), userID)
	if err != nil {
		slog.Error("category.ListIncome: failed", "error", err, "user_id", userID)
		helper.RespondErrorJSON(w, http.StatusInternalServerError, err.Error())
		return
	}
	slog.Debug("category.ListIncome: success", "user_id", userID, "count", len(categories))
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
	slog.Debug("category.UpdateIncome: request received", "id", id, "user_id", userID)
	category, err := h.service.UpdateIncome(r.Context(), id, userID, req)
	if err != nil {
		slog.Error("category.UpdateIncome: failed", "error", err, "id", id, "user_id", userID)
		helper.RespondErrorJSON(w, http.StatusInternalServerError, err.Error())
		return
	}
	slog.Info("category.UpdateIncome: category updated", "id", id, "user_id", userID)
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
	slog.Debug("category.DeleteIncome: request received", "id", id, "user_id", userID)
	if err := h.service.DeleteIncome(r.Context(), id, userID); err != nil {
		slog.Error("category.DeleteIncome: failed", "error", err, "id", id, "user_id", userID)
		helper.RespondErrorJSON(w, http.StatusInternalServerError, err.Error())
		return
	}
	slog.Info("category.DeleteIncome: category deleted", "id", id, "user_id", userID)
	helper.RespondJSON(w, http.StatusNoContent, nil)
}
