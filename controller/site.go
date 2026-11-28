package controller

import (
	"net/http"
	"service/dao"
	"service/model"
)

type SiteController struct{}

type site struct {
	Name           string
	Hidden         bool
	Host           string
	Prefix         string
	RedirectScheme string
	Sources        []*model.Source
}

func (c *SiteController) List(w http.ResponseWriter, r *http.Request) {
	dao := &dao.SiteDao{Context: r.Context()}
	sites, err := dao.Sites()
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	for _, site := range sites {
		for _, src := range site.Sources {
			src.Site = nil
		}
	}
	res := listSites(sites)
	writeJSON(w, r, &res)
}

// listSites is what /v1/sites answers. An upstream is left out: this list is
// what an app falls back to when both entry points are down, and it must only
// name mirrors.
func listSites(sites []*model.Site) []*site {
	res := make([]*site, 0, len(sites))
	for _, s := range sites {
		if s.Upstream {
			continue
		}
		res = append(res, &site{
			Name:           s.Name,
			Hidden:         s.Hidden,
			Host:           s.Host,
			Prefix:         s.Prefix,
			RedirectScheme: s.RedirectScheme,
			Sources:        s.Sources,
		})
	}
	return res
}
