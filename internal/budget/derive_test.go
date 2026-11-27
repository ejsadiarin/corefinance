package budget

import (
	"testing"

	"github.com/google/uuid"
)

// Every service derives a copied row's id the same way, so links between
// parts still meet (coregateway ADR-0008). This vector is the same in
// corefinance, coreban and coredraw.
func TestDeriveIDMatchesEveryService(t *testing.T) {
	owner := uuid.MustParse("6ba7b810-9dad-11d1-80b4-00c04fd430c8")
	id := uuid.MustParse("0b9b5c9e-3c1a-4f7e-9a51-2a3c4d5e6f70")
	if got := DeriveID(owner, id).String(); got != "6969be21-b797-5c02-a3c8-79e5b7ff8f8d" {
		t.Errorf("DeriveID = %s", got)
	}
}
