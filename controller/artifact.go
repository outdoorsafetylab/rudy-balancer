package controller

import (
	"math/rand"
	"net/http"
	"service/dao"
	"service/model"

	log "github.com/sirupsen/logrus"
)

type ArtifactController struct {
	Artifact *model.Artifact
	URI      string
}

func (c *ArtifactController) Download(w http.ResponseWriter, r *http.Request) {
	dao := &dao.HealthDao{Context: r.Context()}
	urls, err := dao.GetURLs(c.Artifact)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	if urls == nil || len(urls) <= 0 {
		log.Warningf("No available sources: %s", c.Artifact.File)
		http.Error(w, err.Error(), 501)
		return
	}
	u := urls[rand.Intn(len(urls))]
	log.Warningf("Redircting: %s => %s", c.Artifact.File, u.String())
	http.Redirect(w, r, u.String(), 302)
}
