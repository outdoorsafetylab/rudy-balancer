package controller

import (
	"encoding/json"
	"net/http"
	"service/api"
	"service/version"
)

type ConfigController struct{}

func (c *ConfigController) GetVersion(w http.ResponseWriter, r *http.Request) {
	res := &api.GetVersionResponse{
		Commit: version.GitHash,
		Tag:    version.GitTag,
	}
	enc := json.NewEncoder(w)
	err := enc.Encode(res)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	w.Header().Set("Content-Type", "application/json")
}
