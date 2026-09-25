package model

import (
	"fmt"
	"strings"
)

// Site is one mirror. MonthlyQuota caps the AWS account's CloudFront bytes
// this UTC month (cloudfront.Meter) while the site stays in rotation; 0 means
// no cap.
type Site struct {
	Name           string
	StatusPage     string    `json:"-" firestore:"-"`
	Firestore      string    `json:"-" firestore:"-"`
	MonthlyQuota   int64     `json:"-" firestore:"-"`
	Hidden         bool      `json:",omitempty"`
	RedirectScheme string    `json:"-"`
	Host           string    `json:"-"`
	Prefix         string    `json:"-"`
	ProxyScheme    string    `json:"-"`
	Landing        string    `json:"-"`
	Sources        []*Source `json:"-"`
	Weight         int       `json:"-"`
}

func (s *Site) GetRedirectURL(uri string) string {
	if uri != "" {
		if !strings.HasPrefix(uri, "/") {
			uri = "/" + uri
		}
	}
	return fmt.Sprintf("%s://%s%s%s", s.RedirectScheme, s.Host, s.Prefix, uri)
}

func (s *Site) GetProxyURL(uri string) string {
	if uri != "" {
		if !strings.HasPrefix(uri, "/") {
			uri = "/" + uri
		}
	}
	return fmt.Sprintf("%s://%s%s%s", s.ProxyScheme, s.Host, s.Prefix, uri)
}
