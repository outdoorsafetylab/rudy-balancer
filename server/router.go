package server

import (
	"fmt"
	"net/http/httputil"
	"net/url"
	"service/config"
	"service/controller"
	"service/middleware"
	"service/mirror"

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

	qrcode := &controller.QRCodeController{}
	endpoint.HandleFunc("/qrcode", qrcode.Generate).Methods("GET", "HEAD")

	mirror, err := mirror.Get()
	if err != nil {
		return nil, err
	}
	for _, file := range mirror.Files {
		c := &controller.FileController{File: file}
		endpoint.HandleFunc(fmt.Sprintf("/%s", file), c.Download).Methods("GET", "HEAD")
	}

	app := &controller.AppController{}
	endpoint.HandleFunc("/apps", app.List).Methods("GET")
	site := &controller.SiteController{}
	endpoint.HandleFunc("/sites", site.List).Methods("GET")
	file := &controller.FileController{}
	endpoint.HandleFunc("/files", file.List).Methods("GET")
	// r.NotFoundHandler = newProxy(mirror.Sites)
	target, err := url.Parse(cfg.GetString("proxy.target"))
	if err != nil {
		return nil, err
	}
	r.NotFoundHandler = httputil.NewSingleHostReverseProxy(target)
	return r, nil
}
