package income

import (
	"encoding/json"
	"net/http"
	"strconv"

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

func (h *Handler) parseQueryInt(r *http.Request, key string, defaultVal int) int {
	val := r.URL.Query().Get(key)
	if val == "" {
		return defaultVal
	}
	parsed, err := strconv.Atoi(val)
	if err != nil {
		return defaultVal
	}
	return parsed
}

func (h *Handler) parseQueryString(r *http.Request, key string) *string {
	val := r.URL.Query().Get(key)
	if val == "" {
		return nil
	}
	return &val
}

func (h *Handler) getUserID(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	userID, ok := auth.GetUserID(r)
	if !ok {
		h.respondError(w, http.StatusUnauthorized, "X-User-ID header is required")
		return uuid.Nil, false
	}
	return userID, true
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var req CreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	userID, ok := h.getUserID(w, r)
	if !ok {
		return
	}
	income, err := h.service.Create(r.Context(), userID, req)
	if err != nil {
		h.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	h.respond(w, http.StatusCreated, income)
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	id, ok := h.parseUUID(w, r, "id")
	if !ok {
		return
	}
	userID, ok := h.getUserID(w, r)
	if !ok {
		return
	}
	income, err := h.service.Get(r.Context(), id, userID)
	if err != nil {
		h.respondError(w, http.StatusNotFound, "income not found")
		return
	}
	h.respond(w, http.StatusOK, income)
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.getUserID(w, r)
	if !ok {
		return
	}
	params := ListParams{
		StartDate:     h.parseQueryString(r, "start_date"),
		EndDate:       h.parseQueryString(r, "end_date"),
		CategoryID:    h.parseQueryString(r, "category_id"),
		Status:        h.parseQueryString(r, "status"),
		RecurringType: h.parseQueryString(r, "recurring_type"),
		Page:          h.parseQueryInt(r, "page", 1),
		PageSize:      h.parseQueryInt(r, "page_size", 50),
	}
	incomes, err := h.service.List(r.Context(), userID, params)
	if err != nil {
		h.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	h.respond(w, http.StatusOK, incomes)
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
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
	income, err := h.service.Update(r.Context(), id, userID, req)
	if err != nil {
		h.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	h.respond(w, http.StatusOK, income)
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	id, ok := h.parseUUID(w, r, "id")
	if !ok {
		return
	}
	userID, ok := h.getUserID(w, r)
	if !ok {
		return
	}
	if err := h.service.Delete(r.Context(), id, userID); err != nil {
		h.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	h.respond(w, http.StatusNoContent, nil)
}

func (h *Handler) Skip(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	id, err := uuid.Parse(req.ID)
	if err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid income id")
		return
	}
	userID, ok := h.getUserID(w, r)
	if !ok {
		return
	}
	if err := h.service.Skip(r.Context(), id, userID); err != nil {
		h.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	h.respond(w, http.StatusOK, map[string]string{"status": "skipped"})
}

func (h *Handler) CheckSkipped(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.getUserID(w, r)
	if !ok {
		return
	}
	startDate := r.URL.Query().Get("start_date")
	endDate := r.URL.Query().Get("end_date")
	if startDate == "" || endDate == "" {
		h.respondError(w, http.StatusBadRequest, "start_date and end_date are required")
		return
	}
	skipped, err := h.service.CheckSkipped(r.Context(), userID, startDate, endDate)
	if err != nil {
		h.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	h.respond(w, http.StatusOK, map[string]bool{"is_skipped": skipped})
}

func (h *Handler) GetOccurrences(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.getUserID(w, r)
	if !ok {
		return
	}
	startDate := h.parseQueryString(r, "start_date")
	endDate := h.parseQueryString(r, "end_date")
	occurrences, err := h.service.Occurrences(r.Context(), userID, startDate, endDate)
	if err != nil {
		h.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	h.respond(w, http.StatusOK, occurrences)
}
