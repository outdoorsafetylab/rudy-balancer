package healthcheck

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"service/config"
	"service/log"
	"service/model"
)

// server answers each path with the statuses in order, then with the last
// one, and counts the requests per path.
type server struct {
	mu         sync.Mutex
	plan       map[string][]int
	counts     map[string]int
	cors       map[string]bool
	corsChecks map[string]int
}

func (s *server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()
	// The CORS check of a GOOD file (Origin set) is not a health check
	// request: it is answered without CORS headers and not counted.
	if r.Header.Get("Origin") != "" {
		if s.cors[r.URL.Path] {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Headers", "Range")
		}
		s.corsChecks[r.URL.Path]++
		return
	}
	n := s.counts[r.URL.Path]
	s.counts[r.URL.Path] = n + 1
	plan := s.plan[r.URL.Path]
	if n >= len(plan) {
		n = len(plan) - 1
	}
	w.WriteHeader(plan[n])
}

func site(base string, files ...string) *model.Site {
	site := &model.Site{Name: "test"}
	for _, f := range files {
		site.Sources = append(site.Sources, &model.Source{Site: site, File: f, URL: base + "/" + f})
	}
	return site
}

func TestCheckSiteRetriesAFailedFileOnce(t *testing.T) {
	if err := config.Init("docker"); err != nil {
		t.Fatal(err)
	}
	if err := log.Init(); err != nil {
		t.Fatal(err)
	}
	srv := &server{counts: map[string]int{}, corsChecks: map[string]int{}, plan: map[string][]int{
		"/ok":    {200},
		"/blip":  {503, 200},
		"/down1": {503},
		"/down2": {503},
		"/down3": {503},
		"/after": {503, 200},
	}}
	ts := httptest.NewServer(srv)
	defer ts.Close()

	s := site(ts.URL, "ok", "blip", "down1", "down2", "down3", "after")
	checkSite(ts.Client(), s, 0)

	want := map[string]model.Status{
		"ok": model.GOOD, "blip": model.GOOD,
		"down1": model.BAD, "down2": model.BAD, "down3": model.BAD,
		"after": model.BAD,
	}
	for _, src := range s.Sources {
		if src.Status != want[src.File] {
			t.Errorf("%s: status %v, want %v", src.File, src.Status, want[src.File])
		}
	}
	// Three retries failed, so "after" is checked once and not retried.
	wantCounts := map[string]int{"/ok": 1, "/blip": 2, "/down1": 2, "/down2": 2, "/down3": 2, "/after": 1}
	for path, n := range wantCounts {
		if srv.counts[path] != n {
			t.Errorf("%s: %d requests, want %d", path, srv.counts[path], n)
		}
	}
}

func TestCheckSiteRecordsCORSOfGoodFilesOnly(t *testing.T) {
	if err := config.Init("docker"); err != nil {
		t.Fatal(err)
	}
	if err := log.Init(); err != nil {
		t.Fatal(err)
	}
	srv := &server{counts: map[string]int{}, corsChecks: map[string]int{},
		plan: map[string][]int{"/web": {200}, "/plain": {200}, "/down": {503}},
		cors: map[string]bool{"/web": true, "/down": true},
	}
	ts := httptest.NewServer(srv)
	defer ts.Close()

	s := site(ts.URL, "web", "plain", "down")
	s.Sources[2].CORS = true // left over from an earlier run
	checkSite(ts.Client(), s, 0)

	want := map[string]bool{"web": true, "plain": false, "down": false}
	for _, src := range s.Sources {
		if src.CORS != want[src.File] {
			t.Errorf("%s: CORS %v, want %v", src.File, src.CORS, want[src.File])
		}
	}
	// A file that is down is not checked for CORS at all.
	if n := srv.corsChecks["/down"]; n != 0 {
		t.Errorf("/down: %d CORS checks, want 0", n)
	}
	if srv.corsChecks["/web"] != 2 || srv.corsChecks["/plain"] != 1 {
		t.Errorf("CORS checks %v, want web: 2 (HEAD, preflight), plain: 1 (HEAD fails)", srv.corsChecks)
	}
}
