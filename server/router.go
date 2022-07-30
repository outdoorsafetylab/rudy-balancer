package server

import (
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"service/config"
	"service/controller"
	"service/middleware"
	"service/mirror"
	"time"

	"github.com/gorilla/mux"
)

func newRouter(webroot string) (*mux.Router, error) {
	cfg := config.Get()

	r := mux.NewRouter()
	r.PathPrefix("/app").Handler(&webrootHandler{
		path: webroot,
	})

	prefix := cfg.GetString("endpoint")
	endpoint := r.PathPrefix(prefix).Subrouter()
	endpoint.Use(middleware.Dump)
	endpoint.Use(middleware.NoCache)

	config := &controller.ConfigController{}
	endpoint.HandleFunc("/version", config.GetVersion).Methods("GET")

	health := &controller.HealthController{}
	endpoint.HandleFunc("/healthcheck", health.Check).Methods("GET")

	qrcode := &controller.QRCodeController{
		Cache: map[string][]byte{},
	}
	endpoint.HandleFunc("/qrcode", qrcode.Generate).Methods("GET", "HEAD")

	mirror, err := mirror.Get()
	if err != nil {
		return nil, err
	}
	for _, files := range mirror.Files {
		for _, file := range files {
			c := &controller.FileController{File: file}
			endpoint.HandleFunc(fmt.Sprintf("/%s", file), c.Download).Methods("GET", "HEAD")
		}
	}
	app := &controller.AppController{}
	endpoint.HandleFunc("/apps", app.List).Methods("GET")
	site := &controller.SiteController{}
	endpoint.HandleFunc("/sites", site.List).Methods("GET")
	file := &controller.FileController{}
	endpoint.HandleFunc("/files", file.List).Methods("GET")
	reverseProxy := &proxyHandler{
		ProbeClient: http.Client{Timeout: time.Second},
		Redirects:   make(map[string]bool),
	}
	for _, files := range mirror.Files {
		for _, file := range files {
			reverseProxy.Redirects["/"+file] = true
		}
	}
	for _, site := range mirror.Sites {
		url, err := url.Parse(site.GetURL(""))
		if err != nil {
			return nil, err
		}
		for i := 0; i < site.Weight; i++ {
			reverseProxy.Targets = append(reverseProxy.Targets, &proxyTarget{
				Site:  site,
				Proxy: httputil.NewSingleHostReverseProxy(url),
			})
		}
	}
	r.NotFoundHandler = reverseProxy
	return r, nil
}
