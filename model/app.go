package model

import (
	"image"
)

type App struct {
	ID        string
	Name      string
	Icon      string
	IconImage image.Image `json:"-"`
	Artifacts []*Artifact `json:",omitempty"`
	Variants  []*Variant  `json:",omitempty"`
}

func (a *App) GetArtifacts() []*Artifact {
	artifacts := make([]*Artifact, 0)
	for _, t := range a.Artifacts {
		t.App = a
		artifacts = append(artifacts, t)
	}
	for _, v := range a.Variants {
		for _, t := range v.Artifacts {
			t.App = a
			t.Variant = v
			artifacts = append(artifacts, t)
		}
	}
	return artifacts
}
