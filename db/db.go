package db

import (
	"fmt"
	"image"
	"net/http"
	"net/url"
	"service/model"

	_ "image/jpeg"
	_ "image/png"

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
		if len(app.Variants) <= 0 {
			return fmt.Errorf("No variants defined")
		}
		for _, v := range app.Variants {
			if v.ID == "" {
				return fmt.Errorf("Missing variant ID")
			}
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
			if a.Icon != "" {
				res, err := http.Get(a.Icon)
				if err != nil {
					s.Errorf("Failed to get icon: %s => %s: %s", a.GetPath(), a.Icon, err.Error())
					return err
				}
				defer res.Body.Close()
				img, _, err := image.Decode(res.Body)
				if err != nil {
					if err != nil {
						s.Errorf("Failed to decode icon: %s => %s: %s", a.GetPath(), a.Icon, err.Error())
						return err
					}
				}
				a.IconImage = img
			}
			healthyUrls := make([]*url.URL, 0)
			size := int64(0)
			count := 0
			healths := make([]*model.Health, len(db.Sites))
			for i, site := range db.Sites {
				urls, err := site.GetURLs(a)
				if err != nil {
					s.Errorf("Failed to get URLs for health check: site=%s, artifact=%s", site.Name, a.GetPath())
					return err
				}
				healthyCount := 0
				for _, u := range urls {
					s.Infof("Checking URL: %s", u)
					res, err := http.Head(u.String())
					if err != nil {
						s.Errorf("Bad artifact: %s => %s: %s", a.GetPath(), u.String(), err.Error())
						continue
					}
					defer res.Body.Close()
					if res.StatusCode != 200 {
						s.Errorf("Bad artifact: %s => %s: %s", a.GetPath(), u.String(), res.Status)
						continue
					}
					size += res.ContentLength
					count++
					healthyCount++
					healthyUrls = append(healthyUrls, u)
					res.Body.Close()
				}
				health := &model.Health{
					Site: site,
				}
				if healthyCount <= 0 {
					health.Status = model.DEAD
				} else if healthyCount == len(urls) {
					health.Status = model.GOOD
				} else {
					health.Status = model.BAD
				}
				healths[i] = health
			}
			a.ContentLength = size / int64(count)
			a.Healths = healths
			a.HealthyURLs = healthyUrls
		}
	}
	return nil
}
