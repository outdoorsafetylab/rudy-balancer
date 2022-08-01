package server

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httputil"
	"time"

	"service/log"
	"service/model"
)

type proxyTarget struct {
	Site  *model.Site
	Proxy *httputil.ReverseProxy
}

func (t *proxyTarget) Probe(client *http.Client) error {
	probeURL := t.Site.GetURL(t.Site.Landing)
	res, err := client.Head(probeURL)
	if err != nil {
		return err
	}
	res.Body.Close()
	if res.StatusCode >= 400 {
		return errors.New(res.Status)
	}
	return nil
}

type proxyTargets []*proxyTarget

func (targets proxyTargets) Probe(client *http.Client, timeout time.Duration) *proxyTarget {
	racer := make(chan *proxyTarget, 1)
	for _, t := range targets {
		go func(t *proxyTarget) {
			err := t.Probe(client)
			if err == nil {
				select {
				case racer <- t:
				default:
				}
			}
		}(t)
	}
	select {
	case t := <-racer:
		return t
	case <-time.After(timeout):
		return nil
	}
}

type proxyHandler struct {
	Timeout     time.Duration
	ProbeClient *http.Client
	Targets     proxyTargets
	Redirects   map[string]bool
}

func (h *proxyHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	target := h.Targets.Probe(h.ProbeClient, h.Timeout)
	if target == nil {
		msg := fmt.Sprintf("All mirrors failed to response in %v", h.Timeout)
		log.Errorf(msg)
		http.Error(w, msg, 504)
		return
	}
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
}
