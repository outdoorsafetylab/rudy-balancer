package dao

import (
	"context"
	"net/http"
	"net/url"
	"service/config"
	"service/firestore"
	"service/mirror"
	"service/model"
	"service/statuspage"

	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type HealthDao struct {
	Context context.Context
}

func (dao *HealthDao) Apps() ([]*model.App, error) {
	apps, err := mirror.Apps()
	if err != nil {
		return nil, err
	}
	cfg := config.Get()
	sites := make(map[string]*model.Site)
	for _, app := range apps {
		for _, v := range app.Variants {
			for _, a := range v.Artifacts {
				for _, s := range a.Sources {
					if sites[s.Site.Name] != nil {
						continue
					}
					sites[s.Site.Name] = s.Site
				}
			}
		}
	}
	sources := make(map[string]*model.Source)
	for name := range sites {
		log.Debugf("Getting site status: %s", name)
		doc, err := firestore.Client().Collection(cfg.GetString("firestore.collection")).Doc(name).Get(dao.Context)
		if err != nil {
			log.Errorf("Failed to get site %s: %s", name, err.Error())
			return nil, err
		}
		data := make(map[string]*model.Source)
		err = doc.DataTo(&data)
		if err != nil {
			return nil, err
		}
		for url, source := range data {
			sources[url] = source
		}
	}
	for _, app := range apps {
		for _, v := range app.Variants {
			for _, a := range v.Artifacts {
				size := int64(0)
				count := 0
				for _, s := range a.Sources {
					ss := sources[s.URL.String()]
					if ss == nil {
						log.Warnf("No such source: %s", s.URL.String())
						continue
					}
					s.Status = ss.Status
					s.Size = ss.Size
					s.LastModified = ss.LastModified
					s.LastModifiedUnix = s.LastModified.Unix()
					if s.LastModifiedUnix < 0 {
						s.LastModifiedUnix = 0
					}
					if s.Status == model.GOOD {
						size += s.Size
						count++
					}
				}
				a.Size = size / int64(count)
			}
		}
	}
	return apps, nil
}

func (dao *HealthDao) Update(artifacts []*model.Artifact) error {
	cfg := config.Get()
	sites := make(map[string]*model.Site)
	for _, a := range artifacts {
		for _, s := range a.Sources {
			if sites[s.Site.Name] != nil {
				continue
			}
			sites[s.Site.Name] = s.Site
		}
	}
	client := &statuspage.Client{Client: http.DefaultClient, APIKey: cfg.GetString("statuspage.key")}
	pageID := cfg.GetString("statuspage.page")
	components, err := client.ListComponents(pageID)
	if err != nil {
		log.Errorf("Failed to list components: %s", err.Error())
		return err
	}
	for name := range sites {
		log.Debugf("Updating site: %s", name)
		doc, err := firestore.Client().Collection(cfg.GetString("firestore.collection")).Doc(name).Get(dao.Context)
		if err != nil {
			if status.Code(err) != codes.NotFound {
				log.Errorf("Failed to get site %s: %s", name, err.Error())
				return err
			}
		}
		data := make(map[string]*model.Source)
		goods := 0
		total := 0
		for _, a := range artifacts {
			for _, s := range a.Sources {
				if name != s.Site.Name {
					continue
				}
				data[s.URL.String()] = s
				total++
				switch s.Status {
				case model.GOOD:
					goods++
				}
			}
		}
		_, err = doc.Ref.Set(dao.Context, &data)
		if err != nil {
			return err
		}
		for _, comp := range components {
			if comp.Name != name {
				continue
			}
			var status string
			percentage := 100.0 * goods / total
			if percentage >= 100 {
				status = "operational"
			} else if percentage <= 0 {
				status = "major_outage"
			} else {
				status = "partial_outage"
			}
			log.Debugf("Updating status: %s => %s", name, status)
			err = client.UpdateComponentStatus(comp, status)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (dao *HealthDao) GetAvailableURLs(artifact *model.Artifact) ([]*url.URL, error) {
	cfg := config.Get()
	sites := make(map[string]map[string]*model.Source)
	for _, src := range artifact.Sources {
		if sites[src.Site.Name] != nil {
			continue
		}
		sites[src.Site.Name] = make(map[string]*model.Source)
	}
	for name, site := range sites {
		log.Debugf("Getting site status: %s", name)
		doc, err := firestore.Client().Collection(cfg.GetString("firestore.collection")).Doc(name).Get(dao.Context)
		if err != nil {
			if status.Code(err) != codes.NotFound {
				log.Errorf("Failed to get site %s: %s", name, err.Error())
				return nil, err
			}
		}
		if doc.Exists() {
			err = doc.DataTo(&site)
			if err != nil {
				return nil, err
			}
		}
	}
	urls := make([]*url.URL, 0)
	for _, source := range artifact.Sources {
		site := sites[source.Site.Name]
		if site == nil {
			log.Warnf("Site not found for %s: %s", source.URL.String(), source.Site.Name)
			continue
		}
		saved := site[source.URL.String()]
		if saved == nil {
			log.Warnf("Health check record not found for %s", source.URL.String())
			continue
		}
		switch saved.Status {
		case model.GOOD:
			urls = append(urls, source.URL)
		}
	}
	return urls, nil
}
