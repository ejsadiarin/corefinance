package helper

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ejsadiarin/corefinance/internal/rates"
)

func TestRespondServiceError(t *testing.T) {
	for _, tc := range []struct {
		name string
		err  error
		want int
	}{
		{"unsupported currency", fmt.Errorf("create: %w", rates.ErrUnsupportedCurrency), http.StatusBadRequest},
		{"provider down", fmt.Errorf("create: %w", rates.ErrRateUnavailable), http.StatusBadGateway},
		{"anything else", errors.New("database down"), http.StatusInternalServerError},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			RespondServiceError(rec, tc.err)
			assert.Equal(t, tc.want, rec.Code)
		})
	}
}
