package controller

import (
	"fmt"
	"math/rand"
	"net/http"
	"time"

	"service/dao"
	"service/log"
)

type FileController struct {
	File string
}

func (c *FileController) Download(w http.ResponseWriter, r *http.Request) {
	dao := &dao.FileDao{SiteDao: dao.SiteDao{Context: r.Context()}}
	sources, err := dao.GetSources(c.File)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	if sources == nil || len(sources) <= 0 {
		log.Warningf("No available sources: %s", c.File)
		http.Error(w, fmt.Sprintf("No available sources: %s", c.File), 502)
		return
	}
	rand.Seed(time.Now().UnixNano())
	src := sources[rand.Intn(len(sources))]
	log.Warningf("Redircting: %s => %s", c.File, src.URL)
	http.Redirect(w, r, src.URL, 302)
	if r.Method == "GET" {
		err = dao.AccumulateRedirect(src)
		if err != nil {
			log.Errorf("Failed to accumuate redirect: %s", err.Error())
		}
	}
}

func (c *FileController) List(w http.ResponseWriter, r *http.Request) {
	dao := &dao.FileDao{SiteDao: dao.SiteDao{Context: r.Context()}}
	files, err := dao.Files()
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	writeJSON(w, r, files)
}
