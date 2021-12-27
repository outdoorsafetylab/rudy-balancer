package model

import (
	"fmt"
	"net/url"
)

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

func (s *Site) GetURLs(a *Artifact) ([]*url.URL, error) {
	urls := make([]*url.URL, 0)
	for _, scheme := range s.GetSchemes() {
		u, err := url.Parse(fmt.Sprintf("%s://%s%s", scheme, s.Endpoint, a.File))
		if err != nil {
			return nil, err
		}
		urls = append(urls, u)
	}
	return urls, nil
}
