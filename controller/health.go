package controller

import (
	"net/http"
	"time"

	"service/config"
	"service/dao"
	"service/log"
	"service/mirror"
)

type HealthController struct{}

func (c *HealthController) Check(w http.ResponseWriter, r *http.Request) {
	auth := config.Get().GetString("healthcheck.auth")
	if auth == "" {
		http.Error(w, "Missing health check authorization", 401)
		return
	}
	if auth != r.Header.Get("Authorization") {
		http.Error(w, http.StatusText(401), 401)
		return
	}
	dao := &dao.SiteDao{Context: r.Context()}
	sites, err := mirror.Sites()
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	client := &http.Client{
		Timeout: 1 * time.Second,
	}
	for _, site := range sites {
		for _, s := range site.Sources {
			log.Debugf("Checking source: %s", s.URL)
			_ = s.Check(client)
			log.Debugf("%s => %s @ %v", s.URL, s.Status.String(), s.Latency)
		}
	}
	err = dao.Update(sites)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
}
