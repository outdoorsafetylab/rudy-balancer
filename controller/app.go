package controller

import (
	"fmt"
	"net/http"
	"service/config"
	"service/dao"
)

type AppController struct{}

func (c *AppController) List(w http.ResponseWriter, r *http.Request) {
	dao := &dao.HealthDao{Context: r.Context()}
	apps, err := dao.Apps()
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	cfg := config.Get()
	prefix := cfg.GetString("endpoint")
	for _, app := range apps {
		for _, v := range app.Variants {
			for _, a := range v.Artifacts {
				if a.Scheme == "" {
					a.Scheme = cfg.GetString("mirrors.default_scheme")
				}
				a.URL = fmt.Sprintf("%s//%s%s/%s", a.Scheme, r.Host, prefix, a.File)
				for _, s := range a.Sources {
					s.URLString = ""
				}
			}
		}
	}
	writeJSON(w, r, apps)
}
