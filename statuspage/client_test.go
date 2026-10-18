package statuspage

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"service/config"
	"service/log"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestCreateComponentShowsUptime(t *testing.T) {
	if err := config.Init("docker"); err != nil {
		t.Fatal(err)
	}
	if err := log.Init(); err != nil {
		t.Fatal(err)
	}
	var sent struct {
		Component map[string]any `json:"component"`
	}
	c := &Client{APIKey: "k", Client: &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if r.Method != "POST" || r.URL.Path != "/v1/pages/p/components" {
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&sent); err != nil {
			t.Fatal(err)
		}
		return &http.Response{StatusCode: 201, Body: io.NopCloser(strings.NewReader(`{"id":"c"}`))}, nil
	})}}

	if _, err := c.CreateComponent("p", "g", "PeaKit"); err != nil {
		t.Fatal(err)
	}
	if sent.Component["showcase"] != true {
		t.Errorf("showcase = %v, want true", sent.Component["showcase"])
	}
	if sent.Component["name"] != "PeaKit" || sent.Component["group_id"] != "g" {
		t.Errorf("component = %v", sent.Component)
	}
}
