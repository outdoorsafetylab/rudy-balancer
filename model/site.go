package model

import "fmt"

type Site struct {
	Name     string
	Hidden   bool      `json:",omitempty"`
	Endpoint string    `json:"-"`
	Scheme   string    `json:"-"`
	Sources  []*Source `json:"-"`
}

func (s *Site) GetURL(file string) string {
	return fmt.Sprintf("%s://%s/%s", s.Scheme, s.Endpoint, file)
}
