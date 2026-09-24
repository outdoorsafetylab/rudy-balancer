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

// HasViaLink reports whether rudymap.tw/app builds a via link for an artifact:
// it names a file and has no hard-coded URL. Only controller/app.go uses it;
// every such artifact also has a route, since ViaLinks covers all artifacts
// with a file.
func HasViaLink(a *model.Artifact) bool {
	return a.File != "" && a.URL == ""
}

// ViaLinks lists the (app, file) pairs that get a /via/<app>/<file> route:
// every artifact that names a file, a superset of what HasViaLink generates.
// The extra ones are Locus's: its portal links are hard-coded locus-actions://
// URLs to XML files kept in alpha-rudy/taiwan-topo, which are to be switched
// to /v1/via/locus/<file> once these routes are live. Do not narrow this to
// HasViaLink, or those XML downloads will 404.
func (m *Mirror) ViaLinks() []ViaLink {
	seen := make(map[ViaLink]bool)
	links := make([]ViaLink, 0)
	for _, app := range m.Apps {
		for _, v := range app.Variants {
			for _, a := range v.Artifacts {
				if a.File == "" {
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
