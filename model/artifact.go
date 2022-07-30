package model

type Artifact struct {
	ID          string
	App         *App     `json:"-"`
	Variant     *Variant `json:"-"`
	AppName     string   `json:"App,omitempty"`
	VariantName string   `json:"Variant,omitempty"`
	Icon        string
	Name        string
	Description string `json:",omitempty"`
	Scheme      string `json:"-"`
	File        string `json:",omitempty"`
	Size        int64  `json:",omitempty"`
	URL         string
	Sources     []*Source `json:",omitempty"`
}
