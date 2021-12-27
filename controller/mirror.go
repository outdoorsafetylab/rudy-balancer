package controller

import (
	"math/rand"
	"net/http"
	"service/model"

	"github.com/crosstalkio/rest"
)

type MirrorController struct {
	Artifact *model.Artifact
}

func (c *MirrorController) Get(s *rest.Session) {
	urls := c.Artifact.HealthyURLs
	if urls == nil || len(urls) <= 0 {
		s.Status(501, nil)
		return
	}
	u := urls[rand.Intn(len(urls))]
	http.Redirect(s.ResponseWriter, s.Request, u.String(), 302)
}
