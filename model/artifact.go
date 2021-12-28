package model

import (
	"fmt"
	"image"
	"net/url"
)

type Artifact struct {
	App           *App     `json:"-"`
	Variant       *Variant `json:"-"`
	ID            string
	Name          string
	Icon          string
	IconImage     image.Image `json:"-"`
	Scheme        string
	File          string
	Size          string
	ContentLength int64
	Healths       []*Health
	HealthyURLs   []*url.URL `json:"-"`
}

func (a *Artifact) GetPath() string {
	return fmt.Sprintf("%s/%s/%s", a.App.ID, a.Variant.ID, a.ID)
}
