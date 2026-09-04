package tag

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
	if err := helper.ValidateRequired(req.Name, "name"); err != nil {
		helper.RespondError(w, http.StatusBadRequest, err.Error())
		return
	}
	userID, ok := helper.GetUserID(w, r)
	if !ok {
		return
	}
	tag, err := h.service.Create(r.Context(), userID, req)
	if err != nil {
		helper.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	helper.Respond(w, http.StatusCreated, tag)
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	userID, ok := helper.GetUserID(w, r)
	if !ok {
		return
	}
	tags, err := h.service.List(r.Context(), userID)
	if err != nil {
		helper.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	helper.Respond(w, http.StatusOK, tags)
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
	if err := helper.ValidateRequired(req.Name, "name"); err != nil {
		helper.RespondError(w, http.StatusBadRequest, err.Error())
		return
	}
	userID, ok := helper.GetUserID(w, r)
	if !ok {
		return
	}
	tag, err := h.service.Update(r.Context(), id, userID, req)
	if err != nil {
		helper.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	helper.Respond(w, http.StatusOK, tag)
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

func (h *Handler) AddTagToExpense(w http.ResponseWriter, r *http.Request) {
	expenseID, ok := helper.ParseUUID(w, r, "id")
	if !ok {
		return
	}
	var req TagAssociationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		helper.RespondError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	tagID, err := uuid.Parse(req.TagID)
	if err != nil {
		helper.RespondError(w, http.StatusBadRequest, "invalid tag_id")
		return
	}
	if err := h.service.AddTagToExpense(r.Context(), expenseID, tagID); err != nil {
		helper.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	helper.Respond(w, http.StatusCreated, nil)
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
	if err := h.service.RemoveTagFromExpense(r.Context(), expenseID, tagID); err != nil {
		helper.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	helper.Respond(w, http.StatusNoContent, nil)
}

func (h *Handler) GetTagsByExpenseID(w http.ResponseWriter, r *http.Request) {
	expenseID, ok := helper.ParseUUID(w, r, "id")
	if !ok {
		return
	}
	tags, err := h.service.GetTagsByExpenseID(r.Context(), expenseID)
	if err != nil {
		helper.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	helper.Respond(w, http.StatusOK, tags)
}

func (h *Handler) AddTagToIncome(w http.ResponseWriter, r *http.Request) {
	incomeID, ok := helper.ParseUUID(w, r, "id")
	if !ok {
		return
	}
	var req TagAssociationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		helper.RespondError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	tagID, err := uuid.Parse(req.TagID)
	if err != nil {
		helper.RespondError(w, http.StatusBadRequest, "invalid tag_id")
		return
	}
	if err := h.service.AddTagToIncome(r.Context(), incomeID, tagID); err != nil {
		helper.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	helper.Respond(w, http.StatusCreated, nil)
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
	if err := h.service.RemoveTagFromIncome(r.Context(), incomeID, tagID); err != nil {
		helper.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	helper.Respond(w, http.StatusNoContent, nil)
}

func (h *Handler) GetTagsByIncomeID(w http.ResponseWriter, r *http.Request) {
	incomeID, ok := helper.ParseUUID(w, r, "id")
	if !ok {
		return
	}
	tags, err := h.service.GetTagsByIncomeID(r.Context(), incomeID)
	if err != nil {
		helper.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	helper.Respond(w, http.StatusOK, tags)
}
