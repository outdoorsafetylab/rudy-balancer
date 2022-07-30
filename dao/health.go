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

type HealthDao struct {
	Context context.Context
}

type site struct {
	Sources   map[string]*model.Source
	LastCheck time.Time
	Latency   time.Duration
}

func (dao *HealthDao) load() (*mirror.Mirror, error) {
	mirror, err := mirror.Get()
	if err != nil {
		return nil, err
	}
	cfg := config.Get()
	for _, s := range mirror.Sites {
		log.Debugf("Loading site: %s (%s)", s.Name, s.Firestore)
		doc, err := firestore.Client().Collection(cfg.GetString("firestore.collection")).Doc(s.Firestore).Get(dao.Context)
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
	for _, app := range mirror.Apps {
		for _, v := range app.Variants {
			for _, a := range v.Artifacts {
				size := int64(0)
				count := 0
				for _, s := range a.Sources {
					if s.Status == model.GOOD {
						size += s.Size
						count++
					}
				}
				a.Size = size / int64(count)
			}
		}
	}
	return mirror, nil
}

func (dao *HealthDao) Apps() ([]*model.App, error) {
	mirror, err := dao.load()
	if err != nil {
		return nil, err
	}
	return mirror.Apps, nil
}

func (dao *HealthDao) Sites() ([]*model.Site, error) {
	mirror, err := dao.load()
	if err != nil {
		return nil, err
	}
	return mirror.Sites, nil
}

func (dao *HealthDao) Files() (map[string][]*model.Source, error) {
	mirror, err := dao.load()
	if err != nil {
		return nil, err
	}
	files := make(map[string][]*model.Source)
	for _, site := range mirror.Sites {
		for _, src := range site.Sources {
			sources := files[src.File]
			if sources == nil {
				sources = make([]*model.Source, 0)
			}
			exist := false
			for _, b := range sources {
				if src == b {
					exist = true
					break
				}
			}
			if !exist {
				sources = append(sources, src)
			}
			files[src.File] = sources
		}
	}
	return files, nil
}

func (dao *HealthDao) Update(sites []*model.Site) error {
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
		doc, err := firestore.Client().Collection(cfg.GetString("firestore.collection")).Doc(s.Firestore).Get(dao.Context)
		if err != nil {
			if status.Code(err) != codes.NotFound {
				log.Errorf("Failed to get document %s: %s", s.Firestore, err.Error())
				return err
			}
		}
		site := &site{
			LastCheck: time.Now(),
			Sources:   make(map[string]*model.Source),
		}
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

func (dao *HealthDao) GetURLs(file string) ([]string, error) {
	mirror, err := dao.load()
	if err != nil {
		return nil, err
	}
	sources := make([]*model.Source, 0)
	for _, site := range mirror.Sites {
		for _, src := range site.Sources {
			if src.File == file {
				sources = append(sources, src)
			}
		}
	}
	// var maxLatency time.Duration
	// for _, src := range sources {
	// 	if src.Site.Hidden {
	// 		continue
	// 	}
	// 	if src.Latency > maxLatency {
	// 		maxLatency = src.Latency
	// 	}
	// }
	weights := make(map[string]int)
	for _, src := range sources {
		if src.Site.Hidden {
			continue
		}
		weights[src.URL] = src.Site.Weight
		// if src.Latency <= 0 {
		// 	continue
		// }
		// weights[src.URL] = int(math.Max(1.0, 100.0*float64(maxLatency)/float64(src.Latency)))
	}
	log.Debugf("URL weights: %v", weights)
	urls := make([]string, 0)
	for _, src := range sources {
		weight := weights[src.URL]
		switch src.Status {
		case model.GOOD:
			for i := 0; i < weight; i++ {
				urls = append(urls, src.URL)
			}
		}
	}
	log.Debugf("URL candidates: %v", urls)
	return urls, nil
}
