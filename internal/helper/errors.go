package helper

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/ejsadiarin/corefinance/internal/rates"
)

// ValidateDate checks a YYYY-MM-DD date.
func ValidateDate(value, fieldName string) error {
	if value == "" {
		return fmt.Errorf("%s is required", fieldName)
	}
	if _, err := time.Parse("2006-01-02", value); err != nil {
		return fmt.Errorf("%s must be YYYY-MM-DD", fieldName)
	}
	return nil
}

// RespondServiceError maps the exchange-rate refusals to the client statuses
// ADR-0003 promises, and everything else to 500.
func RespondServiceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, rates.ErrUnsupportedCurrency):
		RespondErrorJSON(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, rates.ErrRateUnavailable):
		RespondErrorJSON(w, http.StatusBadGateway, err.Error())
	default:
		RespondErrorJSON(w, http.StatusInternalServerError, err.Error())
	}
}
