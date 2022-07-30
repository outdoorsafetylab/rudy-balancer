package model

type App struct {
	ID          string
	Name        string
	Icon        string
	Description string
	Variants    []*Variant `json:",omitempty"`
}
