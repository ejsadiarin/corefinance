package category

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

func (h *Handler) parseUUID(w http.ResponseWriter, r *http.Request, param string) (uuid.UUID, bool) {
	id, err := uuid.Parse(chi.URLParam(r, param))
	if err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid "+param)
		return uuid.Nil, false
	}
	return id, true
}

func (h *Handler) getUserID(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	userID, ok := auth.GetUserID(r)
	if !ok {
		h.respondError(w, http.StatusUnauthorized, "X-User-ID header is required")
		return uuid.Nil, false
	}
	return userID, true
}

// --- Expense Categories ---

func (h *Handler) CreateExpense(w http.ResponseWriter, r *http.Request) {
	var req CreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	userID, ok := h.getUserID(w, r)
	if !ok {
		return
	}
	category, err := h.service.CreateExpense(r.Context(), userID, req)
	if err != nil {
		h.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	h.respond(w, http.StatusCreated, category)
}

func (h *Handler) ListExpense(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.getUserID(w, r)
	if !ok {
		return
	}
	categories, err := h.service.ListExpense(r.Context(), userID)
	if err != nil {
		h.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	h.respond(w, http.StatusOK, categories)
}

func (h *Handler) UpdateExpense(w http.ResponseWriter, r *http.Request) {
	id, ok := h.parseUUID(w, r, "id")
	if !ok {
		return
	}
	var req UpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	userID, ok := h.getUserID(w, r)
	if !ok {
		return
	}
	category, err := h.service.UpdateExpense(r.Context(), id, userID, req)
	if err != nil {
		h.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	h.respond(w, http.StatusOK, category)
}

func (h *Handler) DeleteExpense(w http.ResponseWriter, r *http.Request) {
	id, ok := h.parseUUID(w, r, "id")
	if !ok {
		return
	}
	userID, ok := h.getUserID(w, r)
	if !ok {
		return
	}
	if err := h.service.DeleteExpense(r.Context(), id, userID); err != nil {
		h.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	h.respond(w, http.StatusNoContent, nil)
}

// --- Income Categories ---

func (h *Handler) CreateIncome(w http.ResponseWriter, r *http.Request) {
	var req CreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	userID, ok := h.getUserID(w, r)
	if !ok {
		return
	}
	category, err := h.service.CreateIncome(r.Context(), userID, req)
	if err != nil {
		h.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	h.respond(w, http.StatusCreated, category)
}

func (h *Handler) ListIncome(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.getUserID(w, r)
	if !ok {
		return
	}
	categories, err := h.service.ListIncome(r.Context(), userID)
	if err != nil {
		h.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	h.respond(w, http.StatusOK, categories)
}

func (h *Handler) UpdateIncome(w http.ResponseWriter, r *http.Request) {
	id, ok := h.parseUUID(w, r, "id")
	if !ok {
		return
	}
	var req UpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	userID, ok := h.getUserID(w, r)
	if !ok {
		return
	}
	category, err := h.service.UpdateIncome(r.Context(), id, userID, req)
	if err != nil {
		h.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	h.respond(w, http.StatusOK, category)
}

func (h *Handler) DeleteIncome(w http.ResponseWriter, r *http.Request) {
	id, ok := h.parseUUID(w, r, "id")
	if !ok {
		return
	}
	userID, ok := h.getUserID(w, r)
	if !ok {
		return
	}
	if err := h.service.DeleteIncome(r.Context(), id, userID); err != nil {
		h.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	h.respond(w, http.StatusNoContent, nil)
}
