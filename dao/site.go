package dao

import (
	"context"
	"fmt"
	"net/http"
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

type SiteDao struct {
	Context context.Context
}

type site struct {
	Sources   map[string]*model.Source
	LastCheck time.Time
	Latency   time.Duration
}

func (dao *SiteDao) load() (*mirror.Mirror, error) {
	mirror, err := mirror.Get()
	if err != nil {
		return nil, err
	}
	for _, s := range mirror.Sites {
		log.Debugf("Loading site: %s (%s)", s.Name, s.Firestore)
		doc, err := firestore.Collection().Doc(s.Firestore).Get(dao.Context)
		if err != nil {
			if status.Code(err) != codes.NotFound {
				log.Errorf("Failed to load document %s: %s", s.Firestore, err.Error())
				return nil, err
			} else {
				log.Debugf("Document not found: %s", s.Firestore)
				for _, s := range s.Sources {
					s.Status = model.UNKNWON
				}
				continue
			}
		}
		saved := &site{}
		err = doc.DataTo(&saved)
		if err != nil {
			return nil, err
		}
		for _, s := range s.Sources {
			ss := saved.Sources[s.URL]
			if ss == nil {
				log.Warnf("Source states not available: %s", s.URL)
				s.Status = model.UNKNWON
				continue
			}
			s.LastCheck = ss.LastCheck
			s.LastCheckUnix = s.LastCheck.Unix()
			if s.LastCheckUnix < 0 {
				s.LastCheckUnix = 0
			}
			s.Status = ss.Status
			s.Size = ss.Size
			s.Latency = ss.Latency
			s.LastModified = ss.LastModified
			s.LastModifiedUnix = s.LastModified.Unix()
			if s.LastModifiedUnix < 0 {
				s.LastModifiedUnix = 0
			}
		}
	}
	return mirror, nil
}

func (dao *SiteDao) Apps() ([]*model.App, error) {
	mirror, err := dao.load()
	if err != nil {
		return nil, err
	}
	return mirror.Apps, nil
}

func (dao *SiteDao) Sites() ([]*model.Site, error) {
	mirror, err := dao.load()
	if err != nil {
		return nil, err
	}
	return mirror.Sites, nil
}

func (dao *SiteDao) Update(sites []*model.Site) error {
	cfg := config.Get()
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
	for _, s := range sites {
		log.Debugf("Updating site: %s", s.Name)
		site := &site{
			Sources: make(map[string]*model.Source),
		}
		doc, err := firestore.Collection().Doc(s.Firestore).Get(dao.Context)
		if err != nil {
			if status.Code(err) != codes.NotFound {
				log.Errorf("Failed to get document %s: %s", s.Firestore, err.Error())
				return err
			}
		} else {
			err = doc.DataTo(&site)
			if err != nil {
				return err
			}
		}
		site.LastCheck = time.Now()
		goods := make([]*model.Source, 0)
		var latency time.Duration
		bads := make([]*model.Source, 0)
		for _, s := range s.Sources {
			site.Sources[s.URL] = s
			switch s.Status {
			case model.GOOD:
				goods = append(goods, s)
				latency += s.Latency
			case model.BAD:
				bads = append(bads, s)
			}
		}
		if len(goods) > 0 {
			site.Latency = latency / time.Duration(len(goods))
		}
		_, err = doc.Ref.Set(dao.Context, site)
		if err != nil {
			return err
		}
		if s.Hidden {
			continue
		}
		comp := componentsByName[s.StatusPage]
		if comp == nil {
			log.Warnf("Creating component: %s", s.StatusPage)
			comp, err = client.CreateComponent(pageID, groupID, s.StatusPage)
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
		log.Debugf("Updating component status: %s => %s", s.StatusPage, status)
		err = client.UpdateComponentStatus(comp, status)
		if err != nil {
			return err
		}
		if comp.Status == "operational" && status != comp.Status {
			log.Warnf("Creating incident due to site %s is not operational", s.Name)
			_, err = client.CreateIncident(pageID, comp.ID, fmt.Sprintf("%s is not operational.", s.Name), bads)
			if err != nil {
				return err
			}
		} else if status == "operational" && status != comp.Status {
			log.Warnf("Resolving incident due to site %s is back", s.Name)
			err = client.ResolveIncidents(pageID, comp.ID)
			if err != nil {
				return err
			}
		}
	}
	return nil
}
