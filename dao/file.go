package dao

import (
	"service/firestore"
	"service/model"

	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type FileDao struct {
	SiteDao
}

func (dao *FileDao) Files() (map[string][]*model.Source, error) {
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

func (dao *FileDao) GetSources(file string) ([]*model.Source, error) {
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
	weightedSources := make([]*model.Source, 0)
	for _, src := range sources {
		weight := weights[src.URL]
		switch src.Status {
		case model.GOOD:
			for i := 0; i < weight; i++ {
				weightedSources = append(weightedSources, src)
			}
		}
	}
	return weightedSources, nil
}

type FileStat struct {
	Count int64
	Size  int64
}

const (
	totalStatsDocID = "StatsTotal"
)

func (dao *FileDao) AccumulateRedirect(src *model.Source) error {
	stats := make(map[string]*FileStat)
	doc, err := firestore.Collection().Doc(totalStatsDocID).Get(dao.Context)
	if err != nil {
		if status.Code(err) != codes.NotFound {
			log.Errorf("Failed to get document %s: %s", totalStatsDocID, err.Error())
			return err
		}
	} else {
		err = doc.DataTo(&stats)
		if err != nil {
			return err
		}
	}
	st := stats[src.File]
	if st == nil {
		st = &FileStat{}
		stats[src.File] = st
	}
	st.Count++
	st.Size += src.Size
	_, err = doc.Ref.Set(dao.Context, stats)
	if err != nil {
		return err
	}
	return nil
}

func (dao *FileDao) TotalStats() (map[string]*FileStat, error) {
	stats := make(map[string]*FileStat)
	doc, err := firestore.Collection().Doc(totalStatsDocID).Get(dao.Context)
	if err != nil {
		if status.Code(err) != codes.NotFound {
			log.Errorf("Failed to get document %s: %s", totalStatsDocID, err.Error())
			return nil, err
		} else {
			return stats, nil
		}
	}
	err = doc.DataTo(&stats)
	if err != nil {
		return nil, err
	}
	return stats, nil
}
