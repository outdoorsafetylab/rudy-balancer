package dao

import (
	"context"
	"time"

	"service/db"
	"service/hash"
	"service/log"
	"service/mirror"
	"service/model"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"
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
	fileID, err := hash.SHA1([]byte(file))
	if err != nil {
		return nil, err
	}
	fileSites := &fileSites{}
	doc, err := db.Client().Collection(rudyMirrorFiles).Doc(fileID).Get(dao.Context)
	if err != nil {
		return nil, err
	}
	err = doc.DataTo(fileSites)
	if err != nil {
		return nil, err
	}
	sites, err := mirror.Sites()
	if err != nil {
		return nil, err
	}
	oneMonthAgo := time.Now().Add(-time.Hour * 24 * 31)
	log.Debugf("%v", oneMonthAgo)
	sources := make([]*model.Source, 0)
	for _, site := range sites {
		size := fileSites.Sites[site.Name]
		if size <= 0 {
			continue
		}
		if site.MonthlyQuota > 0 {
			usage := int64(0)
			q := db.Client().Collection(rudyMirrorSiteStats).Doc(site.Name).Collection(dailyStats).Where("Time", ">=", oneMonthAgo)
			iter := q.Documents(dao.Context)
			defer iter.Stop()
			for {
				doc, err := iter.Next()
				if err == iterator.Done {
					break
				}
				if err != nil {
					return nil, err
				}
				st := &FileStat{}
				err = doc.DataTo(st)
				if err != nil {
					return nil, err
				}
				usage += st.Size
			}
			if usage > site.MonthlyQuota {
				log.Infof("Over quota: %s: quata=%d, usage=%d", site.Name, site.MonthlyQuota, usage)
				continue
			}
		}
		for _, src := range site.Sources {
			if src.File == file {
				src.Size = size
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
		for i := 0; i < weight; i++ {
			weightedSources = append(weightedSources, src)
		}
	}
	return weightedSources, nil
}

type FileStat struct {
	Time  *time.Time `json:",omitempty" firestore:",omitempty"`
	Count int64      `json:",omitempty" firestore:",omitempty"`
	Size  int64
}

func (dao *FileDao) AccumulateRedirect(src *model.Source) error {
	today := time.Unix(time.Now().Unix()/86400*86400, 0).UTC()
	dayID := today.Format("2006-01-02")
	docs := []*firestore.DocumentRef{
		db.Client().Collection(rudyMirrorSiteStats).Doc(allSites).Collection(dailyStats).Doc(dayID),
		db.Client().Collection(rudyMirrorSiteStats).Doc(src.Site.Name).Collection(dailyStats).Doc(dayID),
	}
	for _, docRef := range docs {
		err := db.Client().RunTransaction(dao.Context, func(ctx context.Context, tx *firestore.Transaction) error {
			_, err := tx.Get(docRef)
			if err != nil {
				if status.Code(err) != codes.NotFound {
					log.Errorf("Failed to get document: %s", err.Error())
					return err
				} else {
					stats := FileStat{
						Time:  &today,
						Count: 1,
						Size:  src.Size,
					}
					err = tx.Set(docRef, &stats)
					if err != nil {
						return err
					}
				}
			} else {
				err = tx.Update(docRef, []firestore.Update{
					{FieldPath: []string{"Time"}, Value: today},
					{FieldPath: []string{"Count"}, Value: firestore.Increment(1)},
					{FieldPath: []string{"Size"}, Value: firestore.Increment(src.Size)},
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
	}
	return nil
}

func (dao *FileDao) DailyStats(site string, since, until time.Time) ([]*FileStat, error) {
	if site == "" {
		site = allSites
	}
	res := make([]*FileStat, 0)
	q := db.Client().Collection(rudyMirrorSiteStats).Doc(site).Collection(dailyStats).Where("Time", ">=", since).Where("Time", "<=", until)
	iter := q.Documents(dao.Context)
	defer iter.Stop()
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		stats := &FileStat{}
		err = doc.DataTo(&stats)
		if err != nil {
			return nil, err
		}
		res = append(res, stats)
	}
	return res, nil
}
