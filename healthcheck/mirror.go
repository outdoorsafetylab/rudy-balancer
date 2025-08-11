package healthcheck

import (
	"context"
	"fmt"
	"net/http"
	"service/dao"
	"service/log"
	"service/mirror"
	"time"

	"github.com/spf13/viper"
)

func CheckMirrorSites(cfg *viper.Viper, ctx context.Context) error {
	dao := &dao.SiteDao{Context: ctx}
	sites, err := mirror.Sites()
	if err != nil {
		return fmt.Errorf("failed to get sites: %w", err)
	}

	client := &http.Client{
		Timeout: time.Duration(cfg.GetInt("healthcheck.timeout_sec")) * time.Second,
	}

	for _, site := range sites {
		for _, s := range site.Sources {
			log.Debugf("Checking source: %s", s.URL)
			_ = s.Check(client)
			log.Debugf("%s => %s @ %v", s.URL, s.Status.String(), s.Latency)
		}
	}

	err = dao.Update(sites)
	if err != nil {
		return fmt.Errorf("failed to update sites: %w", err)
	}

	return nil
}
