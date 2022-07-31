package controller

import (
	"net/http"
	"service/dao"
)

type StatsController struct {
}

func (c *StatsController) Total(w http.ResponseWriter, r *http.Request) {
	dao := &dao.FileDao{SiteDao: dao.SiteDao{Context: r.Context()}}
	stats, err := dao.TotalStats()
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	writeJSON(w, r, &stats)
}
