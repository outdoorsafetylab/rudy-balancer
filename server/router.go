package server

import (
	"fmt"
	"service/config"
	"service/controller"
	"service/middleware"
	"service/mirror"

	"github.com/gorilla/mux"
)

func NewRouter(webroot string) (*mux.Router, error) {
	cfg := config.Get()

	r := mux.NewRouter()

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
	r.NotFoundHandler = &webrootHandler{
		path: webroot,
	}
	return r, nil
}
