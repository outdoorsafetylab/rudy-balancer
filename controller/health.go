package controller

import (
	"net/http"
	"service/dao"
	"service/model"
	"time"

	log "github.com/sirupsen/logrus"
)

type HealthController struct{}

func (c *HealthController) Check(w http.ResponseWriter, r *http.Request) {
	dao := &dao.HealthDao{Context: r.Context()}
	apps, err := dao.Apps()
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	artifacts := make([]*model.Artifact, 0)
	for _, a := range apps {
		for _, v := range a.Variants {
			artifacts = append(artifacts, v.Artifacts...)
		}
	}
	client := &http.Client{
		Timeout: 5 * time.Second,
	}
	for _, a := range artifacts {
		log.Debugf("Checking artifact: %s", a.File)
		for _, s := range a.Sources {
			_ = s.Check(client)
			log.Debugf("%s => %s @ %v", s.URL.String(), s.Status.String(), s.Latency)
		}
	}
	err = dao.Update(artifacts)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
}
