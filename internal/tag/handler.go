package tag

import (
	"encoding/json"
	"log/slog"
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
	slog.Debug("tag.Create: request received", "user_id", userID)
	tag, err := h.service.Create(r.Context(), userID, req)
	if err != nil {
		slog.Error("tag.Create: failed", "error", err, "user_id", userID)
		helper.RespondErrorJSON(w, http.StatusInternalServerError, err.Error())
		return
	}
	slog.Info("tag.Create: tag created", "id", tag.ID, "user_id", userID)
	helper.RespondJSON(w, http.StatusCreated, tag)
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	userID, ok := helper.GetUserID(w, r)
	if !ok {
		return
	}
	slog.Debug("tag.List: request received", "user_id", userID)
	tags, err := h.service.List(r.Context(), userID)
	if err != nil {
		slog.Error("tag.List: failed", "error", err, "user_id", userID)
		helper.RespondErrorJSON(w, http.StatusInternalServerError, err.Error())
		return
	}
	slog.Debug("tag.List: success", "user_id", userID, "count", len(tags))
	helper.RespondJSON(w, http.StatusOK, tags)
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
	if err := helper.ValidateRequired(req.Name, "name"); err != nil {
		helper.RespondErrorJSON(w, http.StatusBadRequest, err.Error())
		return
	}
	userID, ok := helper.GetUserID(w, r)
	if !ok {
		return
	}
	slog.Debug("tag.Update: request received", "id", id, "user_id", userID)
	tag, err := h.service.Update(r.Context(), id, userID, req)
	if err != nil {
		slog.Error("tag.Update: failed", "error", err, "id", id, "user_id", userID)
		helper.RespondErrorJSON(w, http.StatusInternalServerError, err.Error())
		return
	}
	slog.Info("tag.Update: tag updated", "id", id, "user_id", userID)
	helper.RespondJSON(w, http.StatusOK, tag)
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
	slog.Debug("tag.Delete: request received", "id", id, "user_id", userID)
	if err := h.service.Delete(r.Context(), id, userID); err != nil {
		slog.Error("tag.Delete: failed", "error", err, "id", id, "user_id", userID)
		helper.RespondErrorJSON(w, http.StatusInternalServerError, err.Error())
		return
	}
	slog.Info("tag.Delete: tag deleted", "id", id, "user_id", userID)
	helper.RespondJSON(w, http.StatusNoContent, nil)
}

func (h *Handler) AddTagToExpense(w http.ResponseWriter, r *http.Request) {
	expenseID, ok := helper.ParseUUID(w, r, "id")
	if !ok {
		return
	}
	var req TagAssociationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		helper.RespondErrorJSON(w, http.StatusBadRequest, "invalid request body")
		return
	}
	tagID, err := uuid.Parse(req.TagID)
	if err != nil {
		helper.RespondErrorJSON(w, http.StatusBadRequest, "invalid tag_id")
		return
	}
	slog.Debug("tag.AddTagToExpense: request received", "expense_id", expenseID, "tag_id", tagID)
	if err := h.service.AddTagToExpense(r.Context(), expenseID, tagID); err != nil {
		slog.Error("tag.AddTagToExpense: failed", "error", err, "expense_id", expenseID, "tag_id", tagID)
		helper.RespondErrorJSON(w, http.StatusInternalServerError, err.Error())
		return
	}
	slog.Info("tag.AddTagToExpense: tag added", "expense_id", expenseID, "tag_id", tagID)
	helper.RespondJSON(w, http.StatusCreated, nil)
}

func (h *Handler) RemoveTagFromExpense(w http.ResponseWriter, r *http.Request) {
	expenseID, ok := helper.ParseUUID(w, r, "id")
	if !ok {
		return
	}
	tagID, ok := helper.ParseUUID(w, r, "tag_id")
	if !ok {
		return
	}
	slog.Debug("tag.RemoveTagFromExpense: request received", "expense_id", expenseID, "tag_id", tagID)
	if err := h.service.RemoveTagFromExpense(r.Context(), expenseID, tagID); err != nil {
		slog.Error("tag.RemoveTagFromExpense: failed", "error", err, "expense_id", expenseID, "tag_id", tagID)
		helper.RespondErrorJSON(w, http.StatusInternalServerError, err.Error())
		return
	}
	slog.Info("tag.RemoveTagFromExpense: tag removed", "expense_id", expenseID, "tag_id", tagID)
	helper.RespondJSON(w, http.StatusNoContent, nil)
}

func (h *Handler) GetTagsByExpenseID(w http.ResponseWriter, r *http.Request) {
	expenseID, ok := helper.ParseUUID(w, r, "id")
	if !ok {
		return
	}
	slog.Debug("tag.GetTagsByExpenseID: request received", "expense_id", expenseID)
	tags, err := h.service.GetTagsByExpenseID(r.Context(), expenseID)
	if err != nil {
		slog.Error("tag.GetTagsByExpenseID: failed", "error", err, "expense_id", expenseID)
		helper.RespondErrorJSON(w, http.StatusInternalServerError, err.Error())
		return
	}
	slog.Debug("tag.GetTagsByExpenseID: success", "expense_id", expenseID, "count", len(tags))
	helper.RespondJSON(w, http.StatusOK, tags)
}

func (h *Handler) AddTagToIncome(w http.ResponseWriter, r *http.Request) {
	incomeID, ok := helper.ParseUUID(w, r, "id")
	if !ok {
		return
	}
	var req TagAssociationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		helper.RespondErrorJSON(w, http.StatusBadRequest, "invalid request body")
		return
	}
	tagID, err := uuid.Parse(req.TagID)
	if err != nil {
		helper.RespondErrorJSON(w, http.StatusBadRequest, "invalid tag_id")
		return
	}
	slog.Debug("tag.AddTagToIncome: request received", "income_id", incomeID, "tag_id", tagID)
	if err := h.service.AddTagToIncome(r.Context(), incomeID, tagID); err != nil {
		slog.Error("tag.AddTagToIncome: failed", "error", err, "income_id", incomeID, "tag_id", tagID)
		helper.RespondErrorJSON(w, http.StatusInternalServerError, err.Error())
		return
	}
	slog.Info("tag.AddTagToIncome: tag added", "income_id", incomeID, "tag_id", tagID)
	helper.RespondJSON(w, http.StatusCreated, nil)
}

func (h *Handler) RemoveTagFromIncome(w http.ResponseWriter, r *http.Request) {
	incomeID, ok := helper.ParseUUID(w, r, "id")
	if !ok {
		return
	}
	tagID, ok := helper.ParseUUID(w, r, "tag_id")
	if !ok {
		return
	}
	slog.Debug("tag.RemoveTagFromIncome: request received", "income_id", incomeID, "tag_id", tagID)
	if err := h.service.RemoveTagFromIncome(r.Context(), incomeID, tagID); err != nil {
		slog.Error("tag.RemoveTagFromIncome: failed", "error", err, "income_id", incomeID, "tag_id", tagID)
		helper.RespondErrorJSON(w, http.StatusInternalServerError, err.Error())
		return
	}
	slog.Info("tag.RemoveTagFromIncome: tag removed", "income_id", incomeID, "tag_id", tagID)
	helper.RespondJSON(w, http.StatusNoContent, nil)
}

func (h *Handler) GetTagsByIncomeID(w http.ResponseWriter, r *http.Request) {
	incomeID, ok := helper.ParseUUID(w, r, "id")
	if !ok {
		return
	}
	slog.Debug("tag.GetTagsByIncomeID: request received", "income_id", incomeID)
	tags, err := h.service.GetTagsByIncomeID(r.Context(), incomeID)
	if err != nil {
		slog.Error("tag.GetTagsByIncomeID: failed", "error", err, "income_id", incomeID)
		helper.RespondErrorJSON(w, http.StatusInternalServerError, err.Error())
		return
	}
	slog.Debug("tag.GetTagsByIncomeID: success", "income_id", incomeID, "count", len(tags))
	helper.RespondJSON(w, http.StatusOK, tags)
}
