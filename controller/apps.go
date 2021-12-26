package controller

import (
	"encoding/json"
	"service/db"

	"github.com/crosstalkio/rest"
)

type AppsController struct{}

func (c *AppsController) Get(s *rest.Session) {
	apps := db.GetApps()
	enc := json.NewEncoder(s.ResponseWriter)
	err := enc.Encode(apps)
	if err != nil {
		s.Status(500, err)
		return
	}
	s.Status(200, nil)
}
