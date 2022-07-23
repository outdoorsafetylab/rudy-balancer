package dao

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"service/config"
	"service/firestore"
	"service/mirror"
	"service/model"
	"service/statuspage"
	"time"

	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type HealthDao struct {
	Context context.Context
}

type site struct {
	Sources   map[string]*model.Source
	LastCheck time.Time
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
			if status.Code(err) != codes.NotFound {
				log.Errorf("Failed to get site %s: %s", name, err.Error())
				return nil, err
			} else {
				continue
			}
		}
		site := &site{}
		err = doc.DataTo(&site)
		if err != nil {
			return nil, err
		}
		for url, source := range site.Sources {
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
	groupID := cfg.GetString("statuspage.group")
	components, err := client.ListComponents(pageID)
	if err != nil {
		log.Errorf("Failed to list components: %s", err.Error())
		return err
	}
	componentsByName := make(map[string]*statuspage.Component)
	for _, comp := range components {
		componentsByName[comp.Name] = comp
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
		site := &site{
			LastCheck: time.Now(),
			Sources:   make(map[string]*model.Source),
		}
		goods := make([]string, 0)
		bads := make([]string, 0)
		for _, a := range artifacts {
			for _, s := range a.Sources {
				if name != s.Site.Name {
					continue
				}
				url := s.URL.String()
				site.Sources[url] = s
				switch s.Status {
				case model.GOOD:
					goods = append(goods, url)
				case model.BAD:
					bads = append(bads, url)
				}
			}
		}
		_, err = doc.Ref.Set(dao.Context, site)
		if err != nil {
			return err
		}
		comp := componentsByName[name]
		if comp == nil {
			log.Warnf("Creating component: %s", name)
			comp, err = client.CreateComponent(pageID, groupID, name)
			if err != nil {
				return err
			}
		}
		var status string
		percentage := 100.0 * len(goods) / (len(goods) + len(bads))
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
		if comp.Status == "operational" && status != comp.Status {
			log.Warnf("Creating incident due to %s is not operational", name)
			_, err = client.CreateIncident(pageID, comp.ID, fmt.Sprintf("%s is not operational.", name), bads)
			if err != nil {
				return err
			}
		} else if status == "operational" && status != comp.Status {
			log.Warnf("Resolving incident due to %s is back", name)
			err = client.ResolveIncidents(pageID, comp.ID)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (dao *HealthDao) GetURLs(artifact *model.Artifact) ([]*url.URL, error) {
	cfg := config.Get()
	sites := make(map[string]*site)
	weights := make(map[string]int)
	for _, src := range artifact.Sources {
		if sites[src.Site.Name] != nil {
			continue
		}
		sites[src.Site.Name] = &site{
			Sources: make(map[string]*model.Source),
		}
		weights[src.Site.Name] = src.Site.Weight
	}
	for name, site := range sites {
		log.Debugf("Getting site status: %s", name)
		doc, err := firestore.Client().Collection(cfg.GetString("firestore.collection")).Doc(name).Get(dao.Context)
		if err != nil {
			log.Warnf("Failed to get site %s: %s", name, err.Error())
			continue
		}
		if doc.Exists() {
			err = doc.DataTo(site)
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
		src := site.Sources[source.URL.String()]
		if src == nil {
			log.Warnf("Health check record not found for %s", source.URL.String())
			continue
		}
		weight := weights[source.Site.Name]
		switch src.Status {
		case model.GOOD:
			for i := 0; i < weight; i++ {
				urls = append(urls, source.URL)
			}
		}
	}
	return urls, nil
}
