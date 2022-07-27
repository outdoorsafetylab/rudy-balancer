package server

import (
	"math/rand"
	"net/http"
	"net/http/httputil"
	"service/model"
	"time"

	log "github.com/sirupsen/logrus"
)

type proxyTarget struct {
	Site  *model.Site
	Proxy *httputil.ReverseProxy
}

type proxyHandler struct {
	ProbeClient http.Client
	Targets     []*proxyTarget
	Redirects   map[string]bool
}

func (h *proxyHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	rand.Seed(time.Now().UnixNano())
	rand.Shuffle(len(h.Targets), func(i, j int) { h.Targets[i], h.Targets[j] = h.Targets[j], h.Targets[i] })
	for _, target := range h.Targets {
		probeURL := target.Site.GetURL(target.Site.Landing)
		log.Debugf("Probing: %s", probeURL)
		res, err := h.ProbeClient.Head(probeURL)
		if err != nil {
			log.Debugf("Bad luck for %s: %s", target.Site.Name, err.Error())
			continue
		}
		res.Body.Close()
		if res.StatusCode < 400 {
			if h.Redirects[req.RequestURI] {
				url := target.Site.GetURL(req.RequestURI)
				log.Debugf("Redirecting %s for %s", url, req.RequestURI)
				http.Redirect(w, req, url, 302)
				return
			}
			if req.RequestURI == "/" && target.Site.Landing != "" {
				req.URL.Path = target.Site.Landing
			}
			log.Debugf("Serving %s for %s", target.Site.GetURL(req.URL.Path), req.RequestURI)
			req.Host = target.Site.Host // Some sites will reject without it
			target.Proxy.ServeHTTP(w, req)
			return
		}
	}
	msg := "All targets are down"
	log.Errorf(msg)
	http.Error(w, msg, 502)
}
