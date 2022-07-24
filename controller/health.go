package controller

import (
	"net/http"
	"service/dao"
	"time"

	log "github.com/sirupsen/logrus"
)

type HealthController struct{}

func (c *HealthController) Check(w http.ResponseWriter, r *http.Request) {
	dao := &dao.HealthDao{Context: r.Context()}
	sites, err := dao.Sites()
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	client := &http.Client{
		Timeout: 5 * time.Second,
	}
	for _, site := range sites {
		for _, s := range site.Sources {
			log.Debugf("Checking source: %s", s.URL.String())
			_ = s.Check(client)
			log.Debugf("%s => %s @ %v", s.URL.String(), s.Status.String(), s.Latency)
		}
	}
	err = dao.Update(sites)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
}
