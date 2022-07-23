package model

type Artifact struct {
	App     *App     `json:"-"`
	Variant *Variant `json:"-"`
	ID      string
	Name    string
	Icon    string
	Scheme  string `json:"-"`
	File    string
	Size    int64
	URL     string
	Sources []*Source
}
