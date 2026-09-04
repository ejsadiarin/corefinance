// Package helper have function helpers for handlers
package helper

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/ejsadiarin/corefinance/internal/auth"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

func Respond(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if data != nil {
		err := json.NewEncoder(w).Encode(data)
		if err != nil {
			// if encoding fails then connection must be broken or data is invalid
			slog.Info("failed to encode response %v", err.Error())
		}
	}
}

func RespondError(w http.ResponseWriter, status int, message string) {
	Respond(w, status, map[string]string{"error": message})
}

func ParseUUID(w http.ResponseWriter, r *http.Request, param string) (uuid.UUID, bool) {
	id, err := uuid.Parse(chi.URLParam(r, param))
	if err != nil {
		RespondError(w, http.StatusBadRequest, "invalid "+param)
		return uuid.Nil, false
	}
	return id, true
}

func ParseQueryInt(r *http.Request, key string, defaultVal int) int {
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

func ParseQueryString(r *http.Request, key string) *string {
	val := r.URL.Query().Get(key)
	if val == "" {
		return nil
	}
	return &val
}

func GetUserID(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	userID, ok := auth.GetUserID(r)
	if !ok {
		RespondError(w, http.StatusUnauthorized, "X-User-ID header is required")
		return uuid.Nil, false
	}
	return userID, true
}
