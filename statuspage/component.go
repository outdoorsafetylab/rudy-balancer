package statuspage

type Component struct {
	ID     string `json:"id"`
	PageID string `json:"page_id"`
	Name   string `json:"name"`
	Status string `json:"status"`
}
