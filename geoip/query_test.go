package geoip

import (
	"net/http/httptest"
	"strings"
	"testing"
)

func TestIPAddress(t *testing.T) {
	tests := []struct {
		name, fwd, remote, want string
	}{
		// Cloud Run appends the real peer; the client can prepend anything.
		{"spoofed first entry", "198.51.100.77,203.0.113.7", "", "203.0.113.7"},
		{"spaces", "198.51.100.77, 203.0.113.7 ", "", "203.0.113.7"},
		{"single", "203.0.113.7", "", "203.0.113.7"},
		{"no header", "", "203.0.113.9:4242", "203.0.113.9"},
	}
	for _, tt := range tests {
		r := httptest.NewRequest("GET", "/v1/x.zip", nil)
		if tt.fwd != "" {
			r.Header.Set("X-Forwarded-For", tt.fwd)
		}
		if tt.remote != "" {
			r.RemoteAddr = tt.remote
		}
		ip, err := IPAddress(r)
		if err != nil || ip.String() != tt.want {
			t.Errorf("%s: got %v, %v; want %s", tt.name, ip, err, tt.want)
		}
	}

	r := httptest.NewRequest("GET", "/v1/x.zip", nil)
	r.Header.Set("X-Forwarded-For", "203.0.113.7, not-an-ip-203.0.113.8")
	if _, err := IPAddress(r); err == nil || strings.Contains(err.Error(), "203.0.113") {
		t.Errorf("want an error without the address, got %v", err)
	}
}
