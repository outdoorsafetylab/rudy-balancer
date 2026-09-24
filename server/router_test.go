package server

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"service/config"
	"service/log"
	"service/mirror"

	"github.com/gorilla/mux"
)

// Builds the real router from config/mirrors.yaml and checks the /via/ paths
// (#19). Known links are checked by route template only, so no handler
// touches Firestore; unknown ones are served, since http.NotFound doesn't.
func TestViaRoutes(t *testing.T) {
	// docker.yaml points at /mirrors.yaml inside the image.
	os.Setenv("MIRRORS_FILE", "../config/mirrors.yaml")
	defer os.Unsetenv("MIRRORS_FILE")
	if err := config.Init("docker"); err != nil {
		t.Fatal(err)
	}
	// middleware.Dump logs every request it sees.
	if err := log.Init(); err != nil {
		t.Fatal(err)
	}
	r, err := newRouter()
	if err != nil {
		t.Fatal(err)
	}
	m, err := mirror.Get()
	if err != nil {
		t.Fatal(err)
	}

	// Every link the portal generates has its own route. Checking all of them,
	// not a sample, catches a single file left unregistered.
	links := m.ViaLinks()
	if len(links) < 20 {
		t.Fatalf("only %d via links from mirrors.yaml; expected the oruxmaps/gts/wadi/carto set", len(links))
	}
	for _, link := range links {
		want := "/v1" + link.Path()
		var match mux.RouteMatch
		if !r.Match(httptest.NewRequest("GET", want, nil), &match) || match.MatchErr != nil || match.Route == nil {
			t.Errorf("%s: no route", want)
			continue
		}
		if got, _ := match.Route.GetPathTemplate(); got != want {
			t.Errorf("%s: matched %s, want its own route", want, got)
		}
	}

	// The three files alpha-rudy/taiwan-topo's Taiwan *-cedric.xml download
	// must be routed before those XML files switch to via paths.
	for _, file := range []string{"MOI_OSM_Taiwan_TOPO_Rudy_locus.zip", "MOI_OSM_Taiwan_TOPO_Rudy_locus_style.zip", "hgtmix.zip"} {
		want := "/v1/via/locus/" + file
		var match mux.RouteMatch
		r.Match(httptest.NewRequest("GET", want, nil), &match)
		if match.Route == nil {
			t.Errorf("%s: no route", want)
			continue
		}
		if got, _ := match.Route.GetPathTemplate(); got != want {
			t.Errorf("%s: matched %q, want its own route", want, got)
		}
	}

	// The plain route is untouched.
	var plain mux.RouteMatch
	r.Match(httptest.NewRequest("GET", "/v1/MOI_OSM_Taiwan_TOPO_Rudy.zip", nil), &plain)
	if plain.Route == nil {
		t.Error("/v1/MOI_OSM_Taiwan_TOPO_Rudy.zip: no route")
	} else if got, _ := plain.Route.GetPathTemplate(); got != "/v1/MOI_OSM_Taiwan_TOPO_Rudy.zip" {
		t.Errorf("/v1/MOI_OSM_Taiwan_TOPO_Rudy.zip matched %s", got)
	}

	// Pairs the portal never generates get a 404, not the reverse proxy's
	// redirect to a mirror.
	for _, path := range []string{
		"/v1/via/oruxmaps/MOI_OSM_Taiwan_TOPO_Rudy.zip",  // OruxMaps has no bundle link
		"/v1/via/bogus/MOI_OSM_Taiwan_TOPO_Rudy.map.zip", // unknown app
		"/v1/via/oruxmaps/nope.zip",                      // unknown file
		"/v1/via/",
		"/v1/via",
	} {
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, httptest.NewRequest("GET", path, nil))
		if rec.Code != http.StatusNotFound {
			t.Errorf("%s: status %d (Location %q), want 404", path, rec.Code, rec.Header().Get("Location"))
		}
	}
}
