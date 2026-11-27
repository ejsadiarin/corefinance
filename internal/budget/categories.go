package budget

import "github.com/google/uuid"

// categoryTarget is where an imported record's category lands (ADR-0002):
// the system category of the same name, or `other` plus a tag of the old name.
type categoryTarget struct {
	ID  uuid.UUID
	Tag string
}
