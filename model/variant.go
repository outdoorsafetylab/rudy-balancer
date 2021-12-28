package model

type Variant struct {
	ID        string
	Name      string
	Icon      string
	Artifacts []*Artifact
}
