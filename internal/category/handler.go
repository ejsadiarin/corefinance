package category

import (
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

func (h *Handler) ListExpense(w http.ResponseWriter, r *http.Request) {
	// A targeted read is authorized as before; the list is the same for all.
	if _, ok := helper.ResolveTargetUser(w, r); !ok {
		return
	}
	categories, err := h.service.ListExpense(r.Context())
	if err != nil {
		slog.Error("category.ListExpense: failed", "error", err)
		helper.RespondErrorJSON(w, http.StatusInternalServerError, err.Error())
		return
	}
	helper.RespondJSON(w, http.StatusOK, categories)
}

func (h *Handler) ListIncome(w http.ResponseWriter, r *http.Request) {
	if _, ok := helper.ResolveTargetUser(w, r); !ok {
		return
	}
	categories, err := h.service.ListIncome(r.Context())
	if err != nil {
		slog.Error("category.ListIncome: failed", "error", err)
		helper.RespondErrorJSON(w, http.StatusInternalServerError, err.Error())
		return
	}
	helper.RespondJSON(w, http.StatusOK, categories)
}

// Fixed answers every category write: categories are system-defined
// (ADR-0002), so owners can't create, rename or remove them.
func (h *Handler) Fixed(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Allow", "GET")
	helper.RespondErrorJSON(w, http.StatusMethodNotAllowed,
		"categories are system-defined and can't be created, changed or removed; use a tag instead")
}
