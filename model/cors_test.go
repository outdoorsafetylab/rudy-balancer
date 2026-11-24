package model

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// corsMirror answers HEAD and the preflight the way a mirror's config would.
type corsMirror struct {
	headOrigin   string // Access-Control-Allow-Origin on HEAD; "" for none
	headStatus   int
	preStatus    int
	preOrigin    string
	allowHeaders string
	seenOrigin   []string
}

func (m *corsMirror) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	m.seenOrigin = append(m.seenOrigin, r.Method+" "+r.Header.Get("Origin"))
	switch r.Method {
	case http.MethodHead:
		if m.headOrigin != "" {
			w.Header().Set("Access-Control-Allow-Origin", m.headOrigin)
		}
		w.WriteHeader(m.headStatus)
	case http.MethodOptions:
		if m.preOrigin != "" {
			w.Header().Set("Access-Control-Allow-Origin", m.preOrigin)
		}
		if m.allowHeaders != "" {
			w.Header().Set("Access-Control-Allow-Headers", m.allowHeaders)
		}
		w.WriteHeader(m.preStatus)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func TestCheckCORS(t *testing.T) {
	tests := []struct {
		name string
		m    corsMirror
		want bool
	}{
		// kcwu's nginx config from alpha-rudy/taiwan-topo#113.
		{"full", corsMirror{"*", 200, 204, "*", "Range, If-Range, If-None-Match, If-Modified-Since", nil}, true},
		{"wildcard headers", corsMirror{"*", 200, 200, "*", "*", nil}, true},
		{"lower-case list over two header lines", corsMirror{"*", 200, 204, "*", "if-range,range", nil}, true},
		{"no CORS at all", corsMirror{"", 200, 403, "", "", nil}, false},
		// Happyman today: the preflight succeeds but carries no CORS header.
		{"preflight without CORS", corsMirror{"", 200, 200, "", "", nil}, false},
		// After a cross-origin redirect the browser sends Origin: null; a
		// mirror allowing only a named site fails it.
		{"named origin only", corsMirror{"https://cairn.tw", 200, 204, "https://cairn.tw", "Range", nil}, false},
		{"named origin on the download only", corsMirror{"https://cairn.tw", 200, 204, "*", "Range", nil}, false},
		{"preflight refused", corsMirror{"*", 200, 403, "*", "Range", nil}, false},
		{"Range not allowed", corsMirror{"*", 200, 204, "*", "If-None-Match", nil}, false},
		{"file missing", corsMirror{"*", 404, 204, "*", "Range", nil}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := tt.m
			ts := httptest.NewServer(&m)
			defer ts.Close()
			s := &Source{URL: ts.URL + "/f.zip"}
			if got := s.CheckCORS(ts.Client()); got != tt.want {
				t.Errorf("CheckCORS = %v, want %v", got, tt.want)
			}
			for _, seen := range m.seenOrigin {
				if seen != "HEAD null" && seen != "OPTIONS null" {
					t.Errorf("request %q, want Origin: null", seen)
				}
			}
		})
	}
}
