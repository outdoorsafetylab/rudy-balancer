package mirror

import (
	"fmt"
	"path/filepath"
	"service/config"
	"service/model"

	"github.com/spf13/viper"
)

type Mirror struct {
	Files map[string][]string
	Sites []*model.Site
	Apps  []*model.App
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
	for _, list := range meta.Files {
		for _, filename := range list {
			if files[filename] != "" {
				return nil, fmt.Errorf("duplicated file: %s", filename)
			}
			files[filename] = filename
		}
	}
	sources := make(map[string]*model.Source)
	for _, site := range meta.Sites {
		if site.ID == "" {
			site.ID = site.Name
		}
		for filename := range files {
			source := &model.Source{
				Site: site,
				URL:  site.GetURL(filename),
				File: filename,
			}
			site.Sources = append(site.Sources, source)
			sources[source.URL] = source
		}
	}
	for _, app := range meta.Apps {
		for _, v := range app.Variants {
			if v.Icon == "" {
				v.Icon = app.Icon
			}
			for _, a := range v.Artifacts {
				if files[a.File] == "" {
					return nil, fmt.Errorf("undefined file: %s", a.File)
				}
				if a.Icon == "" {
					a.Icon = v.Icon
				}
				a.App = app
				a.Variant = v
				for _, site := range meta.Sites {
					a.Sources = append(a.Sources, sources[site.GetURL(a.File)])
				}
			}
		}
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
