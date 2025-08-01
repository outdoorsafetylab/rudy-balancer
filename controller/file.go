package controller

import (
	"context"
	"fmt"
	"math/rand"
	"net/http"
	"time"

	"service/config"
	"service/dao"
	"service/geoip"
	"service/log"
	"service/statuspage"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type FileController struct {
	File string
}

func (c *FileController) Download(w http.ResponseWriter, r *http.Request) {
	if r.UserAgent() == "okhttp/4.7.2" {
		return
	}
	start := time.Now()
	dao := &dao.FileDao{SiteDao: dao.SiteDao{Context: r.Context()}}
	sources, err := dao.GetSources(c.File)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if len(sources) <= 0 {
		log.Warningf("No available sources: %s", c.File)
		http.Error(w, fmt.Sprintf("No available sources: %s", c.File), http.StatusBadGateway)
		return
	}
	src := sources[rand.Intn(len(sources))]
	http.Redirect(w, r, src.URL, http.StatusFound)
	stop := time.Now()
	if r.Method != "GET" {
		return
	}
	go func() {
		fields := []zapcore.Field{
			zap.String("Method", r.Method),
			zap.String("Referer", r.Referer()),
			zap.String("UserAgent", r.UserAgent()),
			zap.String("SiteName", src.SiteName),
			zap.String("File", src.File),
			zap.Int64("Size", src.Size),
		}
		country, err := geoip.Country(r)
		if err == nil {
			fields = append(fields, zap.String("Country", country.Country.IsoCode))
		} else {
			log.Warningf("Failed to determine country: %s", err.Error())
		}
		log.Write(log.Info, fmt.Sprintf("Redirecting: %s => %s", c.File, src.URL), fields...)
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
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, r, files)
}
