package dao

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"service/config"
	"service/db"
	"service/hash"
	"service/log"
	"service/mirror"
	"service/model"
	"service/statuspage"

	"cloud.google.com/go/firestore"
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

type siteStatus struct {
	status string
	goods  []*model.Source
	bads   []*model.Source
}

type fileSites struct {
	File  string
	Sites map[string]int64
}

func (dao *SiteDao) load() (*mirror.Mirror, error) {
	mirror, err := mirror.Get()
	if err != nil {
		return nil, err
	}
	for _, s := range mirror.Sites {
		log.Debugf("Loading site: %s (%s)", s.Name, s.Firestore)
		doc, err := db.Client().Collection(rudyMirrorSites).Doc(s.Firestore).Get(dao.Context)
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
	files := make(map[string]map[string]int64)
	statuses := make(map[string]*siteStatus)
	err := db.Client().RunTransaction(dao.Context, func(ctx context.Context, tx *firestore.Transaction) error {
		for _, s := range sites {
			log.Debugf("Updating site: %s", s.Name)
			docRef := db.Client().Collection(rudyMirrorSites).Doc(s.Firestore)
			site := &site{
				LastCheck: time.Now(),
				Sources:   make(map[string]*model.Source),
			}
			st := &siteStatus{
				goods: make([]*model.Source, 0),
				bads:  make([]*model.Source, 0),
			}
			var latency time.Duration
			for _, src := range s.Sources {
				site.Sources[src.URL] = src
				fileSites := files[src.File]
				if fileSites == nil {
					fileSites = make(map[string]int64)
				}
				switch src.Status {
				case model.GOOD:
					st.goods = append(st.goods, src)
					fileSites[s.Name] = src.Size
					latency += src.Latency
				case model.BAD:
					st.bads = append(st.bads, src)
				}
				files[src.File] = fileSites
			}
			if len(st.goods) > 0 {
				site.Latency = latency / time.Duration(len(st.goods))
			}
			err := tx.Set(docRef, site)
			if err != nil {
				return err
			}
			total := len(st.goods) + len(st.bads)
			if total > 0 {
				percentage := 100.0 * len(st.goods) / total
				if percentage >= 100 {
					st.status = "operational"
				} else if percentage <= 0 {
					st.status = "major_outage"
				} else {
					st.status = "partial_outage"
				}
			}
			statuses[s.Name] = st
		}
		for file, sites := range files {
			fileID, err := hash.SHA1([]byte(file))
			if err != nil {
				return err
			}
			docRef := db.Client().Collection(rudyMirrorFiles).Doc(fileID)
			err = tx.Set(docRef, &fileSites{
				File:  file,
				Sites: sites,
			})
			if err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return err
	}
	cfg := config.Get()
	pageID := cfg.GetString("statuspage.page")
	apiKey := cfg.GetString("statuspage.key")

	// Skip statuspage integration if not configured
	if pageID == "" || apiKey == "" {
		log.Debugf("StatusPage not configured, skipping integration")
		return nil
	}

	client := &statuspage.Client{Client: http.DefaultClient, APIKey: apiKey}
	groupID := cfg.GetString("statuspage.group")
	components, err := client.ListComponents(pageID)
	if err != nil {
		log.Errorf("Failed to list components: %s", err.Error())
		log.Warnf("StatusPage integration failed, but continuing with site updates")
		return nil // Don't fail the entire update process
	}
	componentsByName := make(map[string]*statuspage.Component)
	for _, comp := range components {
		componentsByName[comp.Name] = comp
	}
	for _, s := range sites {
		if s.Hidden {
			continue
		}
		st := statuses[s.Name]
		if st.status == "" {
			log.Debugf("Unknown status: %s", s.Name)
			continue
		}
		log.Debugf("Updating statuspage: %s", s.Name)
		comp := componentsByName[s.StatusPage]
		if comp == nil {
			log.Warnf("Creating component: %s", s.StatusPage)
			comp, err = client.CreateComponent(pageID, groupID, s.StatusPage)
			if err != nil {
				log.Errorf("Failed to create component %s: %s", s.StatusPage, err.Error())
				continue // Skip this site but continue with others
			}
		}
		log.Debugf("Updating component status: %s => %s", s.StatusPage, st.status)
		err = client.UpdateComponentStatus(comp, st.status)
		if err != nil {
			log.Errorf("Failed to update component status for %s: %s", s.StatusPage, err.Error())
			continue // Skip this site but continue with others
		}
		if comp.Status == "operational" && st.status != comp.Status {
			log.Warnf("Creating incident due to site %s is not operational", s.Name)
			badResources := make(map[string]error)
			for _, src := range st.bads {
				badResources[src.URL] = src.Error
			}
			_, err = client.CreateIncident(pageID, comp.ID, fmt.Sprintf("%s is not operational.", s.Name), badResources)
			if err != nil {
				log.Errorf("Failed to create incident for %s: %s", s.Name, err.Error())
				continue // Skip this site but continue with others
			}
		} else if st.status == "operational" && st.status != comp.Status {
			log.Warnf("Resolving incident due to site %s is back", s.Name)
			err = client.ResolveIncidents(pageID, comp.ID)
			if err != nil {
				log.Errorf("Failed to resolve incidents for %s: %s", s.Name, err.Error())
				continue // Skip this site but continue with others
			}
		}
	}
	return nil
}
