package model

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	log "github.com/sirupsen/logrus"
)

type Site struct {
	Name    string
	Hidden  bool      `json:",omitempty"`
	Scheme  string    `json:"-"`
	Host    string    `json:"-"`
	Prefix  string    `json:"-"`
	Sources []*Source `json:"-"`
	Weight  int       `json:"-"`
}

func (s *Site) GetURL(uri string) string {
	if uri != "" {
		if !strings.HasPrefix(uri, "/") {
			uri = "/" + uri
		}
	}
	return fmt.Sprintf("%s://%s%s%s", s.Scheme, s.Host, s.Prefix, uri)
}

func (s *Site) Probe(client *http.Client, uri string) error {
	url := s.GetURL(uri)
	log.Debugf("HEAD: %s", url)
	res, err := client.Get(url)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode >= 400 {
		return errors.New(res.Status)
	}
	return nil
}
