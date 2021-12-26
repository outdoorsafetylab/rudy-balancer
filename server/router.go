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

	for _, app := range db.GetApps() {
		for _, a := range app.GetArtifacts() {
			path := fmt.Sprintf("/mirror/%s/%s", a.App.ID, a.ID)
			if a.Variant != nil {
				path = fmt.Sprintf("/mirror/%s/%s/%s", a.App.ID, a.Variant.ID, a.ID)
			}
			mirror := &controller.MirrorController{
				Artifact: a,
			}
			endpoint.Methods("GET").Path(path).Handler(rest.HandlerFunc(mirror.Get))
			endpoint.Methods("HEAD").Path(path).Handler(rest.HandlerFunc(mirror.Get))
		}
	}

	if root != nil {
		r.NotFoundHandler = http.FileServer(root)
	}
	return r
}
