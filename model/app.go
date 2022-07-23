package model

type App struct {
	ID       string
	Name     string
	Icon     string
	Variants []*Variant `json:",omitempty"`
}
