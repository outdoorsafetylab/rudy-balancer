package controller

import (
	"net/http"
	"service/dao"
	"service/model"
)

type SiteController struct{}

type site struct {
	Name    string
	Hidden  bool
	Scheme  string
	Host    string
	Prefix  string
	Sources []*model.Source
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
	res := make([]*site, len(sites))
	for i, s := range sites {
		res[i] = &site{
			Name:    s.Name,
			Hidden:  s.Hidden,
			Scheme:  s.Scheme,
			Host:    s.Host,
			Prefix:  s.Prefix,
			Sources: s.Sources,
		}
	}
	writeJSON(w, r, &res)
}
