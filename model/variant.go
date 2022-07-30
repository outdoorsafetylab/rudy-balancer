package model

type Variant struct {
	ID          string
	Name        string
	Description string `json:",omitempty"`
	Artifacts   []*Artifact
}
