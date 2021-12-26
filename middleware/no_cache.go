package middleware

import (
	"strings"

	"github.com/crosstalkio/rest"
)

var Cacheables = []string{}

func NoCache(handler rest.HandlerFunc) rest.HandlerFunc {
	return func(s *rest.Session) {
		handler(s)
		cacheable := false
		for _, c := range Cacheables {
			if strings.HasPrefix(s.Request.URL.Path, c) {
				cacheable = true
				break
			}
		}
		if !cacheable {
			s.ResponseHeader().Add("Cache-Control", "no-cache")
			s.ResponseHeader().Add("Cache-Control", "no-store")
			s.ResponseHeader().Set("Pragma", "no-cache")
		}
	}
}
