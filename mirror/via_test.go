package mirror

import (
	"reflect"
	"testing"

	"service/model"
)

func TestViaLinks(t *testing.T) {
	m := &Mirror{Apps: []*model.App{
		{ID: "oruxmaps", Variants: []*model.Variant{
			{Artifacts: []*model.Artifact{
				{File: "a.map.zip", Scheme: "orux-map:"},
				{File: "hgtmix.zip"},
			}},
			// The same file in a second variant must not register twice.
			{Artifacts: []*model.Artifact{{File: "a.map.zip", Scheme: "orux-map:"}}},
		}},
		{ID: "locus", Variants: []*model.Variant{
			// Hard-coded URL: the page never links to a via path for it.
			{Artifacts: []*model.Artifact{{File: "a_locus.zip", URL: "locus-actions://https/rudymap.tw/x.xml"}}},
		}},
		{ID: "hikinglogger"}, // no variants, no links
	}}

	want := []ViaLink{
		{App: "oruxmaps", File: "a.map.zip"},
		{App: "oruxmaps", File: "hgtmix.zip"},
	}
	if got := m.ViaLinks(); !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestViaLinkPath(t *testing.T) {
	if got := (ViaLink{App: "oruxmaps", File: "MOI_OSM_Taiwan_TOPO_Rudy.map.zip"}).Path(); got != "/via/oruxmaps/MOI_OSM_Taiwan_TOPO_Rudy.map.zip" {
		t.Errorf("got %s", got)
	}
}
