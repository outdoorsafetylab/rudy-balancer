package server

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

type webrootHandler struct {
	path string
}

func (h *webrootHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := fmt.Sprintf("%s%s", h.path, r.URL.Path)
	st, err := os.Stat(path)
	if err == nil && st.IsDir() {
		// A page of its own, e.g. /app/privacy.
		path = filepath.Join(path, "index.html")
		st, err = os.Stat(path)
	}
	if err != nil || st.IsDir() {
		path = fmt.Sprintf("%s/index.html", h.path)
	}
	sum, err := sha1sum(path)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	// Revalidate every time: these pages change in place under the same URL
	// (a stylesheet left cached for days drew the deployment diagram in black),
	// and with no Cache-Control a browser guesses a lifetime from the file's age.
	// A strong ETag is quoted; the unquoted form older responses carried is
	// still accepted, so those caches revalidate to a 304 too.
	etag := `"` + sum + `"`
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("ETag", etag)
	for _, m := range strings.Split(r.Header.Get("If-None-Match"), ",") {
		m = strings.TrimPrefix(strings.TrimSpace(m), "W/")
		if m == etag || m == sum || m == "*" {
			w.WriteHeader(http.StatusNotModified)
			return
		}
	}
	http.ServeFile(w, r, path)
}

var sha1sumCache = map[string]string{}

func sha1sum(path string) (string, error) {
	sum := sha1sumCache[path]
	if sum != "" {
		return sum, nil
	}
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha1.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	sum = hex.EncodeToString(h.Sum(nil))
	sha1sumCache[path] = sum
	return sum, nil
}
