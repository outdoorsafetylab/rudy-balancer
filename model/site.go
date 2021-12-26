package model

type Site struct {
	Name     string
	Endpoint string
	Schemes  []string
}

func (s *Site) GetSchemes() []string {
	if s.Schemes == nil {
		return []string{"https"}
		// return []string{"https", "http"}
	}
	return s.Schemes
}
