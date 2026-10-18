package healthcheck

import (
	"fmt"
	"net/http"
	"service/log"
	"service/statuspage"
	"time"

	"github.com/spf13/viper"
)

func CheckPortalSites(cfg *viper.Viper) error {
	// Get statuspage configuration
	pageID := cfg.GetString("statuspage.page")
	apiKey := cfg.GetString("statuspage.key")

	if pageID == "" || apiKey == "" {
		log.Debugf("StatusPage not configured, skipping portals health check")
		return nil // Not an error, just skip
	}

	// Parse portals configuration
	var portalConfig portalConfig
	err := cfg.UnmarshalKey("statuspage.portals", &portalConfig)
	if err != nil {
		return fmt.Errorf("failed to parse portals config: %w", err)
	}

	client := &http.Client{
		Timeout: time.Duration(cfg.GetInt("healthcheck.timeout_sec")) * time.Second,
	}

	observed := make([]statuspage.Observation, 0, len(portalConfig.Sites))
	for _, site := range portalConfig.Sites {
		log.Debugf("Checking portal site: %s", site.URL)

		// Check each asset URL
		goodResources := make([]string, 0)
		badResources := make(map[string]error)

		for _, asset := range site.Assets {
			url := fmt.Sprintf("%s%s", site.URL, asset)
			log.Debugf("Checking asset: %s", url)

			req, err := http.NewRequest("GET", url, nil)
			if err != nil {
				log.Errorf("Failed to create request for %s: %s", url, err.Error())
				badResources[url] = err
				continue
			}

			resp, err := client.Do(req)
			if err != nil {
				log.Errorf("Failed to check %s: %s", url, err.Error())
				badResources[url] = err
				continue
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				log.Errorf("Bad status for %s: %d", url, resp.StatusCode)
				badResources[url] = fmt.Errorf("%s", resp.Status)
			} else {
				log.Debugf("Asset %s is operational", url)
				goodResources = append(goodResources, url)
			}
		}

		// Determine status based on percentage of operational assets
		var newStatus string
		total := len(goodResources) + len(badResources)
		if total > 0 {
			percentage := 100.0 * len(goodResources) / total
			log.Debugf("Portal %s: %d/%d assets operational (%d%%)", site.Name, len(goodResources), total, percentage)
			if percentage >= 100 {
				newStatus = "operational"
			} else if percentage <= 0 {
				newStatus = "major_outage"
			} else {
				newStatus = "partial_outage"
			}
		} else {
			// No assets to check, default to operational
			newStatus = "operational"
		}

		var incidentMessage string
		if newStatus == "partial_outage" {
			incidentMessage = fmt.Sprintf("%s is experiencing partial outage. %d/%d assets are operational.", site.Name, len(goodResources), total)
		} else {
			incidentMessage = fmt.Sprintf("%s is not operational.", site.Name)
		}
		observed = append(observed, statuspage.Observation{
			Component: site.Name,
			Status:    newStatus,
			Incident:  incidentMessage,
			Bad:       badResources,
		})
	}

	statusClient := &statuspage.Client{Client: http.DefaultClient, APIKey: apiKey}
	return statusClient.SyncPage(pageID, portalConfig.Group, observed)
}

type portalSite struct {
	Name   string   `yaml:"name"`
	URL    string   `yaml:"url"`
	Assets []string `yaml:"assets"`
}

type portalConfig struct {
	Group string       `yaml:"group"`
	Sites []portalSite `yaml:"sites"`
}
