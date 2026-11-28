package server

import (
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// A directory with its own index.html is a page of its own (/app/privacy);
// any other path falls back to the portal page.
func TestWebrootServesDirectoryIndex(t *testing.T) {
	root := t.TempDir()
	write := func(name, body string) {
		t.Helper()
		path := filepath.Join(root, name)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("index.html", "portal")
	write("app/privacy/index.html", "privacy")
	write("app/app.css", "css")
	if err := os.MkdirAll(filepath.Join(root, "app", "empty"), 0o755); err != nil {
		t.Fatal(err)
	}

	h := &webrootHandler{path: root}
	for path, want := range map[string]string{
		"/app":          "portal",
		"/app/":         "portal",
		"/app/privacy":  "privacy",
		"/app/privacy/": "privacy",
		"/app/app.css":  "css",
		"/app/empty":    "portal",
		"/app/missing":  "portal",
	} {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest("GET", path, nil))
		if got := w.Body.String(); got != want {
			t.Errorf("%s: got %q (status %d), want %q", path, got, w.Code, want)
		}
	}
}

// The portal's first line is its doctype: anything before it (a stray
// </html> once) puts the document in quirks mode and moves <head>, the icon
// link included, into <body>, where browsers ignore it.
func TestPortalPagesStartWithDoctype(t *testing.T) {
	for _, page := range []string{"index.html", "app/privacy/index.html", "deployment/index.html", "developers/index.html", "faq/index.html"} {
		b, err := os.ReadFile(filepath.Join("..", "webroot", page))
		if err != nil {
			t.Fatal(err)
		}
		if !strings.HasPrefix(string(b), "<!DOCTYPE html>") {
			t.Errorf("webroot/%s does not start with <!DOCTYPE html>", page)
		}
	}
}

// Webroot responses revalidate every time with a quoted strong ETag; the
// unquoted ETag older responses carried still gets a 304.
func TestWebrootRevalidates(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "index.html"), []byte("portal"), 0o644); err != nil {
		t.Fatal(err)
	}
	h := &webrootHandler{path: root}
	get := func(inm string) *httptest.ResponseRecorder {
		req := httptest.NewRequest("GET", "/app", nil)
		if inm != "" {
			req.Header.Set("If-None-Match", inm)
		}
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)
		return w
	}
	w := get("")
	etag := w.Header().Get("ETag")
	if w.Code != 200 || w.Header().Get("Cache-Control") != "no-cache" || len(etag) < 3 || etag[0] != '"' || etag[len(etag)-1] != '"' {
		t.Fatalf("first GET: %d, Cache-Control %q, ETag %q; want 200, no-cache, a quoted ETag", w.Code, w.Header().Get("Cache-Control"), etag)
	}
	for _, inm := range []string{etag, `W/` + etag, strings.Trim(etag, `"`), `"other", ` + etag} {
		if w := get(inm); w.Code != 304 || w.Header().Get("ETag") != etag || w.Header().Get("Cache-Control") != "no-cache" {
			t.Errorf("If-None-Match %s: %d, ETag %q, Cache-Control %q; want 304 with both", inm, w.Code, w.Header().Get("ETag"), w.Header().Get("Cache-Control"))
		}
	}
	if w := get(`"other"`); w.Code != 200 {
		t.Errorf("a stale ETag: %d, want 200", w.Code)
	}
}

// The deployment diagram carries its own colours, so it draws even when the
// stylesheet is stale or missing (an SVG shape defaults to black fill).
func TestDeploymentDiagramSelfStyled(t *testing.T) {
	b, err := os.ReadFile(filepath.Join("..", "webroot", "deployment", "index.html"))
	if err != nil {
		t.Fatal(err)
	}
	s := string(b)
	svg := s[strings.Index(s, "<svg"):strings.Index(s, "</svg>")]
	for _, tag := range []string{"<rect ", "<text "} {
		for _, el := range strings.Split(svg, tag)[1:] {
			el = el[:strings.Index(el, ">")]
			if !strings.Contains(el, `fill="`) {
				t.Errorf("%s%s>: no fill attribute", tag, el)
			}
		}
	}
	if !strings.Contains(svg, `<g class="flow" fill="none" stroke=`) {
		t.Error("the arrows' group has no stroke of its own")
	}
	if !strings.Contains(svg, `<path d="M0,0 L10,5 L0,10 z" fill="#`) {
		t.Error("the arrowhead has no colour of its own (currentColor is black without the stylesheet)")
	}
}

// Pages ask for the stylesheet under a versioned URL: responses before
// Cache-Control was set left browsers holding the old app.css for weeks, and a
// new URL is the only way past that cached copy.
func TestPagesUseVersionedStylesheet(t *testing.T) {
	for _, page := range []string{"index.html", "app/privacy/index.html", "deployment/index.html", "developers/index.html", "faq/index.html"} {
		b, err := os.ReadFile(filepath.Join("..", "webroot", page))
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(b), `href="/app/app.css"`) || !strings.Contains(string(b), `href="/app/app.css?v=`) {
			t.Errorf("webroot/%s: stylesheet not under a versioned URL", page)
		}
	}
}

// Every map-kind button on the developers page has a section to show, under
// the half it belongs to, with the id the button's link and the page script use.
func TestDevelopersKindPicks(t *testing.T) {
	b, err := os.ReadFile(filepath.Join("..", "webroot", "developers", "index.html"))
	if err != nil {
		t.Fatal(err)
	}
	s := string(b)
	want := map[string][]string{
		"web": {"web-tiles", "web-pmtiles"},
		"app": {"app-tiles", "app-mapsforge", "app-pmtiles"},
	}
	for half, kinds := range want {
		start := strings.Index(s, `<section data-for-pane="`+half+`" id="`+half+`">`)
		if start < 0 {
			t.Fatalf("no %s pane", half)
		}
		end := strings.Index(s[start:], "\n    </section>")
		pane := s[start : start+end]
		for _, k := range kinds {
			for _, frag := range []string{
				`href="#` + k + `" data-pick="` + k + `"`,
				`<section data-kind="` + k + `">` + "\n" + `        <h2 class="h5 mt-4" id="` + k + `">`,
			} {
				if !strings.Contains(pane, frag) {
					t.Errorf("%s pane: missing %s", half, frag)
				}
			}
		}
		if n := strings.Count(pane, "data-pick="); n != len(kinds) {
			t.Errorf("%s pane: %d kind buttons, want %d", half, n, len(kinds))
		}
		// online tiles first: the easiest to integrate
		last := -1
		for _, k := range kinds {
			i := strings.Index(pane, `data-pick="`+k+`"`)
			if i < last {
				t.Errorf("%s pane: %s button out of order, want %v", half, k, kinds)
			}
			last = i
		}
	}
}

// The footer reads "© years 魯地圖 · Outdoor Safety Lab · ..." on every page:
// Rudy Map is the first name after the copyright years.
func TestFooterOrder(t *testing.T) {
	for _, page := range []string{"index.html", "app/privacy/index.html", "deployment/index.html", "developers/index.html", "faq/index.html"} {
		b, err := os.ReadFile(filepath.Join("..", "webroot", page))
		if err != nil {
			t.Fatal(err)
		}
		s := string(b)
		footer := s[strings.Index(s, "<footer"):]
		footer = footer[:strings.Index(footer, "</footer>")]
		c, r, o := strings.Index(footer, "&copy;"), strings.Index(footer, ">魯地圖</a>"), strings.Index(footer, ">Outdoor Safety Lab</a>")
		if c < 0 || r < 0 || o < 0 || !(c < r && r < o) {
			t.Errorf("webroot/%s footer: want © then 魯地圖 then Outdoor Safety Lab (positions %d, %d, %d)", page, c, r, o)
		}
	}
}
