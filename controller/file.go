package controller

import (
	"fmt"
	"math/rand"
	"net/http"
	"service/dao"

	log "github.com/sirupsen/logrus"
)

type FileController struct {
	File string
}

func (c *FileController) Download(w http.ResponseWriter, r *http.Request) {
	dao := &dao.HealthDao{Context: r.Context()}
	urls, err := dao.GetURLs(c.File)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	if urls == nil || len(urls) <= 0 {
		log.Warningf("No available sources: %s", c.File)
		http.Error(w, fmt.Sprintf("No available sources: %s", c.File), 501)
		return
	}
	u := urls[rand.Intn(len(urls))]
	log.Warningf("Redircting: %s => %s", c.File, u)
	http.Redirect(w, r, u, 302)
}

func (c *FileController) List(w http.ResponseWriter, r *http.Request) {
	dao := &dao.HealthDao{Context: r.Context()}
	files, err := dao.Files()
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	writeJSON(w, r, files)
}
