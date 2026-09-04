package recurring

import (
	"encoding/json"
	"net/http"

	"github.com/ejsadiarin/corefinance/internal/auth"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) respond(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if data != nil {
		json.NewEncoder(w).Encode(data)
	}
}

func (h *Handler) respondError(w http.ResponseWriter, status int, message string) {
	h.respond(w, status, map[string]string{"error": message})
}

func (h *Handler) getUserID(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	userID, ok := auth.GetUserID(r)
	if !ok {
		h.respondError(w, http.StatusUnauthorized, "X-User-ID header is required")
		return uuid.Nil, false
	}
	return userID, true
}

func (h *Handler) parseUUID(w http.ResponseWriter, r *http.Request, param string) (uuid.UUID, bool) {
	id, err := uuid.Parse(chi.URLParam(r, param))
	if err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid "+param)
		return uuid.Nil, false
	}
	return id, true
}

// --- Income Rules ---

func (h *Handler) CreateIncomeRule(w http.ResponseWriter, r *http.Request) {
	var req CreateIncomeRuleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	userID, ok := h.getUserID(w, r)
	if !ok {
		return
	}
	rule, err := h.service.CreateIncomeRule(r.Context(), userID, req)
	if err != nil {
		h.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	h.respond(w, http.StatusCreated, rule)
}

func (h *Handler) ListIncomeRules(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.getUserID(w, r)
	if !ok {
		return
	}
	rules, err := h.service.ListIncomeRules(r.Context(), userID)
	if err != nil {
		h.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	h.respond(w, http.StatusOK, rules)
}

func (h *Handler) GetIncomeRule(w http.ResponseWriter, r *http.Request) {
	id, ok := h.parseUUID(w, r, "id")
	if !ok {
		return
	}
	userID, ok := h.getUserID(w, r)
	if !ok {
		return
	}
	rule, err := h.service.GetIncomeRule(r.Context(), id, userID)
	if err != nil {
		h.respondError(w, http.StatusNotFound, "rule not found")
		return
	}
	h.respond(w, http.StatusOK, rule)
}

func (h *Handler) UpdateIncomeRule(w http.ResponseWriter, r *http.Request) {
	id, ok := h.parseUUID(w, r, "id")
	if !ok {
		return
	}
	var req UpdateIncomeRuleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	userID, ok := h.getUserID(w, r)
	if !ok {
		return
	}
	rule, err := h.service.UpdateIncomeRule(r.Context(), id, userID, req)
	if err != nil {
		h.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	h.respond(w, http.StatusOK, rule)
}

func (h *Handler) DeleteIncomeRule(w http.ResponseWriter, r *http.Request) {
	id, ok := h.parseUUID(w, r, "id")
	if !ok {
		return
	}
	userID, ok := h.getUserID(w, r)
	if !ok {
		return
	}
	if err := h.service.DeleteIncomeRule(r.Context(), id, userID); err != nil {
		h.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	h.respond(w, http.StatusNoContent, nil)
}

// --- Expense Rules (read-only) ---

func (h *Handler) ListExpenseRules(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.getUserID(w, r)
	if !ok {
		return
	}
	rules, err := h.service.ListExpenseRules(r.Context(), userID)
	if err != nil {
		h.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	h.respond(w, http.StatusOK, rules)
}
