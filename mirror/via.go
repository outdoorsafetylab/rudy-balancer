package mirror

import (
	"fmt"

	"service/model"
)

// ViaLink is one download link rudymap.tw/app generates for an app: the app's
// id and the file it points at (#19).
type ViaLink struct {
	App  string
	File string
}

// ViaLinks lists the (app, file) pairs that get a /via/<app>/<file> route.
// Only artifacts whose link the balancer builds itself (File set, URL empty)
// qualify; an artifact with a hard-coded URL (Locus's locus-actions://) never
// points at a via path, so registering one would only accept strings no page
// ever produces.
// HasViaLink reports whether the balancer builds a via link for an artifact:
// it names a file and has no hard-coded URL. controller/app.go and ViaLinks
// both use it, so a generated link always has a route.
func HasViaLink(a *model.Artifact) bool {
	return a.File != "" && a.URL == ""
}

func (m *Mirror) ViaLinks() []ViaLink {
	seen := make(map[ViaLink]bool)
	links := make([]ViaLink, 0)
	for _, app := range m.Apps {
		for _, v := range app.Variants {
			for _, a := range v.Artifacts {
				if !HasViaLink(a) {
					continue
				}
				link := ViaLink{App: app.ID, File: a.File}
				if !seen[link] {
					seen[link] = true
					links = append(links, link)
				}
			}
		}
	}
	return links
}

// Path is the link's path under the endpoint prefix.
func (l ViaLink) Path() string {
	return fmt.Sprintf("/via/%s/%s", l.App, l.File)
}
