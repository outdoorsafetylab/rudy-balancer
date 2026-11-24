package controller

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"service/config"
	"service/log"
	"service/model"
)

func TestDownloadServesWebPages(t *testing.T) {
	if err := config.Init("docker"); err != nil {
		t.Fatal(err)
	}
	if err := log.Init(); err != nil {
		t.Fatal(err)
	}
	site := &model.Site{Name: "kcwu", Weight: 1}
	src := &model.Source{Site: site, SiteName: "kcwu", File: "f.zip", URL: "https://kcwu/f.zip"}
	var gotCORSOnly []bool
	c := &FileController{File: "f.zip", sources: func(_ context.Context, file string, corsOnly bool) ([]*model.Source, error) {
		gotCORSOnly = append(gotCORSOnly, corsOnly)
		if file != "f.zip" {
			t.Errorf("file %q", file)
		}
		return []*model.Source{src}, nil
	}}
	serve := func(method string, origin string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(method, "/v1/f.zip", nil)
		if origin != "" {
			req.Header.Set("Origin", origin)
		}
		rec := httptest.NewRecorder()
		c.Download(rec, req)
		return rec
	}

	// Preflight: answered here, without looking for a mirror.
	rec := serve(http.MethodOptions, "https://cairn.tw")
	if rec.Code != http.StatusNoContent {
		t.Errorf("OPTIONS: status %d, want 204", rec.Code)
	}
	for k, want := range map[string]string{
		"Access-Control-Allow-Origin":  "*",
		"Access-Control-Allow-Methods": "GET, HEAD, OPTIONS",
		"Access-Control-Allow-Headers": corsAllowHeaders,
	} {
		if got := rec.Header().Get(k); got != want {
			t.Errorf("OPTIONS: %s %q, want %q", k, got, want)
		}
	}
	if len(gotCORSOnly) != 0 {
		t.Errorf("OPTIONS looked up sources")
	}

	// HEAD, not GET: GET also records the redirect in Firestore.
	rec = serve(http.MethodHead, "https://cairn.tw")
	if rec.Code != http.StatusFound || rec.Header().Get("Location") != src.URL {
		t.Errorf("HEAD with Origin: %d %q, want 302 to %s", rec.Code, rec.Header().Get("Location"), src.URL)
	}
	if rec.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Errorf("HEAD with Origin: redirect without Access-Control-Allow-Origin: *")
	}
	rec = serve(http.MethodHead, "")
	if rec.Code != http.StatusFound {
		t.Errorf("HEAD without Origin: %d, want 302", rec.Code)
	}
	if want := []bool{true, false}; len(gotCORSOnly) != 2 || gotCORSOnly[0] != want[0] || gotCORSOnly[1] != want[1] {
		t.Errorf("corsOnly %v, want %v (only a request with Origin is limited)", gotCORSOnly, want)
	}

	// No mirror serves web pages: a 502 the page can read.
	c.sources = func(context.Context, string, bool) ([]*model.Source, error) { return nil, nil }
	rec = serve(http.MethodGet, "https://cairn.tw")
	if rec.Code != http.StatusBadGateway || rec.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Errorf("GET with Origin, no source: %d, Access-Control-Allow-Origin %q; want 502 with *", rec.Code, rec.Header().Get("Access-Control-Allow-Origin"))
	}
}
