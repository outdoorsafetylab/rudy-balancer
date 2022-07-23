package controller

import (
	"fmt"
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

func (c *ArtifactController) URL(w http.ResponseWriter, r *http.Request) {
	url := fmt.Sprintf("%s//%s%s", c.Artifact.Scheme, r.Host, c.URI)
	http.Error(w, url, 200)
}

func (c *ArtifactController) Download(w http.ResponseWriter, r *http.Request) {
	dao := &dao.HealthDao{Context: r.Context()}
	urls, err := dao.GetAvailableURLs(c.Artifact)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	if urls == nil || len(urls) <= 0 {
		log.Warningf("No available sources: %s (%s)", c.Artifact.File, c.Artifact.File)
		http.Error(w, err.Error(), 501)
		return
	}
	u := urls[rand.Intn(len(urls))]
	log.Warningf("Redircting: %s => %s", c.Artifact.File, u.String())
	http.Redirect(w, r, u.String(), 302)
}
