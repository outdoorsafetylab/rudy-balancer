package controller

import (
	"net/http"
	"service/dao"
	"time"

	"github.com/gorilla/mux"
)

type StatsController struct {
}

func (c *StatsController) Stats(w http.ResponseWriter, r *http.Request) {
	dao := &dao.FileDao{SiteDao: dao.SiteDao{Context: r.Context()}}
	stats, err := dao.Stats(mux.Vars(r)["site"])
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	writeJSON(w, r, &stats)
}

func (c *StatsController) DailyStats(w http.ResponseWriter, r *http.Request) {
	until := time.Now()
	since := until.Add(-time.Hour * 24 * 30)
	dao := &dao.FileDao{SiteDao: dao.SiteDao{Context: r.Context()}}
	stats, err := dao.DailyStats(mux.Vars(r)["site"], since, until)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	writeJSON(w, r, &stats)
}
