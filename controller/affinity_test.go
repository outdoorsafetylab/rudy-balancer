package controller

import (
	"context"
	"fmt"
	"math"
	"net/http"
	"net/http/httptest"
	"testing"

	"service/config"
	"service/log"
	"service/model"
)

// weighted lists the sources as GetSources does: each repeated by its weight.
func weighted(file string, weights map[string]int, order ...string) []*model.Source {
	var out []*model.Source
	for _, name := range order {
		src := &model.Source{SiteName: name, File: file, URL: "https://" + name + "/" + file}
		for i := 0; i < weights[name]; i++ {
			out = append(out, src)
		}
	}
	return out
}

func clientIP(i int) string { return fmt.Sprintf("10.%d.%d.%d", i>>16&255, i>>8&255, i&255) }

func TestStickySourceSameSiteForEveryFile(t *testing.T) {
	w := map[string]int{"kcwu": 4, "OSL": 2, "PeaKit": 1, "Happyman": 1}
	for i := 0; i < 500; i++ {
		ip := clientIP(i)
		first := stickySource(weighted("a.pmtiles", w, "kcwu", "OSL", "PeaKit", "Happyman"), ip).SiteName
		// another file, the list in another order: still the same site
		for _, file := range []string{"style.json", "glyphs/Noto Sans Regular/0-255.pbf"} {
			got := stickySource(weighted(file, w, "Happyman", "PeaKit", "OSL", "kcwu"), ip)
			if got.SiteName != first || got.File != file {
				t.Fatalf("%s: %s on %s, want %s", ip, file, got.SiteName, first)
			}
		}
	}
}

func TestStickySourceSpreadsByWeight(t *testing.T) {
	w := map[string]int{"kcwu": 4, "OSL": 2, "PeaKit": 1, "Happyman": 1}
	sources := weighted("a.pmtiles", w, "kcwu", "OSL", "PeaKit", "Happyman")
	const n = 40000
	got := map[string]int{}
	for i := 0; i < n; i++ {
		got[stickySource(sources, clientIP(i)).SiteName]++
	}
	for name, weight := range w {
		share, want := float64(got[name])/n, float64(weight)/8
		if math.Abs(share-want) > 0.015 {
			t.Errorf("%s: %.3f of clients, want %.3f", name, share, want)
		}
	}
}

func TestStickySourceDroppedSiteMovesOnlyItsClients(t *testing.T) {
	w := map[string]int{"kcwu": 4, "OSL": 2, "PeaKit": 1}
	all := weighted("a.pmtiles", w, "kcwu", "OSL", "PeaKit")
	lagging := weighted("a.pmtiles", w, "kcwu", "PeaKit") // OSL excluded for this file
	moved := 0
	for i := 0; i < 5000; i++ {
		ip := clientIP(i)
		before, after := stickySource(all, ip).SiteName, stickySource(lagging, ip).SiteName
		if before != "OSL" && after != before {
			t.Fatalf("%s moved %s -> %s though %s is still there", ip, before, after, before)
		}
		if before == "OSL" {
			moved++
		}
	}
	if moved == 0 {
		t.Fatal("no client was on OSL")
	}
}

func TestDownloadStickyForWebPagesOnly(t *testing.T) {
	if err := config.Init("docker"); err != nil {
		t.Fatal(err)
	}
	if err := log.Init(); err != nil {
		t.Fatal(err)
	}
	w := map[string]int{"kcwu": 1, "OSL": 1}
	c := &FileController{sources: func(_ context.Context, file string, _ bool) ([]*model.Source, error) {
		return weighted(file, w, "kcwu", "OSL"), nil
	}}
	serve := func(file, origin, ip string) string {
		c.File = file
		req := httptest.NewRequest(http.MethodHead, "/v1/"+file, nil)
		if origin != "" {
			req.Header.Set("Origin", origin)
		}
		req.Header.Set("X-Forwarded-For", ip)
		rec := httptest.NewRecorder()
		c.Download(rec, req)
		return rec.Header().Get("Location")
	}
	// a web page: every request from one client to the same mirror
	for i := 0; i < 50; i++ {
		ip := clientIP(i)
		want := stickySource(weighted("a.pmtiles", w, "kcwu", "OSL"), ip).SiteName
		for _, file := range []string{"a.pmtiles", "style.json"} {
			for j := 0; j < 5; j++ {
				if got := serve(file, "https://example.org", ip); got != "https://"+want+"/"+file {
					t.Fatalf("%s %s: %s, want on %s", ip, file, got, want)
				}
			}
		}
	}
	// an app or a plain download: still random, so one client sees both mirrors
	seen := map[string]bool{}
	for i := 0; i < 200; i++ {
		seen[serve("a.pmtiles", "", clientIP(1))] = true
	}
	if len(seen) != 2 {
		t.Errorf("without Origin: %d mirrors over 200 requests, want both", len(seen))
	}
}
