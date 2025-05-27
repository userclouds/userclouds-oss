package pagination

// ResponseFields represents pagination-specific fields present in every response.
type ResponseFields struct {
	HasNext bool   `json:"has_next" yaml:"has_next"`
	Next    Cursor `json:"next" yaml:"next"`
	HasPrev bool   `json:"has_prev" yaml:"has_prev"`
	Prev    Cursor `json:"prev" yaml:"prev"`
}
