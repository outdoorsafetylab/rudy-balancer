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
	mu     sync.Mutex
	plan   map[string][]int
	counts map[string]int
}

func (s *server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()
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
	srv := &server{counts: map[string]int{}, plan: map[string][]int{
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
