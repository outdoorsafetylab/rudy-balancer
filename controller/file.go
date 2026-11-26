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
	"service/model"
	"service/statuspage"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type FileController struct {
	File string
	// Via is the app whose rudymap.tw/app link led here (#19), empty for the
	// plain /v1/<file> route. Many apps download with a library-default
	// User-Agent, so this is the only way to tell them apart.
	Via string
	// sources stands in for dao.FileDao.GetSources in tests.
	sources func(ctx context.Context, file string, corsOnly bool) ([]*model.Source, error)
}

// corsAllowHeaders are the request headers a web page's download may send:
// Range and its validators, for resuming.
const corsAllowHeaders = "Range, If-Range, If-None-Match, If-Modified-Since"

// Download redirects to a mirror holding the file. Web pages may call it too
// (#53): every answer carries Access-Control-Allow-Origin: *, a preflight gets
// a 204, and a request carrying Origin is only sent to a mirror the health
// check found serving the file to web pages, since the browser would block the
// download from any other. Apps and plain downloads send no Origin and are
// redirected as before.
func (c *FileController) Download(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	if r.Method == http.MethodOptions {
		w.Header().Set("Access-Control-Allow-Methods", "GET, HEAD, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", corsAllowHeaders)
		w.Header().Set("Access-Control-Max-Age", "86400")
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if r.UserAgent() == "okhttp/4.7.2" {
		return
	}
	start := time.Now()
	dao := &dao.FileDao{SiteDao: dao.SiteDao{Context: r.Context()}}
	getSources := c.sources
	if getSources == nil {
		getSources = func(_ context.Context, file string, corsOnly bool) ([]*model.Source, error) {
			return dao.GetSources(file, corsOnly)
		}
	}
	corsOnly := r.Header.Get("Origin") != ""
	sources, err := getSources(r.Context(), c.File, corsOnly)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if len(sources) <= 0 {
		if corsOnly {
			log.Warningf("No sources serving web pages: %s", c.File)
			http.Error(w, fmt.Sprintf("No available sources for web pages: %s", c.File), http.StatusBadGateway)
			return
		}
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
		if c.Via != "" {
			fields = append(fields, zap.String("Via", c.Via))
		}
		id, err := clientID(context.Background(), r, start)
		if err == nil {
			fields = append(fields, zap.String("Client", id))
		} else {
			log.Warningf("Failed to derive client id: %s", err.Error())
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
		apiKey := cfg.GetString("statuspage.key")
		pageID := cfg.GetString("statuspage.page")

		// Skip statuspage metrics if not configured
		if metricID == "" || apiKey == "" || pageID == "" {
			log.Debugf("StatusPage metrics not configured, skipping")
			return
		}

		client := &statuspage.Client{Client: http.DefaultClient, APIKey: apiKey}
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
