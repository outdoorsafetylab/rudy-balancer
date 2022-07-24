package statuspage

type Incident struct {
	ID         string `json:"id"`
	PageID     string `json:"page_id"`
	Components []struct {
		ID string `json:"id"`
	} `json:"components"`
}
