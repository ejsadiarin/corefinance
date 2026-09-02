package tag

type CreateRequest struct {
	Name  string  `json:"name"`
	Color *string `json:"color,omitempty"`
}

type UpdateRequest struct {
	Name  string  `json:"name"`
	Color *string `json:"color,omitempty"`
}
