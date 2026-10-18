package statuspage

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"service/config"
	"service/log"
)

// fakePage answers the StatusPage calls SyncPage makes from a fixed starting
// state, records each call as "METHOD path [status]", and fails the calls in
// fail.
type fakePage struct {
	t          *testing.T
	components string
	incidents  string
	fail       map[string]bool
	calls      []string
}

func (f *fakePage) client() *Client {
	return &Client{APIKey: "k", Client: &http.Client{Transport: roundTripFunc(f.roundTrip)}}
}

func (f *fakePage) roundTrip(r *http.Request) (*http.Response, error) {
	call := r.Method + " " + r.URL.Path
	if r.Body != nil {
		var body map[string]map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			f.t.Fatal(err)
		}
		for _, v := range body {
			if s, ok := v["status"]; ok {
				call += fmt.Sprintf(" %v", s)
			}
		}
	}
	f.calls = append(f.calls, call)
	reply := func(code int, body string) (*http.Response, error) {
		return &http.Response{StatusCode: code, Body: io.NopCloser(strings.NewReader(body))}, nil
	}
	if f.fail[r.Method+" "+r.URL.Path] {
		return reply(500, `{}`)
	}
	switch {
	case r.Method == "GET" && r.URL.Path == "/v1/pages/p/components":
		return reply(200, f.components)
	case r.Method == "GET" && r.URL.Path == "/v1/pages/p/incidents/unresolved":
		return reply(200, f.incidents)
	case r.Method == "POST" && r.URL.Path == "/v1/pages/p/components":
		return reply(201, `{"id":"new","page_id":"p","status":"operational"}`)
	}
	return reply(200, `{}`)
}

func setupLog(t *testing.T) {
	t.Helper()
	if err := config.Init("docker"); err != nil {
		t.Fatal(err)
	}
	if err := log.Init(); err != nil {
		t.Fatal(err)
	}
}

func component(status string) string {
	return fmt.Sprintf(`[{"id":"c","page_id":"p","name":"site","status":%q}]`, status)
}

const (
	noIncident     = `[]`
	openIncident   = `[{"id":"i","page_id":"p","components":[{"id":"c"}]}]`
	sharedIncident = `[{"id":"i","page_id":"p","components":[{"id":"c"},{"id":"other"}]}]`
)

func observe(status string) []Observation {
	return []Observation{{Component: "site", Status: status, Incident: "site is not operational.",
		Bad: map[string]error{"https://site/": errors.New("timeout")}}}
}

func TestSyncPage(t *testing.T) {
	setupLog(t)
	const (
		list      = "GET /v1/pages/p/components"
		unresolve = "GET /v1/pages/p/incidents/unresolved"
		incident  = "POST /v1/pages/p/incidents investigating"
		resolve   = "PATCH /v1/pages/p/incidents/i resolved"
	)
	status := func(s string) string { return "PATCH /v1/pages/p/components/c " + s }
	for _, tc := range []struct {
		name      string
		current   string
		incidents string
		observed  string
		want      []string
	}{
		{"first failed run is shown but not announced", "operational", noIncident, "major_outage",
			[]string{list, unresolve, status("degraded_performance")}},
		{"second failed run opens the incident", "degraded_performance", noIncident, "major_outage",
			[]string{list, unresolve, status("major_outage"), incident}},
		{"partial outage also waits a run", "operational", noIncident, "partial_outage",
			[]string{list, unresolve, status("degraded_performance")}},
		{"an outage without its incident gets one", "major_outage", noIncident, "major_outage",
			[]string{list, unresolve, incident}},
		{"an outage with its incident is left alone", "major_outage", openIncident, "major_outage",
			[]string{list, unresolve}},
		{"recovery resolves the incident", "major_outage", openIncident, "operational",
			[]string{list, unresolve, status("operational"), resolve}},
		{"an incident left open while operational is resolved", "operational", openIncident, "operational",
			[]string{list, unresolve, resolve}},
		{"recovery after one failed run opens nothing", "degraded_performance", noIncident, "operational",
			[]string{list, unresolve, status("operational")}},
		{"a healthy component is not written", "operational", noIncident, "operational",
			[]string{list, unresolve}},
		{"a shared incident is not resolved by one component", "major_outage", sharedIncident, "operational",
			[]string{list, unresolve, status("operational")}},
		{"a shared incident covers an outage", "major_outage", sharedIncident, "major_outage",
			[]string{list, unresolve}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := &fakePage{t: t, components: component(tc.current), incidents: tc.incidents}
			if err := f.client().SyncPage("p", "g", observe(tc.observed)); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(f.calls, tc.want) {
				t.Errorf("calls\n got %q\nwant %q", f.calls, tc.want)
			}
		})
	}
}

func TestSyncPageCreatesMissingComponent(t *testing.T) {
	setupLog(t)
	f := &fakePage{t: t, components: `[]`, incidents: noIncident}
	if err := f.client().SyncPage("p", "g", observe("major_outage")); err != nil {
		t.Fatal(err)
	}
	want := []string{
		"GET /v1/pages/p/components",
		"GET /v1/pages/p/incidents/unresolved",
		"POST /v1/pages/p/components",
		"PATCH /v1/pages/p/components/new degraded_performance",
	}
	if !reflect.DeepEqual(f.calls, want) {
		t.Errorf("calls\n got %q\nwant %q", f.calls, want)
	}
}

func TestSyncPageReportsFailedWrites(t *testing.T) {
	setupLog(t)
	for _, tc := range []struct {
		name      string
		current   string
		incidents string
		observed  string
		fail      string
		// noIncident means the failed status write must stop the incident
		// from being opened against a status the page does not show.
		noIncident bool
	}{
		{name: "listing components", current: "operational", incidents: noIncident, observed: "operational",
			fail: "GET /v1/pages/p/components"},
		{name: "listing incidents", current: "operational", incidents: noIncident, observed: "operational",
			fail: "GET /v1/pages/p/incidents/unresolved"},
		{name: "updating the status", current: "degraded_performance", incidents: noIncident, observed: "major_outage",
			fail: "PATCH /v1/pages/p/components/c", noIncident: true},
		{name: "opening the incident", current: "major_outage", incidents: noIncident, observed: "major_outage",
			fail: "POST /v1/pages/p/incidents"},
		{name: "resolving the incident", current: "operational", incidents: openIncident, observed: "operational",
			fail: "PATCH /v1/pages/p/incidents/i"},
		{name: "creating the component", current: "", incidents: noIncident, observed: "operational",
			fail: "POST /v1/pages/p/components"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			components := component(tc.current)
			if tc.current == "" {
				components = `[]`
			}
			f := &fakePage{t: t, components: components, incidents: tc.incidents, fail: map[string]bool{tc.fail: true}}
			if err := f.client().SyncPage("p", "g", observe(tc.observed)); err == nil {
				t.Errorf("SyncPage returned nil after %s failed; calls %q", tc.fail, f.calls)
			}
			if tc.noIncident {
				for _, c := range f.calls {
					if strings.HasPrefix(c, "POST /v1/pages/p/incidents") {
						t.Errorf("opened an incident after the status write failed: %q", f.calls)
					}
				}
			}
		})
	}
}

func TestSyncPageCarriesOnPastAFailedComponent(t *testing.T) {
	setupLog(t)
	f := &fakePage{t: t, incidents: noIncident,
		components: `[{"id":"a","page_id":"p","name":"a","status":"operational"},` +
			`{"id":"b","page_id":"p","name":"b","status":"degraded_performance"}]`,
		fail: map[string]bool{"PATCH /v1/pages/p/components/a": true}}
	err := f.client().SyncPage("p", "g", []Observation{
		{Component: "a", Status: "major_outage", Incident: "a"},
		{Component: "b", Status: "operational", Incident: "b"},
	})
	if err == nil || !strings.Contains(err.Error(), "a:") {
		t.Errorf("err = %v, want the failure on a", err)
	}
	last := f.calls[len(f.calls)-1]
	if last != "PATCH /v1/pages/p/components/b operational" {
		t.Errorf("b was not written after a failed: %q", f.calls)
	}
}
