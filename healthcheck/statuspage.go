package healthcheck

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

// RequireStatusPage fails when the StatusPage page or key is missing. Publishing
// to StatusPage is what the scheduled healthcheck is for, so skipping it quietly
// would report success while the page stays frozen (outdoorsafetylab#50).
func RequireStatusPage(cfg *viper.Viper) error {
	var missing []string
	if cfg.GetString("statuspage.page") == "" {
		missing = append(missing, "statuspage.page")
	}
	if cfg.GetString("statuspage.key") == "" {
		missing = append(missing, "statuspage.key (env STATUSPAGE_KEY)")
	}
	if len(missing) > 0 {
		return fmt.Errorf("StatusPage not configured: missing %s", strings.Join(missing, ", "))
	}
	return nil
}
