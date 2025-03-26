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
	cfg := config.Get()
	auth := cfg.GetString("healthcheck.auth")
	if auth == "" {
		http.Error(w, "Missing health check authorization", http.StatusUnauthorized)
		return
	}
	if auth != r.Header.Get("Authorization") {
		http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}
	dao := &dao.SiteDao{Context: r.Context()}
	sites, err := mirror.Sites()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	client := &http.Client{
		Timeout: time.Duration(cfg.GetInt("healthcheck.timeout_sec")) * time.Second,
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
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}
