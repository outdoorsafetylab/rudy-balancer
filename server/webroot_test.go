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
	for _, page := range []string{"index.html", "app/privacy/index.html", "deployment/index.html", "developers/index.html"} {
		b, err := os.ReadFile(filepath.Join("..", "webroot", page))
		if err != nil {
			t.Fatal(err)
		}
		if !strings.HasPrefix(string(b), "<!DOCTYPE html>") {
			t.Errorf("webroot/%s does not start with <!DOCTYPE html>", page)
		}
	}
}
