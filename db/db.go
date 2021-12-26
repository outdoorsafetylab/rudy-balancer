package db

import (
	"fmt"
	"net/http"
	"service/model"

	"github.com/crosstalkio/log"
)

type DB struct {
	Sites []*model.Site
	Apps  []*model.App
}

func (db *DB) validate(s log.Sugar) error {
	if len(db.Sites) <= 0 {
		return fmt.Errorf("No sites defined")
	}
	for _, site := range db.Sites {
		if site.Name == "" {
			return fmt.Errorf("Missing site name")
		}
		if site.Endpoint == "" {
			return fmt.Errorf("Missing site endpoint")
		}
	}
	if len(db.Apps) <= 0 {
		return fmt.Errorf("No apps defined")
	}
	for _, app := range db.Apps {
		if app.ID == "" {
			return fmt.Errorf("Missing app ID")
		}
		if app.Name == "" {
			return fmt.Errorf("Missing app name")
		}
		artifacts := app.GetArtifacts()
		if len(artifacts) <= 0 {
			return fmt.Errorf("No artifacts defined: %s", app.ID)
		}
		for _, a := range artifacts {
			if a.ID == "" {
				return fmt.Errorf("Missing artifact ID")
			}
			if a.Name == "" {
				return fmt.Errorf("Missing artifact name")
			}
			if a.File == "" {
				return fmt.Errorf("Missing artifact file")
			}
		}
	}
	return nil
}

func (db *DB) check(s log.Sugar) error {
	for _, app := range db.Apps {
		s.Infof("Checking app: %s (%s)", app.ID, app.Name)
		for _, a := range app.GetArtifacts() {
			s.Infof("Checking artifact: %s (%s)", a.ID, a.Name)
			urls, err := a.GetURLs(db.Sites)
			if err != nil {
				s.Errorf("Failed to get URLs for health check: %s", a.GetID())
				return err
			}
			size := int64(0)
			count := 0
			for _, u := range urls {
				s.Infof("Checking URL: %s", u)
				res, err := http.Head(u.String())
				if err != nil {
					s.Errorf("Bad artifact: %s => %s: %s", a.GetID(), u.String(), err.Error())
					continue
				}
				defer res.Body.Close()
				if res.StatusCode != 200 {
					s.Errorf("Bad artifact: %s => %s: %s", a.GetID(), u.String(), res.Status)
					continue
				}
				size += res.ContentLength
				count++
				res.Body.Close()
			}
			a.ContentLength = size
		}
	}
	return nil
}
