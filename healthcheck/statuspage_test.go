package healthcheck

import (
	"strings"
	"testing"

	"github.com/spf13/viper"
)

func TestRequireStatusPage(t *testing.T) {
	for _, tc := range []struct {
		name, page, key string
		missing         []string
	}{
		{name: "configured", page: "p", key: "k"},
		{name: "no key", page: "p", missing: []string{"STATUSPAGE_KEY"}},
		{name: "no page", key: "k", missing: []string{"statuspage.page"}},
		{name: "neither", missing: []string{"statuspage.page", "STATUSPAGE_KEY"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cfg := viper.New()
			cfg.Set("statuspage.page", tc.page)
			cfg.Set("statuspage.key", tc.key)
			err := RequireStatusPage(cfg)
			if len(tc.missing) == 0 {
				if err != nil {
					t.Fatalf("want nil, got %v", err)
				}
				return
			}
			if err == nil {
				t.Fatal("want an error, got nil")
			}
			for _, m := range tc.missing {
				if !strings.Contains(err.Error(), m) {
					t.Errorf("error %q does not name %q", err, m)
				}
			}
		})
	}
}
