package server

import (
	"net/http/httptest"
	"os"
	"testing"

	"service/config"

	"github.com/gorilla/mux"
)

// Builds the real router from config/mirrors.yaml and checks which /via/
// paths resolve to a route (#19). Only route matching runs, so no handler
// touches Firestore.
func TestViaRoutes(t *testing.T) {
	// docker.yaml points at /mirrors.yaml inside the image.
	os.Setenv("MIRRORS_FILE", "../config/mirrors.yaml")
	defer os.Unsetenv("MIRRORS_FILE")
	if err := config.Init("docker"); err != nil {
		t.Fatal(err)
	}
	r, err := newRouter()
	if err != nil {
		t.Fatal(err)
	}

	for path, routed := range map[string]bool{
		// Links rudymap.tw/app generates.
		"/v1/via/oruxmaps/MOI_OSM_Taiwan_TOPO_Rudy.map.zip": true,
		"/v1/via/oruxmaps/hgtmix.zip":                       true,
		"/v1/via/gts/MOI_OSM_Taiwan_TOPO_Rudy.zip":          true,
		"/v1/via/wadi/MOI_OSM_Taiwan_TOPO_Lite.zip":         true,
		"/v1/via/carto/carto_all.cpkg":                      true,
		// The plain route is unchanged.
		"/v1/MOI_OSM_Taiwan_TOPO_Rudy.zip": true,
		// OruxMaps has no bundle link, so this pair is never generated.
		"/v1/via/oruxmaps/MOI_OSM_Taiwan_TOPO_Rudy.zip": false,
		// Unknown app, unknown file.
		"/v1/via/bogus/MOI_OSM_Taiwan_TOPO_Rudy.map.zip": false,
		"/v1/via/oruxmaps/nope.zip":                      false,
		// Locus's links are hard-coded locus-actions:// URLs, never via paths.
		"/v1/via/locus/MOI_OSM_Taiwan_TOPO_Rudy_locus.zip": false,
	} {
		var m mux.RouteMatch
		r.Match(httptest.NewRequest("GET", path, nil), &m)
		// With a NotFoundHandler set, Match reports a miss as MatchErr.
		if got := m.MatchErr == nil && m.Route != nil; got != routed {
			t.Errorf("%s: routed=%v, want %v", path, got, routed)
		}
	}
}
