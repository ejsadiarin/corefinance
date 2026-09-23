package service

import (
	"log/slog"
	"net/http"

	"github.com/ejsadiarin/corefinance/internal/helper"
)

type Handler struct {
	service *Service
}

// NewHandler builds a Handler over a Service.
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// ActiveRecurringRules serves GET /internal/recurring/active. It takes no
// user identity: the route's scope middleware already established a
// service identity with the required scope.
func (h *Handler) ActiveRecurringRules(w http.ResponseWriter, r *http.Request) {
	expenses, incomes, err := h.service.ActiveRecurringRules(r.Context())
	if err != nil {
		slog.Error("service.ActiveRecurringRules: failed", "error", err)
		helper.RespondErrorJSON(w, http.StatusInternalServerError, "Failed to list active rules")
		return
	}
	helper.RespondJSON(w, http.StatusOK, map[string]any{
		"expense_rules": expenses,
		"income_rules":  incomes,
	})
}
