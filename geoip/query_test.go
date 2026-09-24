package geoip

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"service/config"
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

func TestCountryErrorHidesAddress(t *testing.T) {
	if err := config.Init("docker"); err != nil {
		t.Fatal(err)
	}
	// A GeoIP endpoint that refuses connections: the transport error must
	// not repeat the query URL with the client's address in it.
	srv := httptest.NewServer(http.NotFoundHandler())
	endpoint := srv.URL
	srv.Close()
	t.Setenv("GEOIP_ENDPOINT", endpoint)

	r := httptest.NewRequest("GET", "/v1/x.zip", nil)
	r.Header.Set("X-Forwarded-For", "203.0.113.7")
	_, err := Country(r)
	if err == nil {
		t.Fatal("want an error from a closed endpoint")
	}
	if strings.Contains(err.Error(), "203.0.113") {
		t.Errorf("error leaks the client address: %v", err)
	}
}
