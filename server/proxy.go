package server

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httputil"
	"path/filepath"
	"time"

	"service/log"
	"service/model"
)

type proxyTarget struct {
	Site  *model.Site
	Proxy *httputil.ReverseProxy
}

func (t *proxyTarget) Probe(client *http.Client) error {
	probeURL := t.Site.GetProxyURL(t.Site.Landing)
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
	Redirects   map[string]string
	Timeout     time.Duration
	ProbeClient *http.Client
	Targets     proxyTargets
	Suffixes    map[string]bool
}

func (h *proxyHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	redirect := h.Redirects[req.URL.Path]
	if redirect != "" {
		log.Debugf("Redirecting %s for %s", redirect, req.RequestURI)
		http.Redirect(w, req, redirect, http.StatusFound)
		return
	}
	target := h.Targets.Probe(h.ProbeClient, h.Timeout)
	if target == nil {
		msg := fmt.Sprintf("All mirrors failed to response in %v", h.Timeout)
		log.Errorf("%s", msg)
		http.Error(w, msg, http.StatusGatewayTimeout)
		return
	}
	if req.URL.Path == "/" {
		if target.Site.Landing != "" {
			req.URL.Path = target.Site.Landing
		}
	} else {
		ext := filepath.Ext(req.URL.Path)
		if !h.Suffixes[ext] {
			url := target.Site.GetRedirectURL(req.URL.Path)
			log.Debugf("Redirecting %s for %s", url, req.RequestURI)
			http.Redirect(w, req, url, http.StatusFound)
			return
		}
	}
	log.Debugf("Serving %s for %s", target.Site.GetProxyURL(req.URL.Path), req.RequestURI)
	req.Host = target.Site.Host // Some sites will reject without it
	target.Proxy.ServeHTTP(w, req)
}
