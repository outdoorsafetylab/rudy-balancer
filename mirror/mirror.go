package mirror

import (
	"fmt"
	"net/url"
	"path/filepath"
	"service/config"
	"service/model"

	"github.com/spf13/viper"
)

type metadata struct {
	Sites []*model.Site
	Apps  []*model.App
}

func load() (*metadata, error) {
	filename := config.Get().GetString("mirrors.file")
	cfg := viper.New()
	cfg.SetConfigType("yaml")
	cfg.SetConfigName(filepath.Base(filename))
	cfg.AddConfigPath(filepath.Dir(filename))
	err := cfg.ReadInConfig()
	if err != nil {
		return nil, err
	}
	meta := &metadata{}
	err = cfg.Unmarshal(meta)
	if err != nil {
		return nil, err
	}
	files := make(map[string]*model.Artifact)
	for _, app := range meta.Apps {
		for _, v := range app.Variants {
			if v.Icon == "" {
				v.Icon = app.Icon
			}
			for _, a := range v.Artifacts {
				if a.Icon == "" {
					a.Icon = v.Icon
				}
				b := files[a.File]
				if b != nil {
					return nil, fmt.Errorf("duplicated artifact: %s", b.File)
				}
				a.App = app
				a.Variant = v
				a.Sources = make([]*model.Source, 0)
				for _, s := range meta.Sites {
					u, err := url.Parse(fmt.Sprintf("%s://%s%s", s.Scheme, s.Endpoint, a.File))
					if err != nil {
						return nil, err
					}
					a.Sources = append(a.Sources, &model.Source{
						Site: s,
						URL:  u,
					})
				}
			}
		}
	}
	return meta, nil
}

func Apps() ([]*model.App, error) {
	meta, err := load()
	if err != nil {
		return nil, err
	}
	return meta.Apps, nil
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
