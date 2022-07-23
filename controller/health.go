package controller

import (
	"net/http"
	"service/dao"
	"service/mirror"
	"time"

	log "github.com/sirupsen/logrus"
)

type HealthController struct{}

func (c *HealthController) Check(w http.ResponseWriter, r *http.Request) {
	artifacts, err := mirror.Artifacts()
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	client := &http.Client{
		Timeout: 5 * time.Second,
	}
	for _, a := range artifacts {
		log.Debugf("Checking artifact: %s", a.File)
		for _, s := range a.Sources {
			_ = s.Check(client)
			log.Debugf("%s => %s", s.URL.String(), s.Status.String())
		}
	}
	dao := &dao.HealthDao{Context: r.Context()}
	err = dao.Update(artifacts)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
}
