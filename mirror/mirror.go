package mirror

import (
	"path/filepath"
	"service/config"
	"service/model"

	"github.com/spf13/viper"
)

type Mirror struct {
	Sites []*model.Site
	Apps  []*model.App
	Files []string
}

func load() (*Mirror, error) {
	filename := config.Get().GetString("mirrors.file")
	cfg := viper.New()
	cfg.SetConfigType("yaml")
	cfg.SetConfigName(filepath.Base(filename))
	cfg.AddConfigPath(filepath.Dir(filename))
	err := cfg.ReadInConfig()
	if err != nil {
		return nil, err
	}
	meta := &Mirror{}
	err = cfg.Unmarshal(meta)
	if err != nil {
		return nil, err
	}
	files := make(map[string]string)
	sources := make(map[string]*model.Source)
	for _, app := range meta.Apps {
		for _, v := range app.Variants {
			if v.Icon == "" {
				v.Icon = app.Icon
			}
			for _, a := range v.Artifacts {
				if a.Icon == "" {
					a.Icon = v.Icon
				}
				a.App = app
				a.Variant = v
				a.Sources = make([]*model.Source, 0)
				for _, s := range meta.Sites {
					u := s.GetURL(a.File)
					src := sources[u]
					if src == nil {
						src = &model.Source{
							Site: s,
							URL:  u,
							File: a.File,
						}
						s.Sources = append(s.Sources, src)
						sources[u] = src
					}
					a.Sources = append(a.Sources, src)
				}
				if files[a.File] == "" {
					files[a.File] = a.File
				}
			}
		}
	}
	meta.Files = make([]string, 0)
	for _, file := range files {
		meta.Files = append(meta.Files, file)
	}
	return meta, nil
}

func Get() (*Mirror, error) {
	return load()
}

func Apps() ([]*model.App, error) {
	meta, err := load()
	if err != nil {
		return nil, err
	}
	return meta.Apps, nil
}

func Sites() ([]*model.Site, error) {
	meta, err := load()
	if err != nil {
		return nil, err
	}
	return meta.Sites, nil
}

func Artifacts() ([]*model.Artifact, error) {
	meta, err := load()
	if err != nil {
		return nil, err
	}
	artifacts := make([]*model.Artifact, 0)
	for _, a := range meta.Apps {
		for _, v := range a.Variants {
			artifacts = append(artifacts, v.Artifacts...)
		}
	}
	return artifacts, nil
}
