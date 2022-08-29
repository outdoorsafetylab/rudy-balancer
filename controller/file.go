package controller

import (
	"context"
	"fmt"
	"math/rand"
	"net/http"
	"time"

	"service/config"
	"service/dao"
	"service/log"
	"service/statuspage"

	"go.uber.org/zap"
)

type FileController struct {
	File string
}

func (c *FileController) Download(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
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
	http.Redirect(w, r, src.URL, 302)
	stop := time.Now()
	if r.Method != "GET" {
		return
	}
	log.Write(log.Info, fmt.Sprintf("Redircting: %s => %s", c.File, src.URL),
		zap.String("UserAgent", r.UserAgent()),
		zap.String("SiteName", src.SiteName),
		zap.String("File", src.File),
		zap.Int64("Size", src.Size))
	go func() {
		dao.Context = context.Background()
		err = dao.AccumulateRedirect(src)
		if err != nil {
			log.Errorf("Failed to accumuate redirect: %s", err.Error())
		}
		cfg := config.Get()
		metricID := cfg.GetString("statuspage.metrics.redirect_time")
		if metricID == "" {
			log.Warnf("Missing metric ID for redirect time")
			return
		}
		client := &statuspage.Client{Client: http.DefaultClient, APIKey: cfg.GetString("statuspage.key")}
		pageID := cfg.GetString("statuspage.page")
		err = client.AddDataPoint(pageID, metricID, &statuspage.DataPoint{
			Time:  start,
			Value: float64(stop.Sub(start).Milliseconds()),
		})
		if err != nil {
			log.Errorf("Failed to add metric for redirect time: %s", err.Error())
		}
	}()
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
