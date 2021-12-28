package model

type App struct {
	ID       string
	Name     string
	Icon     string
	Variants []*Variant `json:",omitempty"`
}

func (a *App) GetArtifacts() []*Artifact {
	artifacts := make([]*Artifact, 0)
	for _, v := range a.Variants {
		for _, t := range v.Artifacts {
			t.App = a
			t.Variant = v
			if t.Icon == "" {
				t.Icon = v.Icon
			}
			if t.Icon == "" {
				t.Icon = a.Icon
			}
			artifacts = append(artifacts, t)
		}
	}
	return artifacts
}
