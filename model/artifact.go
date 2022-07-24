package model

type Artifact struct {
	ID          string
	App         *App     `json:"-"`
	Variant     *Variant `json:"-"`
	AppName     string   `json:"App,omitempty"`
	VariantName string   `json:"Variant,omitempty"`
	Name        string
	Icon        string
	Scheme      string `json:"-"`
	File        string
	Size        int64
	URL         string
	Sources     []*Source
}
