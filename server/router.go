package server

import (
	"fmt"
	"net/http"
	"service/config"
	"service/controller"
	"service/db"
	"service/middleware"

	"github.com/crosstalkio/log"
	"github.com/crosstalkio/rest"
	"github.com/gorilla/mux"
)

func NewRouter(s log.Logger, root http.FileSystem) *mux.Router {
	cfg := config.Get()
	rest := rest.NewServer(s)
	rest.Use(middleware.Dump)
	rest.Use(middleware.NoCache)

	r := mux.NewRouter()

	endpoint := r.PathPrefix(cfg.GetString("endpoint")).Subrouter()

	config := &controller.ConfigController{}
	endpoint.Methods("GET").Path("/version").Handler(rest.HandlerFunc(config.Get))

	apps := &controller.AppsController{}
	endpoint.Methods("GET").Path("/apps").Handler(rest.HandlerFunc(apps.Get))

	for _, app := range db.GetApps() {
		for _, a := range app.GetArtifacts() {
			path := fmt.Sprintf("/mirror/%s", a.GetPath())
			mirror := &controller.MirrorController{
				Artifact: a,
			}
			endpoint.Methods("GET").Path(path).Handler(rest.HandlerFunc(mirror.Get))
			endpoint.Methods("HEAD").Path(path).Handler(rest.HandlerFunc(mirror.Get))
			qrcode := &controller.QRCodeController{
				Artifact: a,
			}
			path = fmt.Sprintf("/qrcode/%s", a.GetPath())
			endpoint.Methods("GET").Path(path).Handler(rest.HandlerFunc(qrcode.Get))
		}
	}

	if root != nil {
		r.NotFoundHandler = http.FileServer(root)
	}
	return r
}
