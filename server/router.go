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

	artifacts, err := mirror.Artifacts()
	if err != nil {
		return nil, err
	}
	for _, a := range artifacts {
		artifact := &controller.ArtifactController{
			Artifact: a,
		}
		endpoint.HandleFunc(fmt.Sprintf("/%s", a.File), artifact.Download).Methods("GET", "HEAD")
	}

	app := &controller.AppController{}
	endpoint.HandleFunc("/apps", app.List).Methods("GET")
	artifact := &controller.ArtifactController{}
	endpoint.HandleFunc("/artifacts", artifact.List).Methods("GET")
	r.NotFoundHandler = &webrootHandler{
		path: webroot,
	}
	return r, nil
}
