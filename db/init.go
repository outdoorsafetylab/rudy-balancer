package db

import (
	"io/ioutil"
	"service/config"
	"service/model"
	"time"

	"github.com/crosstalkio/log"
	"gopkg.in/yaml.v2"
)

var db *DB
var ticker *time.Ticker
var done chan bool

func Init(s log.Sugar) error {
	cfg := config.Get()
	filename := cfg.GetString("db.file")
	s.Infof("Loading DB: %s", filename)
	file, err := ioutil.ReadFile(filename)
	if err != nil {
		s.Errorf("Failed to load DB '%s': %s", filename, err.Error())
		return err
	}
	db = &DB{}
	err = yaml.Unmarshal(file, db)
	if err != nil {
		if err != nil {
			return err
		}
	}
	err = db.validate(s)
	if err != nil {
		s.Errorf("Invalid definition of DB '%s': %s", filename, err.Error())
		return err
	}
	go func() {
		_ = db.check(s)
	}()
	check := cfg.GetString("db.check")
	if check != "" {
		du, err := time.ParseDuration(check)
		if err != nil {
			s.Errorf("Invalid duration: %s", check)
			return err
		}
		s.Infof("Scheduling heath check every %s", check)
		ticker = time.NewTicker(du)
		done = make(chan bool)
		go func() {
			for {
				select {
				case <-done:
					return
				case t := <-ticker.C:
					s.Infof("Renewing DB at %s", t.String())
					_ = db.check(s)
				}
			}
		}()
	}
	return nil
}

func Deinit(s log.Sugar) {
	if ticker != nil {
		ticker.Stop()
		done <- true
		ticker = nil
	}
}

func GetSites() []*model.Site {
	return db.Sites
}

func GetApps() []*model.App {
	return db.Apps
}
