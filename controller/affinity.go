package controller

import (
	"crypto/sha256"
	"encoding/binary"
	"math"

	"service/model"
)

// stickySource picks the mirror for a web page's request (one carrying Origin)
// so that one client lands on the same mirror for every file (#58): a page that
// sends each range of a PMTiles archive through the balancer then reads one
// copy with one ETag, and its style, sprites, glyphs and archive come from the
// same mirror. It is weighted rendezvous hashing over the distinct sites in
// sources, which GetSources repeats by weight: each site scores
// weight / -ln(u), u in (0, 1) from a hash of the client and the site's name,
// and the highest wins. Clients spread over the sites in proportion to their
// weights, and a site dropped for one file (missing, lagging, over quota)
// moves only the clients it had, each to their next site.
func stickySource(sources []*model.Source, client string) *model.Source {
	weights := make(map[string]int)
	sites := make([]*model.Source, 0, len(sources))
	for _, src := range sources {
		if weights[src.SiteName] == 0 {
			sites = append(sites, src)
		}
		weights[src.SiteName]++
	}
	var best *model.Source
	bestScore := math.Inf(-1)
	for _, src := range sites {
		sum := sha256.Sum256([]byte(client + "\x00" + src.SiteName))
		u := (float64(binary.BigEndian.Uint64(sum[:8])>>11) + 0.5) / (1 << 53)
		score := float64(weights[src.SiteName]) / -math.Log(u)
		if score > bestScore {
			best, bestScore = src, score
		}
	}
	return best
}
