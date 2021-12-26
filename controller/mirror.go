package controller

import (
	"math/rand"
	"net/http"
	"service/db"
	"service/model"

	"github.com/crosstalkio/rest"
)

type MirrorController struct {
	Artifact *model.Artifact
}

func (c *MirrorController) Get(s *rest.Session) {
	urls, err := c.Artifact.GetURLs(db.GetSites())
	if err != nil {
		s.Status(500, err)
		return
	}
	u := urls[rand.Intn(len(urls))]
	http.Redirect(s.ResponseWriter, s.Request, u.String(), 302)
}
