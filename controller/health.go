package controller

import (
	"net/http"

	"service/config"
	"service/healthcheck"
	"service/log"
)

type HealthController struct{}

func (c *HealthController) Check(w http.ResponseWriter, r *http.Request) {
	cfg := config.Get()
	auth := cfg.GetString("healthcheck.auth")
	if auth != "" && auth != r.Header.Get("Authorization") {
		http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}

	// Part 1: Portal Sites Health Check
	log.Debugf("Starting portal sites health check")
	err := healthcheck.CheckPortalSites(cfg)
	if err != nil {
		log.Errorf("Portal sites health check failed: %s", err.Error())
		// Don't return error here, continue with mirror sites check
	}

	// Part 2: Mirror Sites Health Check
	log.Debugf("Starting mirror sites health check")
	err = healthcheck.CheckMirrorSites(cfg, r.Context())
	if err != nil {
		log.Errorf("Mirror sites health check failed: %s", err.Error())
	}

	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Health check completed"))
}
