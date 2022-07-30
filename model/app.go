package model

type App struct {
	ID          string
	Name        string
	Icon        string
	Description string
	Android     bool
	IOS         bool
	Variants    []*Variant `json:",omitempty"`
}
