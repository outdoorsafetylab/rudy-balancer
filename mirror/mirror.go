package mirror

import (
	"fmt"
	"path/filepath"
	"service/config"
	"service/model"
	"sort"
	"strings"

	"github.com/spf13/viper"
)

type Mirror struct {
	Files map[string][]string
	// Folders maps a folder published as a whole (the unpacked PMTiles style)
	// to the file in it that stands for it. The health check probes that file
	// like any listed one, and /v1/<folder>/<path> picks a mirror by it (#63).
	Folders map[string]string
	Sites   []*model.Site
	Apps    []*model.App
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
	folders := make([]string, 0, len(meta.Folders))
	for folder := range meta.Folders {
		folders = append(folders, folder)
	}
	sort.Strings(folders)
	for _, folder := range folders {
		probe := meta.Folders[folder]
		if folder == "" || probe == "" || strings.Contains(folder, "/") || strings.HasPrefix(probe, "/") {
			return nil, fmt.Errorf("bad folder: %q: %q", folder, probe)
		}
		// A folder route is matched before the file routes, so a listed file
		// inside a folder of that name would never be reached.
		for _, list := range meta.Files {
			for _, f := range list {
				if strings.HasPrefix(f, folder+"/") {
					return nil, fmt.Errorf("folder %q would shadow listed file %s", folder, f)
				}
			}
		}
		if meta.Files == nil {
			meta.Files = make(map[string][]string)
		}
		meta.Files["folders"] = append(meta.Files["folders"], FolderProbe(folder, probe))
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
		if site.StatusPage == "" {
			site.StatusPage = site.Name
		}
		if site.Firestore == "" {
			site.Firestore = site.Name
		}
		// An upstream is never redirected to, whatever weight it is given, but
		// stays visible so its status is published.
		if site.Upstream {
			site.Weight = 0
			site.Hidden = false
		} else if site.Weight <= 0 {
			// A site with no weight gets no downloads, so it is hidden as well:
			// the portal leaves it out and its status is no longer published.
			site.Hidden = true
		}
		for filename := range files {
			source := &model.Source{
				Site:     site,
				SiteName: site.Name,
				URL:      site.GetRedirectURL(filename),
				File:     filename,
			}
			site.Sources = append(site.Sources, source)
			sources[source.URL] = source
		}
	}
	for _, app := range meta.Apps {
		for _, v := range app.Variants {
			for _, a := range v.Artifacts {
				a.App = app
				a.Variant = v
				if a.File == "" {
					continue
				}
				if files[a.File] == "" {
					return nil, fmt.Errorf("undefined file: %s > %s > %s: %s", app.Name, v.Name, a.Name, a.File)
				}
				for _, site := range meta.Sites {
					a.Sources = append(a.Sources, sources[site.GetRedirectURL(a.File)])
				}
			}
		}
	}
	return meta, nil
}

// FolderProbe is the listed file that stands for a folder.
func FolderProbe(folder, probe string) string {
	return folder + "/" + probe
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
