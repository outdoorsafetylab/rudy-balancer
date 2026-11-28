package controller

import (
	"testing"

	"service/model"
)

// /v1/sites names mirrors only: an app falling back to it must never be
// sent to the upstream copy.
func TestListSitesLeavesOutUpstream(t *testing.T) {
	got := listSites([]*model.Site{
		{Name: "kcwu", Host: "moi.kcwu.csie.org"},
		{Name: "RudyR2", Host: "data.rudymap.tw", Upstream: true},
		{Name: "OSL", Host: "d3r5lsn28erp7o.cloudfront.net"},
	})
	if len(got) != 2 || got[0].Name != "kcwu" || got[1].Name != "OSL" {
		names := []string{}
		for _, s := range got {
			names = append(names, s.Name)
		}
		t.Errorf("listed %v, want [kcwu OSL]", names)
	}
}
