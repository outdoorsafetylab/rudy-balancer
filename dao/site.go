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
	File     string
	Sites    map[string]int64
	Modified map[string]time.Time `firestore:",omitempty"`
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
	modified := make(map[string]map[string]time.Time)
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
					if !src.LastModified.IsZero() {
						if modified[src.File] == nil {
							modified[src.File] = make(map[string]time.Time)
						}
						modified[src.File][s.Name] = src.LastModified
					}
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
				File:     file,
				Sites:    sites,
				Modified: modified[file],
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

	observed := make([]statuspage.Observation, 0, len(sites))
	for _, s := range sites {
		if s.Hidden {
			continue
		}
		st := statuses[s.Name]
		if st.status == "" {
			log.Debugf("Unknown status: %s", s.Name)
			continue
		}
		badResources := make(map[string]error)
		for _, src := range st.bads {
			badResources[src.URL] = src.Error
		}
		observed = append(observed, statuspage.Observation{
			Component: s.StatusPage,
			Status:    st.status,
			Incident:  fmt.Sprintf("%s is not operational.", s.Name),
			Bad:       badResources,
		})
	}
	client := &statuspage.Client{Client: http.DefaultClient, APIKey: apiKey}
	return client.SyncPage(pageID, cfg.GetString("statuspage.group"), observed)
}
