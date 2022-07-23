package model

import (
	"fmt"
	"net/url"
)

type Site struct {
	Name     string
	Endpoint string `json:"-"`
	Scheme   string `json:"-"`
}

func (s *Site) GetURLs(a *Artifact) ([]*url.URL, error) {
	urls := make([]*url.URL, 0)
	u, err := url.Parse(fmt.Sprintf("%s://%s%s", s.Scheme, s.Endpoint, a.File))
	if err != nil {
		return nil, err
	}
	urls = append(urls, u)
	return urls, nil
}
