package tag

import "github.com/google/uuid"

type CreateRequest struct {
	Name  string  `json:"name"`
	Color *string `json:"color,omitempty"`
}

type UpdateRequest struct {
	Name  string  `json:"name"`
	Color *string `json:"color,omitempty"`
}

type TagAssociationRequest struct {
	TagID string `json:"tag_id"`
}

// Link says a record carries a tag.
type Link struct {
	RecordID uuid.UUID `json:"record_id"`
	TagID    uuid.UUID `json:"tag_id"`
}

// Links are the tag links of all of a user's expenses and incomes.
type Links struct {
	Expenses []Link `json:"expenses"`
	Incomes  []Link `json:"incomes"`
}
