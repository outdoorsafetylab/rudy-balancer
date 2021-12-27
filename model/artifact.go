package model

import (
	"fmt"
	"net/url"
)

type Artifact struct {
	App           *App     `json:"-"`
	Variant       *Variant `json:"-"`
	ID            string
	Name          string
	Scheme        string
	File          string
	ContentLength int64
	Healths       []*Health
	HealthyURLs   []*url.URL `json:"-"`
}

// func (a *Artifact) GetURLs(sites []*Site) ([]*url.URL, error) {
// 	urls := make([]*url.URL, 0)
// 	for _, s := range sites {
// 		for _, scheme := range s.GetSchemes() {
// 			u, err := url.Parse(fmt.Sprintf("%s://%s%s", scheme, s.Endpoint, a.File))
// 			if err != nil {
// 				return nil, err
// 			}
// 			urls = append(urls, u)
// 		}
// 	}
// 	return urls, nil
// }

func (a *Artifact) GetID() string {
	if a.Variant == nil {
		return fmt.Sprintf("%s/%s", a.App.ID, a.ID)
	}
	return fmt.Sprintf("%s/%s/%s", a.App.ID, a.Variant.ID, a.ID)
}
