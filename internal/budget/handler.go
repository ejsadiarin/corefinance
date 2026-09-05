package budget

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

func (h *Handler) PriorityGroups(w http.ResponseWriter, r *http.Request) {
	slog.Debug("budget.Handler.PriorityGroups: called")
	groups := h.service.PriorityGroups()
	helper.RespondJSON(w, http.StatusOK, groups)
}

func (h *Handler) Remaining(w http.ResponseWriter, r *http.Request) {
	userID, ok := helper.GetUserID(w, r)
	if !ok {
		return
	}

	slog.Debug("budget.Handler.Remaining: called", "user_id", userID)

	result, err := h.service.Remaining(r.Context(), userID)
	if err != nil {
		slog.Error("budget.Handler.Remaining: failed", "error", err, "user_id", userID)
		helper.RespondErrorJSON(w, http.StatusInternalServerError, err.Error())
		return
	}

	helper.RespondJSON(w, http.StatusOK, result)
}

func (h *Handler) Export(w http.ResponseWriter, r *http.Request) {
	userID, ok := helper.GetUserID(w, r)
	if !ok {
		return
	}

	slog.Debug("budget.Handler.Export: called", "user_id", userID)

	export, err := h.service.Export(r.Context(), userID)
	if err != nil {
		slog.Error("budget.Handler.Export: failed", "error", err, "user_id", userID)
		helper.RespondErrorJSON(w, http.StatusInternalServerError, err.Error())
		return
	}

	slog.Info("budget.Handler.Export: completed", "user_id", userID)

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Disposition", "attachment; filename=budget_export.json")
	json.NewEncoder(w).Encode(export)
}

func (h *Handler) Import(w http.ResponseWriter, r *http.Request) {
	userID, ok := helper.GetUserID(w, r)
	if !ok {
		return
	}

	slog.Debug("budget.Handler.Import: called", "user_id", userID)

	var export BudgetExport
	if err := json.NewDecoder(r.Body).Decode(&export); err != nil {
		helper.RespondErrorJSON(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := h.service.Import(r.Context(), userID, &export); err != nil {
		slog.Error("budget.Handler.Import: failed", "error", err, "user_id", userID)
		helper.RespondErrorJSON(w, http.StatusInternalServerError, "failed to import: "+err.Error())
		return
	}

	slog.Info("budget.Handler.Import: completed", "user_id", userID)
	helper.RespondJSON(w, http.StatusOK, map[string]string{"status": "imported"})
}
