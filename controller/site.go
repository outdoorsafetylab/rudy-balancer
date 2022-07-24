package controller

import (
	"net/http"
	"service/dao"
	"service/model"
)

type SiteController struct{}

type site struct {
	Name     string
	Hidden   bool
	Endpoint string
	Scheme   string
	Sources  []*model.Source
}

func (c *SiteController) List(w http.ResponseWriter, r *http.Request) {
	dao := &dao.HealthDao{Context: r.Context()}
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
			Name:     s.Name,
			Hidden:   s.Hidden,
			Endpoint: s.Endpoint,
			Scheme:   s.Scheme,
			Sources:  s.Sources,
		}
	}
	writeJSON(w, r, &res)
}
