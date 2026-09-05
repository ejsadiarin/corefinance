package helper_test

import (
	"net/http"
	"testing"

	"github.com/ejsadiarin/corefinance/internal/helper"
)

func TestRespond(t *testing.T) {
	tests := []struct {
		name string // description of this test case
		// Named input parameters for target function.
		w      http.ResponseWriter
		status int
		data   interface{}
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			helper.Respond(tt.w, tt.status, tt.data)
		})
	}
}
