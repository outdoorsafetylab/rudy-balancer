package controller

import (
	"net/http"
	"service/version"
)

type ConfigController struct{}

func (c *ConfigController) GetVersion(w http.ResponseWriter, r *http.Request) {
	res := &struct {
		Commit string `json:"commit"`
		Tag    string `json:"tag"`
	}{
		Commit: version.GitHash,
		Tag:    version.GitTag,
	}
	writeJSON(w, r, res)
}
