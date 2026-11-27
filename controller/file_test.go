package controller

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"service/config"
	"service/log"
	"service/model"

	"github.com/gorilla/mux"
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

// /v1/<folder>/<path>: the mirror is picked by the folder's probe and the
// redirect goes to <path> in that mirror's folder, with CORS; such reads are
// not recorded (a GET below would otherwise reach Firestore) (#63).
func TestDownloadFolder(t *testing.T) {
	if err := config.Init("docker"); err != nil {
		t.Fatal(err)
	}
	if err := log.Init(); err != nil {
		t.Fatal(err)
	}
	site := &model.Site{Name: "kcwu", Host: "kcwu.example.org", Prefix: "/rudy", RedirectScheme: "https", Weight: 1}
	var asked []string
	c := &FileController{File: "style/version.txt", Folder: "style", sources: func(_ context.Context, file string, corsOnly bool) ([]*model.Source, error) {
		asked = append(asked, file)
		return []*model.Source{{Site: site, SiteName: "kcwu", File: file, URL: site.GetRedirectURL(file)}}, nil
	}}
	r := mux.NewRouter()
	r.HandleFunc("/v1/style/{path:.+}", c.Download)
	serve := func(method, path string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(method, path, nil)
		req.Header.Set("Origin", "https://example.org")
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)
		return rec
	}
	for path, want := range map[string]string{
		"/v1/style/style.json":                             "https://kcwu.example.org/rudy/style/style.json",
		"/v1/style/glyphs/Noto%20Sans%20Regular/0-255.pbf": "https://kcwu.example.org/rudy/style/glyphs/Noto%20Sans%20Regular/0-255.pbf",
	} {
		for _, method := range []string{http.MethodHead, http.MethodGet} {
			rec := serve(method, path)
			if rec.Code != http.StatusFound || rec.Header().Get("Location") != want {
				t.Errorf("%s %s: %d %q, want 302 to %s", method, path, rec.Code, rec.Header().Get("Location"), want)
			}
			if rec.Header().Get("Access-Control-Allow-Origin") != "*" {
				t.Errorf("%s %s: no Access-Control-Allow-Origin", method, path)
			}
		}
	}
	for _, f := range asked {
		if f != "style/version.txt" {
			t.Errorf("sources asked for %q, want the probe", f)
		}
	}
	if rec := serve(http.MethodOptions, "/v1/style/style.json"); rec.Code != http.StatusNoContent {
		t.Errorf("OPTIONS: %d, want 204", rec.Code)
	}
	if rec := serve(http.MethodHead, "/v1/style/a/../../secret"); rec.Code != http.StatusMovedPermanently || rec.Header().Get("Location") != "/v1/secret" {
		t.Errorf("a path climbing out of the folder: %d %q, want mux's 301 to the cleaned path", rec.Code, rec.Header().Get("Location"))
	}
	time.Sleep(100 * time.Millisecond) // a recording goroutine would have hit Firestore by now
}
