package stats

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCashFlowHandler_RejectsBadInput(t *testing.T) {
	h := NewHandler(nil)
	for _, tc := range []struct {
		name string
		url  string
	}{
		{"missing bucket", "/cash-flow?count=7"},
		{"unknown bucket", "/cash-flow?bucket=year&count=7"},
		{"missing count", "/cash-flow?bucket=day"},
		{"zero count", "/cash-flow?bucket=day&count=0"},
		{"count over the bound", "/cash-flow?bucket=day&count=63"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			h.CashFlow(rec, httptest.NewRequest(http.MethodGet, tc.url, nil))
			assert.Equal(t, http.StatusBadRequest, rec.Code)
		})
	}
}

func TestPeriodBudgetHandler_RejectsBadInput(t *testing.T) {
	h := NewHandler(nil)
	for _, tc := range []struct {
		name string
		url  string
	}{
		{"missing period", "/period-budget"},
		{"unknown period", "/period-budget?period=year"},
		{"bad date", "/period-budget?period=day&date=2026-13-01"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			h.PeriodBudget(rec, httptest.NewRequest(http.MethodGet, tc.url, nil))
			assert.Equal(t, http.StatusBadRequest, rec.Code)
		})
	}
}
