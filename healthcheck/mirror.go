package healthcheck

import (
	"context"
	"fmt"
	"net/http"
	"service/dao"
	"service/log"
	"service/mirror"
	"service/model"
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
		checkSite(client, site, retryDelay)
	}

	err = dao.Update(sites)
	if err != nil {
		return fmt.Errorf("failed to update sites: %w", err)
	}

	return nil
}

// retryDelay is how long a failed check waits before it is tried again.
const retryDelay = 3 * time.Second

// maxFailedRetries is how many retries may fail on one site before its
// remaining files are no longer retried.
const maxFailedRetries = 3

// checkSite checks each of the site's files. A file that fails is tried once
// more after delay, so a single reset or timeout on an otherwise healthy site
// does not count as an outage. Once maxFailedRetries retries have failed too,
// the site is taken as down and its remaining files are not retried: a dead
// site costs the run a few retries, not one per file, while a file or two
// still missing on a syncing mirror does not stop the retries for the rest.
func checkSite(client *http.Client, site *model.Site, delay time.Duration) {
	failed := 0
	for _, s := range site.Sources {
		log.Debugf("Checking source: %s", s.URL)
		if err := s.Check(client); err != nil && failed < maxFailedRetries {
			log.Debugf("%s => %v, retrying", s.URL, err)
			time.Sleep(delay)
			if s.Check(client) != nil {
				failed++
			}
		}
		log.Debugf("%s => %s @ %v", s.URL, s.Status.String(), s.Latency)
	}
}
