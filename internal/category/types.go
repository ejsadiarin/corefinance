package category

type CreateRequest struct {
	Name  string  `json:"name"`
	Color *string `json:"color,omitempty"`
	Icon  *string `json:"icon,omitempty"`
}

type UpdateRequest struct {
	Name  string  `json:"name"`
	Color *string `json:"color,omitempty"`
	Icon  *string `json:"icon,omitempty"`
}
