// Package helper have function helpers for handlers
package helper

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/ejsadiarin/corefinance/internal/auth"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

func RespondJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if data != nil {
		err := json.NewEncoder(w).Encode(data)
		if err != nil {
			// if encoding fails then connection must be broken or data is invalid
			slog.Info("failed to encode response", "error", err.Error())
		}
	}
}

func RespondErrorJSON(w http.ResponseWriter, status int, message string) {
	RespondJSON(w, status, map[string]string{"error": message})
}

func ParseUUID(w http.ResponseWriter, r *http.Request, param string) (uuid.UUID, bool) {
	id, err := uuid.Parse(chi.URLParam(r, param))
	if err != nil {
		RespondErrorJSON(w, http.StatusBadRequest, "invalid "+param)
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
		RespondErrorJSON(w, http.StatusUnauthorized, "X-User-ID header is required")
		return uuid.Nil, false
	}
	return userID, true
}

// ValidateRequired returns an error if the value is empty
func ValidateRequired(value, fieldName string) error {
	if value == "" {
		return fmt.Errorf("%s is required", fieldName)
	}
	return nil
}

// ValidateAmount returns an error if the amount is negative
func ValidateAmount(amount float64, fieldName string) error {
	if amount < 0 {
		return fmt.Errorf("%s must not be negative", fieldName)
	}
	return nil
}

// ValidatePriority returns an error if the priority is not one of the allowed values
func ValidatePriority(priority string) error {
	switch priority {
	case "need", "want", "savings":
		return nil
	default:
		return fmt.Errorf("priority must be one of: need, want, savings")
	}
}

// ValidateRecurringType returns an error if the recurring_type is not one of the allowed values
func ValidateRecurringType(recurringType string) error {
	switch recurringType {
	case "one-time", "daily", "weekly", "monthly", "yearly":
		return nil
	default:
		return fmt.Errorf("recurring_type must be one of: one-time, daily, weekly, monthly, yearly")
	}
}

// ValidateStatus returns an error if the status is not one of the allowed values
func ValidateStatus(status string) error {
	switch status {
	case "pending", "posted", "skipped":
		return nil
	default:
		return fmt.Errorf("status must be one of: pending, posted, skipped")
	}
}

type PaginatedResponse struct {
	Data     interface{} `json:"data"`
	Total    int64       `json:"total"`
	Page     int         `json:"page"`
	PageSize int         `json:"page_size"`
	HasMore  bool        `json:"has_more"`
}
