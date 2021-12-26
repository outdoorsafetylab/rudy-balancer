package controller

import (
	"service/api"
	"service/version"
	"time"

	"github.com/crosstalkio/rest"
)

type ConfigController struct{}

func (c *ConfigController) Get(s *rest.Session) {
	v := version.Get()
	res := &api.GetVersionResponse{
		Time: &api.GetVersionResponse_Time{
			Epoch:   v.Time.Unix(),
			Rfc3339: v.Time.Format(time.RFC3339),
		},
		Commit: v.Commit,
		Tag:    v.Tag,
	}
	s.Status(200, res)
}
